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

package hooks

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/deckhouse/deckhouse/testing/hooks"
)

var _ = Describe("Modules :: virtualization :: hooks :: disk_handler ::", func() {
	f := HookExecutionConfigInit(initValuesString, initConfigValuesString)
	f.RegisterCRD("deckhouse.io", "v1alpha1", "VirtualMachineDisk", true)
	f.RegisterCRD("deckhouse.io", "v1alpha1", "VirtualMachineDiskClass", true)
	f.RegisterCRD("deckhouse.io", "v1alpha1", "ClusterVirtualMachineImage", true)
	f.RegisterCRD("cdi.kubevirt.io", "v1beta1", "DataVolume", true)

	// Set Kind for binding.
	diskHandlerHookConfig.Kubernetes[0].Kind = "DataVolume"
	diskHandlerHookConfig.Kubernetes[0].ApiVersion = "cdi.kubevirt.io/v1beta1"

	Context("Empty cluster", func() {
		BeforeEach(func() {
			f.BindingContexts.Set(
				f.KubeStateSet(``),
			)
			f.RunHook()
		})

		It("ExecuteSuccessfully", func() {
			Expect(f).To(ExecuteSuccessfully())
		})
	})

	Context("VirtualMachineDisks creation", func() {
		BeforeEach(func() {
			f.BindingContexts.Set(
				f.KubeStateSet(`
---
apiVersion: deckhouse.io/v1alpha1
kind: ClusterVirtualMachineImage
metadata:
  name: centos-7
spec:
  registry:
    url: "docker://dev-registry.deckhouse.io/sys/deckhouse-oss:8ebc42b654b8e98d9de0f061087cc3b7b3f341ea62374382ece804fb-1658984394800"
---
apiVersion: deckhouse.io/v1alpha1
kind: VirtualMachineDiskClass
metadata:
  name: linstor-slow
spec:
  accessModes: [ "ReadWriteMany" ]
  volumeMode: Block
  storageClassName: linstor-data-r2
---
apiVersion: deckhouse.io/v1alpha1
kind: VirtualMachineDisk
metadata:
  name: mydata
  namespace: ns1
spec:
  source:
    name: centos-7
    scope: public
  type: linstor-slow
  size: 10Gi
`),
			)
			f.RunHook()
		})

		It("Creates DataVolume out of VirtualMachineDisk", func() {
			Expect(f).To(ExecuteSuccessfully())
			By("Checking existing VM, IPAddressClaim is not static, should be kept")
			dataVolume := f.KubernetesResource("DataVolume", "ns1", "mydata")
			Expect(dataVolume).To(Not(BeEmpty()))
			Expect(dataVolume.Field(`spec.source.registry.url`).String()).To(Equal("docker://dev-registry.deckhouse.io/sys/deckhouse-oss:8ebc42b654b8e98d9de0f061087cc3b7b3f341ea62374382ece804fb-1658984394800"))
			Expect(dataVolume.Field(`spec.pvc.resources.requests.storage`).String()).To(Equal("10Gi"))
			Expect(dataVolume.Field(`spec.pvc.storageClassName`).String()).To(Equal("linstor-data-r2"))
			Expect(dataVolume.Field(`spec.pvc.volumeMode`).String()).To(Equal("Block"))
			// TODO: dataVolume.Field(`spec.pvc.accessModes`)
		})
	})

})
