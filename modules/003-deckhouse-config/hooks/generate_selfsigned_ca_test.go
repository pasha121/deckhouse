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
	"crypto/x509"
	"encoding/pem"
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/deckhouse/deckhouse/testing/hooks"
)

const (
	initValuesString = `{
  "deckhouseConfig": {
    "internal": {
      "webhookCert":{}
    }
  }
}`
	initConfigValuesString = `{}`
)

const (
	stateSecretCreated = `
apiVersion: v1
kind: Secret
type: kubernetes.io/tls
metadata:
  name: deckhouse-config-webhook-tls
  namespace: d8-system
data:
  tls.crt: YQ== # a
  tls.key: Yg== # b
  ca.crt:  Yw== # c
`
)

var _ = Describe("DeckhouseConfig hooks :: generate self-signed CA :: ", func() {
	f := HookExecutionConfigInit(initValuesString, initConfigValuesString)

	Context("giving no Secret", func() {
		BeforeEach(func() {
			f.KubeStateSet(``)
			f.BindingContexts.Set(f.GenerateBeforeHelmContext())
			f.RunHook()
		})

		It("Should generate new certificates and set values", func() {
			Expect(f).To(ExecuteSuccessfully())
			Expect(f.ValuesGet(webhookHandlerCertPath).Exists()).To(BeTrue())
			Expect(f.ValuesGet(webhookHandlerKeyPath).Exists()).To(BeTrue())
			Expect(f.ValuesGet(webhookHandlerCAPath).Exists()).To(BeTrue())

			certPool := x509.NewCertPool()
			ok := certPool.AppendCertsFromPEM([]byte(f.ValuesGet(webhookHandlerCAPath).String()))
			Expect(ok).To(BeTrue())

			block, _ := pem.Decode([]byte(f.ValuesGet(webhookHandlerCertPath).String()))
			Expect(block).ShouldNot(BeNil())

			cert, err := x509.ParseCertificate(block.Bytes)
			Expect(err).ShouldNot(HaveOccurred())

			opts := x509.VerifyOptions{
				DNSName: fmt.Sprintf("%s.%s.svc", webhookServiceHost, webhookServiceNamespace),
				Roots:   certPool,
			}
			_, err = cert.Verify(opts)
			Expect(err).ShouldNot(HaveOccurred())
		})
	})

	Context("giving existing Secret", func() {
		BeforeEach(func() {
			f.KubeStateSet(stateSecretCreated)
			f.BindingContexts.Set(f.GenerateBeforeHelmContext())
			f.RunHook()
		})

		It("should restore certificates from secret", func() {
			Expect(f).To(ExecuteSuccessfully())
			Expect(f.ValuesGet(webhookHandlerCertPath).String()).To(Equal("a"))
			Expect(f.ValuesGet(webhookHandlerKeyPath).String()).To(Equal("b"))
			Expect(f.ValuesGet(webhookHandlerCAPath).String()).To(Equal("c"))
		})
	})
})
