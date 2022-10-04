package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"text/tabwriter"

	"github.com/sirupsen/logrus"
	kwhhttp "github.com/slok/kubewebhook/v2/pkg/http"
	kwhlog "github.com/slok/kubewebhook/v2/pkg/log"
	kwhlogrus "github.com/slok/kubewebhook/v2/pkg/log/logrus"
	kwhmodel "github.com/slok/kubewebhook/v2/pkg/model"
	kwhvalidating "github.com/slok/kubewebhook/v2/pkg/webhook/validating"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"

	"vmi-ipam-webhook/api/v1alpha1"
	d8v1alpha1 "vmi-ipam-webhook/api/v1alpha1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type config struct {
	cidrs      cidrFlag
	certFile   string
	keyFile    string
	kubeconfig string
}

type ipAddressLeaseValidator struct {
	logger kwhlog.Logger
}

type cidrFlag []string

func (f *cidrFlag) String() string { return "" }
func (f *cidrFlag) Set(s string) error {
	*f = append(*f, s)
	return nil
}

func (v *ipAddressLeaseValidator) Validate(_ context.Context, ar *kwhmodel.AdmissionReview, obj metav1.Object) (*kwhvalidating.ValidatorResult, error) {
	ipAddressLease, ok := obj.(*d8v1alpha1.IPAddressLease)
	if !ok {
		// If not a pod just continue the mutation chain(if there is one) and don't do nothing.
		return &kwhvalidating.ValidatorResult{}, nil
	}

	if ar.Operation == kwhmodel.OperationCreate {
		if ipAddressLease.Spec.Address != "" {
			if !isIPAddressInRange(ipAddressLease.Spec.Address) {
				return &kwhvalidating.ValidatorResult{
					Valid:   false,
					Message: "requested IP address is not in range",
				}, nil
			}
			if isIPAddressAllocated(ipAddressLease.Spec.Address) {
				return &kwhvalidating.ValidatorResult{
					Valid:   false,
					Message: "requested IP address is already allocated",
				}, nil
			}
		}

		return &kwhvalidating.ValidatorResult{
			Valid:   true,
			Message: "ip address is valid",
		}, nil
	}

	if ar.Operation == kwhmodel.OperationUpdate {
		// Mutate our object with the required annotations.
		if ipAddressLease.Spec.Address == "" {
			return &kwhvalidating.ValidatorResult{}, fmt.Errorf("Not allowed to change spec.address")
		}

		oldIPAddressLease := d8v1alpha1.IPAddressLease{}
		if err := json.Unmarshal(ar.OldObjectRaw, &oldIPAddressLease); err != nil {
			return &kwhvalidating.ValidatorResult{
				Valid:   false,
				Message: "cannot unmarshal old IPAddressLease object: %v\n",
			}, nil
		}

		if oldIPAddressLease.Spec.Address != "" {
			newIPAddressLease := d8v1alpha1.IPAddressLease{}
			if err := json.Unmarshal(ar.OldObjectRaw, &newIPAddressLease); err != nil {
				return &kwhvalidating.ValidatorResult{
					Valid:   false,
					Message: "cannot unmarshal new IPAddressLease object: %v\n",
				}, nil
			}
			if oldIPAddressLease.Spec.Address != newIPAddressLease.Spec.Address {
				return &kwhvalidating.ValidatorResult{
					Valid:   false,
					Message: "Field spec.address is immutable after first assignment: %v\n",
				}, nil
			}
		}
	}
	return &kwhvalidating.ValidatorResult{}, fmt.Errorf("Unknown operation")

}

func isIPAddressAllocated(string) bool { return false } // TODO
func isIPAddressInRange(string) bool   { return true }  // TODO

func main() {
	var cidrs cidrFlag
	allocatedIPAddresses := make(map[string]struct{})
	logrusLogEntry := logrus.NewEntry(logrus.New())
	logrusLogEntry.Logger.SetLevel(logrus.DebugLevel)
	logger := kwhlogrus.NewLogrus(logrusLogEntry)
	cfg := &config{}

	flag.Var(&cfg.cidrs, "cidr", "CIDRs enabled to route (multiple flags allowed)")
	flag.StringVar(&cfg.certFile, "tls-cert-file", "", "TLS certificate file")
	flag.StringVar(&cfg.keyFile, "tls-key-file", "", "TLS key file")
	if kubeconfig, exist := os.LookupEnv("KUBECONFIG"); exist {
		cfg.kubeconfig = kubeconfig
	} else if home := homedir.HomeDir(); home != "" {
		cfg.kubeconfig = *flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
	} else {
		cfg.kubeconfig = *flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	}
	flag.Parse()

	var parsedCIDRs []*net.IPNet
	for _, cidr := range cidrs {
		_, parsedCIDR, err := net.ParseCIDR(cidr)
		if err != nil || parsedCIDR == nil {
			fmt.Println(err, "failed to parse CIDR")
			os.Exit(1)
		}
		parsedCIDRs = append(parsedCIDRs, parsedCIDR)
	}

	logger.Infof(fmt.Sprintf("managed CIDRs: %+v", cidrs))

	// kubernetes config loaded from ./config or whatever the flag was set to
	config, err := clientcmd.BuildConfigFromFlags("", cfg.kubeconfig)
	if err != nil {
		logger.Errorf("cannot load Kubernetes config: %v\n", err)
		os.Exit(1)
	}

	// instantiate our client with config
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		logger.Errorf("cannot create Kunernetes client: %v\n", err)
		os.Exit(1)
	}

	// Fetch list of IPAddressLeases
	ipList := d8v1alpha1.IPAddressLeaseList{}
	ipListRaw, err := clientset.RESTClient().Get().AbsPath("/apis/deckhouse.io/v1alpha1/ipaddressleases").DoRaw(context.TODO())
	if err != nil {
		logger.Errorf("cannot obtain IPAddressLeases list: %v\n", err)
		os.Exit(1)
	}

	if err := json.Unmarshal(ipListRaw, &ipList); err != nil {
		logger.Errorf("cannot unmarshal IPAddressLeases list: %v\n", err)
		os.Exit(1)
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 5, ' ', 0)
	fmt.Fprintln(w, "Type\tName\tNamespace\tStatus\tip\tmac")

	for _, lease := range ipList.Items {
		ip := lease.Spec.Address
		allocatedIPAddresses[ip] = struct{}{}
		fmt.Fprintf(w, "%s\t%s\t%s\t%v\n", lease.Kind, lease.Name, lease.Namespace, ip)
	}
	w.Flush()

	////////////////////////////

	// Create our mutator
	vl := &ipAddressLeaseValidator{
		logger: logger,
	}

	mcfg := kwhvalidating.WebhookConfig{
		ID:        "ipAddressLeaseValidator",
		Obj:       &v1alpha1.IPAddressLease{},
		Validator: vl,
		Logger:    logger,
	}
	wh, err := kwhvalidating.NewWebhook(mcfg)
	if err != nil {
		logger.Errorf("error creating webhook: %s", err)
		os.Exit(1)
	}

	// Get the handler for our webhook.
	whHandler, err := kwhhttp.HandlerFor(kwhhttp.HandlerConfig{Webhook: wh, Logger: logger})
	if err != nil {
		logger.Errorf("error creating webhook handler: %s", err)
		os.Exit(1)
	}
	logger.Infof("Listening on :8080")
	err = http.ListenAndServeTLS(":8080", cfg.certFile, cfg.keyFile, whHandler)
	if err != nil {
		logger.Errorf("error serving webhook: %s", err)
		os.Exit(1)
	}

}
