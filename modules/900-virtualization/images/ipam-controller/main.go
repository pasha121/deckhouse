package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"sync"
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

	d8v1alpha1 "vmi-ipam-controller/api/v1alpha1"
	"vmi-ipam-controller/controllers"
	"vmi-ipam-controller/webhooks"
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
	var pendingLeases sync.Map

	// pass webhook parameters
	webhooks.Logger = logger
	webhooks.CertFile = cfg.certFile
	webhooks.KeyFile = cfg.keyFile
	webhooks.IPAM = ipam
	webhooks.PendingLeases = pendingLeases

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var prefixes []*goipam.Prefix
	for _, cidr := range cfg.cidrs {
		prefix, err := ipam.NewPrefix(ctx, cidr)
		if err != nil {
			logger.Errorf("error creating new prefix for IPAM: %s", err)
			os.Exit(1)
		}
		prefixes = append(prefixes, prefix)
	}

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
		NodeName:      os.Getenv("NODE_NAME"),
		RESTClient:    restClient,
		Ipam:          ipam,
		Log:           logger,
		Prefixes:      prefixes,
		PendingLeases: pendingLeases,
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

}
