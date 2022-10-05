package main

import (
	"context"
	"flag"
	"fmt"
	"net"
	"os"
	"time"

	goipam "github.com/metal-stack/go-ipam"
	"github.com/sirupsen/logrus"
	kwhlogrus "github.com/slok/kubewebhook/v2/pkg/log/logrus"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
	"sigs.k8s.io/controller-runtime/pkg/healthz"

	d8v1alpha1 "vmi-ipam-webhook/api/v1alpha1"
	"vmi-ipam-webhook/controllers"
)

type config struct {
	cidrs       cidrFlag
	certFile    string
	keyFile     string
	metricsAddr string
	probeAddr   string
}

var scheme = runtime.NewScheme()

type cidrFlag []string

func (f *cidrFlag) String() string { return "" }
func (f *cidrFlag) Set(s string) error {
	*f = append(*f, s)
	return nil
}

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(d8v1alpha1.AddToScheme(scheme))
}

func main() {
	// allocatedIPAddresses := make(map[string]struct{})
	logrusLogEntry := logrus.NewEntry(logrus.New())
	logrusLogEntry.Logger.SetLevel(logrus.DebugLevel)
	logger := kwhlogrus.NewLogrus(logrusLogEntry)
	cfg := &config{}

	flag.Var(&cfg.cidrs, "cidr", "CIDRs enabled to route (multiple flags allowed)")
	flag.StringVar(&cfg.certFile, "tls-cert-file", "", "TLS certificate file")
	flag.StringVar(&cfg.keyFile, "tls-key-file", "", "TLS key file")
	flag.StringVar(&cfg.metricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	flag.StringVar(&cfg.probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")

	flag.Parse()

	logger.Infof(fmt.Sprintf("managed CIDRs: %+v", cfg.cidrs))

	// create a ipamer with in memory storage
	ipam := goipam.New()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var parsedCIDRs []*net.IPNet
	for _, cidr := range cfg.cidrs {
		_, parsedCIDR, err := net.ParseCIDR(cidr)
		if err != nil {
			logger.Errorf("failed to add CIDR: %s", err)
			os.Exit(1)
		}
		parsedCIDRs = append(parsedCIDRs, parsedCIDR)
		_, err = ipam.NewPrefix(ctx, cidr)
		if err != nil {
			logger.Errorf("error creating new prefix for IPAM: %s", err)
			os.Exit(1)
		}
	}

	// // kubernetes config loaded from ./config or whatever the flag was set to
	// config, err := clientcmd.BuildConfigFromFlags("", cfg.kubeconfig)
	// if err != nil {
	// 	logger.Errorf("cannot load Kubernetes config: %v\n", err)
	// 	os.Exit(1)
	// }

	// // instantiate our client with config
	// clientset, err := kubernetes.NewForConfig(config)
	// if err != nil {
	// 	logger.Errorf("cannot create Kunernetes client: %v\n", err)
	// 	os.Exit(1)
	// }

	// // Fetch list of IPAddressLeases
	// ipList := d8v1alpha1.IPAddressLeaseList{}
	// ipListRaw, err := clientset.RESTClient().Get().AbsPath("/apis/deckhouse.io/v1alpha1/ipaddressleases").DoRaw(context.TODO())
	// if err != nil {
	// 	logger.Errorf("cannot obtain IPAddressLeases list: %v\n", err)
	// 	os.Exit(1)
	// }

	// if err := json.Unmarshal(ipListRaw, &ipList); err != nil {
	// 	logger.Errorf("cannot unmarshal IPAddressLeases list: %v\n", err)
	// 	os.Exit(1)
	// }

	// w := tabwriter.NewWriter(os.Stdout, 0, 0, 5, ' ', 0)
	// fmt.Fprintln(w, "Type\tName\tNamespace\tStatus\tip\tmac")

	// for _, lease := range ipList.Items {
	// 	ip := lease.Spec.Address
	// 	allocatedIPAddresses[ip] = struct{}{}
	// 	fmt.Fprintf(w, "%s\t%s\t%s\t%v\n", lease.Kind, lease.Name, lease.Namespace, ip)
	// }
	// w.Flush()

	//////////////////////////////

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:                 scheme,
		MetricsBindAddress:     cfg.metricsAddr,
		Port:                   9443,
		HealthProbeBindAddress: cfg.probeAddr,
	})
	if err != nil {
		logger.Errorf("unable to start manager: %s", err)
		os.Exit(1)
	}

	restClient, err := apiutil.RESTClientForGVK(d8v1alpha1.GroupVersion.WithKind("IPAddressLease"), false, mgr.GetConfig(), serializer.NewCodecFactory(mgr.GetScheme()))
	if err != nil {
		logger.Errorf("unable to create REST client: %s", err)
		os.Exit(1)
	}

	controller := controllers.IPAddressLeaseController{
		NodeName:   os.Getenv("NODE_NAME"),
		RESTClient: restClient,
		Ipam:       ipam,
		Log:        logger,
		Cidrs:      parsedCIDRs,
	}

	if err := mgr.Add(controller); err != nil {
		logger.Errorf("unable to add ipaddressleases controller to manager %s", err)
		os.Exit(1)
	}

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		logger.Errorf("unable to set up health check: %s", err)
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		logger.Errorf("unable to set up ready check: %s", err)
		os.Exit(1)
	}

	logger.Infof("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		logger.Errorf("problem running manager: %s", err)
		os.Exit(1)
	}

	////////////////////////////

}
