package webhooks

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	d8v1alpha1 "vmi-ipam-webhook/api/v1alpha1"

	kwhhttp "github.com/slok/kubewebhook/v2/pkg/http"
	kwhlog "github.com/slok/kubewebhook/v2/pkg/log"
	kwhmodel "github.com/slok/kubewebhook/v2/pkg/model"
	kwhvalidating "github.com/slok/kubewebhook/v2/pkg/webhook/validating"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	certFile string
	keyFile  string
	logger   kwhlog.Logger
)

func Start() {
	// Create our validator
	vl := &ipAddressLeaseValidator{
		logger: logger,
	}

	mcfg := kwhvalidating.WebhookConfig{
		ID:        "ipAddressLeaseValidator",
		Obj:       &d8v1alpha1.IPAddressLease{},
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
	logger.Infof("Listening on :8082")
	err = http.ListenAndServeTLS(":8082", certFile, keyFile, whHandler)
	if err != nil {
		logger.Errorf("error serving webhook: %s", err)
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
	return &kwhvalidating.ValidatorResult{}, fmt.Errorf("Unknown operation")

}

func isIPAddressAllocated(string) bool { return false } // TODO
func isIPAddressInRange(string) bool   { return true }  // TODO
