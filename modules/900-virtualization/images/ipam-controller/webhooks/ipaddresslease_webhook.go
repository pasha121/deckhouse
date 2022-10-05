package webhooks

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"

	d8v1alpha1 "vmi-ipam-controller/api/v1alpha1"

	goipam "github.com/metal-stack/go-ipam"
	kwhhttp "github.com/slok/kubewebhook/v2/pkg/http"
	kwhlog "github.com/slok/kubewebhook/v2/pkg/log"
	kwhmodel "github.com/slok/kubewebhook/v2/pkg/model"
	kwhvalidating "github.com/slok/kubewebhook/v2/pkg/webhook/validating"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	CertFile string
	KeyFile  string
	Logger   kwhlog.Logger
	IPAM     goipam.Ipamer
	Prefixes []goipam.Prefix
)

func Start() {
	// Create our validator
	vl := &ipAddressLeaseValidator{
		logger: Logger,
	}

	mcfg := kwhvalidating.WebhookConfig{
		ID:        "ipAddressLeaseValidator",
		Obj:       &d8v1alpha1.IPAddressLease{},
		Validator: vl,
		Logger:    Logger,
	}
	wh, err := kwhvalidating.NewWebhook(mcfg)
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
	return &kwhvalidating.ValidatorResult{Valid: true}, nil
}

func isIPAddressAllocated(address string) bool {
	prefix := prefixForIP(address)
	ip, err := IPAM.AcquireSpecificIP(context.TODO(), prefix, address)
	if err != nil {
		return true
	}
	IPAM.ReleaseIPFromPrefix(context.TODO(), prefix, ip.IP.String())
	return false
}

func isIPAddressInRange(ip string) bool {
	return prefixForIP(ip) != ""
}

func prefixForIP(ip string) string {
	if ip == "" {
		return availablePrefix()
	}
	for _, prefix := range Prefixes {
		_, cidr, err := net.ParseCIDR(prefix.Cidr)
		if err != nil {
			Logger.Errorf("failed to parse CIDR: %s", err)
			os.Exit(1)
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
