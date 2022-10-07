package webhooks

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"sync"
	"time"

	d8v1alpha1 "github.com/deckhouse/deckhouse/modules/900-virtualization/api/v1alpha1"

	goipam "github.com/metal-stack/go-ipam"
	kwhhttp "github.com/slok/kubewebhook/v2/pkg/http"
	kwhlog "github.com/slok/kubewebhook/v2/pkg/log"
	"github.com/slok/kubewebhook/v2/pkg/model"
	kwhmodel "github.com/slok/kubewebhook/v2/pkg/model"
	kwhmutating "github.com/slok/kubewebhook/v2/pkg/webhook/mutating"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	CertFile      string
	KeyFile       string
	Logger        kwhlog.Logger
	IPAM          goipam.Ipamer
	Prefixes      []*goipam.Prefix
	PendingLeases *sync.Map
)

type Lease struct {
	IP   *goipam.IP
	Time time.Time
}

func Start() {
	// Create our validator
	mt := kwhmutating.MutatorFunc(ipAddressLeaseMutator)

	mcfg := kwhmutating.WebhookConfig{
		ID:      "ipAddressLeaseMutator",
		Obj:     &d8v1alpha1.IPAddressLease{},
		Mutator: mt,
		Logger:  Logger,
	}
	wh, err := kwhmutating.NewWebhook(mcfg)
	if err != nil {
		Logger.Errorf("error creating webhook: %s", err)
		os.Exit(1)
	}

	// Get the handler for our webhook.
	whHandler, err := kwhhttp.HandlerFor(kwhhttp.HandlerConfig{Webhook: wh, Logger: Logger})
	if err != nil {
		Logger.Errorf("error creating webhook handler: %s", err)
		os.Exit(1)
	}
	Logger.Infof("Listening on :8082")
	err = http.ListenAndServeTLS(":8082", CertFile, KeyFile, whHandler)
	if err != nil {
		Logger.Errorf("error serving webhook: %s", err)
		os.Exit(1)
	}

}

type ipAddressLeaseValidator struct {
	logger kwhlog.Logger
}

func ipAddressLeaseMutator(_ context.Context, ar *kwhmodel.AdmissionReview, obj metav1.Object) (*kwhmutating.MutatorResult, error) {
	lease, ok := obj.(*d8v1alpha1.IPAddressLease)
	if !ok {
		// If not a IPAddressLease just continue the mutation chain(if there is one) and don't do nothing.
		return &kwhmutating.MutatorResult{}, nil
	}

	if lease.Spec.Address != "" && net.ParseIP(lease.Spec.Address) == nil {
		return &kwhmutating.MutatorResult{}, fmt.Errorf("specified ip address is not valid")
	}

	if ar.Operation == model.OperationUpdate {
		var oldLease d8v1alpha1.IPAddressLease
		if err := json.Unmarshal(ar.OldObjectRaw, &oldLease); err != nil {
			return &kwhmutating.MutatorResult{}, fmt.Errorf("unable to unmarshal old lease object: %s", err)
		}
		if oldLease.Spec.Address != "" && oldLease.Spec.Address != lease.Spec.Address {
			return &kwhmutating.MutatorResult{}, fmt.Errorf("field address is immutable after the first assignment")
		} else {
			// Allow other updates
			return &kwhmutating.MutatorResult{}, nil
		}
	}

	prefix := prefixForIP(lease.Spec.Address)
	if prefix == "" {
		return &kwhmutating.MutatorResult{}, fmt.Errorf("unable to find suitable CIDR for allocation, available ranges: %+v", Prefixes)
	}

	ip, err := IPAM.AcquireSpecificIP(context.TODO(), prefix, lease.Spec.Address)
	if err != nil {
		return &kwhmutating.MutatorResult{}, fmt.Errorf("unable to allocate ip address: %s", err)
	}

	PendingLeases.Store(ip.IP.String(), Lease{IP: ip, Time: time.Now()})

	if lease.Spec.Address == "" {
		// Assign IP-address
		lease.Spec.Address = ip.IP.String()
	}

	return &kwhmutating.MutatorResult{
		MutatedObject: lease,
	}, nil

}

func prefixForIP(ip string) string {
	if ip == "" {
		return availablePrefix()
	}
	for _, prefix := range Prefixes {
		_, cidr, err := net.ParseCIDR(prefix.Cidr)
		if err != nil {
			return ""
		}

		if cidr.Contains(net.ParseIP(ip)) {
			return cidr.String()
		}
	}
	return ""
}

func availablePrefix() string {
	for _, prefix := range Prefixes {
		if prefix.Usage().AvailableIPs != 0 {
			return prefix.Cidr
		}
	}
	return ""
}
