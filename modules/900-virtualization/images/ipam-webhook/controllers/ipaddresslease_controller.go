/*
Copyright 2022 Flant JSC

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"
	"fmt"
	"net"
	"os"
	"sync"
	"time"

	d8v1alpha1 "vmi-ipam-webhook/api/v1alpha1"
	"vmi-ipam-webhook/webhooks"

	goipam "github.com/metal-stack/go-ipam"
	kwhlog "github.com/slok/kubewebhook/v2/pkg/log"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/fields"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
)

type IPAddressLeaseController struct {
	RESTClient    rest.Interface
	Ipam          goipam.Ipamer
	Log           kwhlog.Logger
	Prefixes      []*goipam.Prefix
	PendingLeases *sync.Map
}

func (c IPAddressLeaseController) Start(ctx context.Context) error {
	c.Log.Infof("starting ipaddressleases controller")

	lw := cache.NewListWatchFromClient(c.RESTClient, "ipaddressleases", v1.NamespaceAll, fields.Everything())
	informer := cache.NewSharedIndexInformer(lw, &d8v1alpha1.IPAddressLease{}, 12*time.Hour,
		cache.Indexers{
			"namespace_name": func(obj interface{}) ([]string, error) {
				return []string{obj.(*d8v1alpha1.IPAddressLease).GetName()}, nil
			},
		},
	)
	informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    c.addFunc,
		DeleteFunc: c.deleteFunc,
	})

	stopper := make(chan struct{})
	defer close(stopper)
	defer utilruntime.HandleCrash()
	go informer.Run(stopper)
	c.Log.Infof("syncronizing")

	//syncronize the cache before starting to process
	if !cache.WaitForCacheSync(stopper, informer.HasSynced) {
		utilruntime.HandleError(fmt.Errorf("Timed out waiting for caches to sync"))
		c.Log.Errorf("syncronization failed")
		return fmt.Errorf("syncronization failed")
	}
	c.Log.Infof("syncronization completed")

	c.Log.Infof("starting validation webhook")
	go webhooks.Start()

	<-ctx.Done()
	c.Log.Infof("shutting down ipaddressleases controller")

	return nil
}

func (c IPAddressLeaseController) addFunc(obj interface{}) {
	lease, ok := obj.(*d8v1alpha1.IPAddressLease)
	var err error
	if !ok {
		// object is not IPAddressLease
		return
	}
	if lease.Spec.Address == "" {
		c.Log.Errorf("missing address for %s/%s", lease.Namespace, lease.Name)
		return
	}

	if _, ok := c.PendingLeases.LoadAndDelete(lease.Spec.Address); ok {
		c.Log.Infof("allocated %s/%s: %s", lease.Namespace, lease.Name, lease.Spec.Address)
		return
	}

	prefix := c.prefixForIP(lease.Spec.Address)
	if prefix == "" {
		c.Log.Errorf("unable to find prefix for IP %s", lease.Spec.Address)
		return
	}
	_, err = c.Ipam.AcquireSpecificIP(context.TODO(), prefix, lease.Spec.Address)
	if err != nil {
		c.Log.Errorf("error allocating ip %s: %+s", lease.Spec.Address, err)
		return
	}
	c.Log.Infof("loaded %s/%s: %s", lease.Namespace, lease.Name, lease.Spec.Address)
}
func (c IPAddressLeaseController) deleteFunc(obj interface{}) {
	lease, ok := obj.(*d8v1alpha1.IPAddressLease)
	if !ok {
		// object is not IPAddressLease
		return
	}
	if lease.Spec.Address != "" {
		err := c.Ipam.ReleaseIPFromPrefix(context.TODO(), c.prefixForIP(lease.Spec.Address), lease.Spec.Address)
		if err != nil {
			c.Log.Errorf("error releasing ip %s: %+s", lease.Spec.Address, err)
		}
	}
	c.Log.Infof("released %s/%s: %s", lease.Namespace, lease.Name, lease.Spec.Address)
}

func (c IPAddressLeaseController) prefixForIP(ip string) string {
	if ip == "" {
		return c.availablePrefix()
	}
	for _, prefix := range c.Prefixes {
		_, cidr, err := net.ParseCIDR(prefix.Cidr)
		if err != nil {
			c.Log.Errorf("failed to parse CIDR: %s", err)
			os.Exit(1)
		}

		if cidr.Contains(net.ParseIP(ip)) {
			return cidr.String()
		}
	}
	return ""
}

func (c IPAddressLeaseController) availablePrefix() string {
	for _, prefix := range c.Prefixes {
		if prefix.Usage().AvailableIPs != 0 {
			return prefix.Cidr
		}
	}
	c.Log.Errorf("no available ips found")
	return ""
}
