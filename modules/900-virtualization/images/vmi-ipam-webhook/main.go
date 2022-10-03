package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"text/tabwriter"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	virtv1 "kubevirt.io/api/core/v1"

	api "vmi-ipam-webhook/api/v1alpha1"
)

func main() {

	var kubeconfig *string
	if config, exist := os.LookupEnv("KUBECONFIG"); exist {
		kubeconfig = &config
	} else if home := homedir.HomeDir(); home != "" {
		kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
	} else {
		kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	}
	flag.Parse()

	// kubernetes config loaded from ./config or whatever the flag was set to
	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		panic(err)
	}

	// instantiate our client with config
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err)
	}

	// Fetch list of VMIs
	vmiList := virtv1.VirtualMachineInstanceList{}
	vmiListRaw, err := clientset.RESTClient().Get().AbsPath("/apis/kubevirt.io/v1/virtualmachineinstances").DoRaw(context.TODO())
	//vmiList, err := virtClient.VirtualMachineInstance(namespace).List(&k8smetav1.ListOptions{})
	if err != nil {
		log.Fatalf("cannot obtain KubeVirt vmi list: %v\n", err)
	}

	if err := json.Unmarshal(vmiListRaw, &vmiList); err != nil {
		log.Fatalf("cannot unmarshal KubeVirt vmi list: %v\n", err)
	}

	// Fetch list of IPAddressLeases
	ipList := api.IPAddressLeaseList{}
	ipListRaw, err := clientset.RESTClient().Get().AbsPath("/apis/deckhouse.io/v1alpha1/ipaddressleases").DoRaw(context.TODO())
	if err != nil {
		log.Fatalf("cannot obtain IPAddressLeases list: %v\n", err)
	}

	if err := json.Unmarshal(ipListRaw, &ipList); err != nil {
		log.Fatalf("cannot unmarshal IPAddressLeases list: %v\n", err)
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 5, ' ', 0)
	fmt.Fprintln(w, "Type\tName\tNamespace\tStatus\tip\tmac")

	for _, vmi := range vmiList.Items {
		annotations := vmi.GetAnnotations()
		ip := annotations["cni.cilium.io/ipAddrs"]
		mac := vmi.Annotations["cni.cilium.io/macAddrs"]
		fmt.Fprintf(w, "%s\t%s\t%s\t%v\t%v\t%v\n", vmi.Kind, vmi.Name, vmi.Namespace, vmi.Status.Phase, ip, mac)
	}

	for _, lease := range ipList.Items {
		ip := lease.Spec.Address
		static := lease.Spec.Static
		fmt.Fprintf(w, "%s\t%s\t%s\t%v\t%v\n", lease.Kind, lease.Name, lease.Namespace, ip, static)
	}
	w.Flush()
}

// Create VM.deckhouse --> Creates IPAddressLease.deckhouse (only if IP specified)
//  									 --> Creates VM.kubevirt --> Creates VMI.kubevirt
//
// Create VMI.kubevirt --> Checks annotations -->
// 																								If IP assigned: check if lease is not used and belongs to the same namespace
// 																								If IP is not assigned: check if it is free and in vmCIDR range
// 																										If it is used: reject creation
// 																										If it is not used: Create IPAddressLease object with ownerReference + add to map
// Create IPAddressLease --> Check if it is free and in vmCIDR range
// 															If it is used: reject creation
// 															If it is not used: pass + add to map
// Remove IPAddressLease --> If not vmi exists. Release the IP
//                           If vmi exists. Restrict deletion

// List all reserved IPs:
// 		Load VMIs and read annotations
// 		Load IPAddressLeases
