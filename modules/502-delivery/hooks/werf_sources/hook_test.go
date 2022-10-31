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
	"context"

	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	yamlSrlzr "k8s.io/apimachinery/pkg/runtime/serializer/yaml"
	"sigs.k8s.io/yaml"
	// . "github.com/deckhouse/deckhouse/testing/hooks"
)

var _ = Describe("Modules :: delivery :: hooks :: werf_sources ::", func() {

	decUnstructured := yamlSrlzr.NewDecodingSerializer(unstructured.UnstructuredJSONScheme)
	Context("parsing of WerfSources resource into inner formet", func() {
		table.DescribeTable("Parsing werf_sources", func(wsyaml string, expected werfSource) {
			// Setup
			obj := &unstructured.Unstructured{}
			_, _, err := decUnstructured.Decode([]byte(wsyaml), nil, obj)
			Expect(err).ToNot(HaveOccurred())

			// Action
			ws, err := filterWerfSource(obj)

			// Assert
			Expect(err).ToNot(HaveOccurred())
			Expect(ws).To(Equal(expected))
		},
			table.Entry("Minimal: only image repo", `
apiVersion: deckhouse.io/v1alpha1
kind: WerfSource
metadata:
  name: minimal
spec:
  imageRepo: cr.example.com/the/path
`,
				werfSource{
					name:   "minimal",
					repo:   "cr.example.com/the/path",
					apiUrl: "https://cr.example.com",
					argocdRepo: &argocdRepoConfig{
						project: "default",
					},
				}),

			table.Entry("Full", `
apiVersion: deckhouse.io/v1alpha1
kind: WerfSource
metadata:
  name: full-object
spec:
  imageRepo: cr.example.com/the/path
  apiUrl: https://different.example.com
  pullSecretName: registry-credentials
  argocdRepoEnabled: true
  argocdRepo:
    project: ecommerce

`,
				werfSource{
					name:   "full-object",
					repo:   "cr.example.com/the/path",
					apiUrl: "https://different.example.com",

					pullSecretName: "registry-credentials",
					argocdRepo: &argocdRepoConfig{
						project: "ecommerce",
					},
				}),

			table.Entry("argocdRepoEnabled=false omits the repo config for Argo", `
apiVersion: deckhouse.io/v1alpha1
kind: WerfSource
metadata:
  name: repo-off
spec:
  imageRepo: cr.example.com/the/path
  argocdRepoEnabled: false
`,
				werfSource{
					name:   "repo-off",
					repo:   "cr.example.com/the/path",
					apiUrl: "https://cr.example.com",
				}),

			table.Entry("argocdRepoEnabled=false omits the repo config for Argo even when repo options are specified ", `
apiVersion: deckhouse.io/v1alpha1
kind: WerfSource
metadata:
  name: repo-off-yet-specified
spec:
  imageRepo: cr.example.com/the/path
  argocdRepoEnabled: false
  argocdRepo:
    project: actually-skipped
`,
				werfSource{
					name:   "repo-off-yet-specified",
					repo:   "cr.example.com/the/path",
					apiUrl: "https://cr.example.com",
				}),

			table.Entry("Argo CD non-defaul project", `
apiVersion: deckhouse.io/v1alpha1
kind: WerfSource
metadata:
  name: not-default-project
spec:
  imageRepo: cr.example.com/the/path
  argocdRepo:
    project: greater-good
`,
				werfSource{
					name:   "not-default-project",
					repo:   "cr.example.com/the/path",
					apiUrl: "https://cr.example.com",
					argocdRepo: &argocdRepoConfig{
						project: "greater-good",
					},
				}),
		)
	})

	Context("Converting werf sources to configs ", func() {

		ws1 := werfSource{
			name:           "ws1",
			repo:           "cr-1.example.com/the/path",
			apiUrl:         "https://cr.example.com",
			pullSecretName: "registry-credentials-1",
			argocdRepo: &argocdRepoConfig{
				project: "default",
			},
		}

		ws2 := werfSource{
			name:           "ws2",
			repo:           "cr-2.example.com/the/path",
			apiUrl:         "https://registry-api.other.com",
			pullSecretName: "registry-credentials-2",
			argocdRepo: &argocdRepoConfig{
				project: "top-secret",
			},
		}

		ws3 := werfSource{
			name: "ws3-no-creds",
			repo: "open.example.com/the/path",
			argocdRepo: &argocdRepoConfig{
				project: "default",
			},
		}

		ws4 := werfSource{
			name:           "ws4-no-repo",
			repo:           "cr-4.example.com/the/path",
			pullSecretName: "registry-credentials-4",
		}

		credGetter := mockCredGetter(map[string][]byte{
			"registry-credentials-1":      []byte(`{"auths": {"cr-1.example.com": {"username":"n-1", "password":"pwd-1" }}}`),
			"registry-credentials-2":      []byte(`{"auths": {"cr-2.example.com": {"username":"n-2", "password":"pwd-2" }}}`),
			"unused-registry-credentials": []byte(`{"auths": {"noop.example.com": {"username":"n-3", "password":"pwd-3" }}}`),
			"registry-credentials-4":      []byte(`{"auths": {"cr-4.example.com": {"username":"n-4", "password":"pwd-4" }}}`),
		})

		vals, err := mapWerfSources([]werfSource{ws1, ws2, ws3, ws4}, credGetter)

		It("returns no errors", func() {
			Expect(err).ToNot(HaveOccurred())
		})
		It("parses to argo cd repositories as expected", func() {

			Expect(vals.ArgoCD.Repositories).To(ConsistOf(
				argocdHelmOCIRepository{
					Name:     "ws1",
					URL:      "cr-1.example.com/the/path",
					Username: "n-1",
					Password: "pwd-1",
					Project:  "default",
				},
				argocdHelmOCIRepository{
					Name:     "ws2",
					URL:      "cr-2.example.com/the/path",
					Username: "n-2",
					Password: "pwd-2",
					Project:  "top-secret",
				},
				argocdHelmOCIRepository{
					Name:    "ws3-no-creds",
					URL:     "open.example.com/the/path",
					Project: "default",
				},
			))
		})

		It("parses to argo cd image updater registries as expected", func() {

			Expect(vals.ArgoCDImageUpdater.Registries).To(ConsistOf(
				imageUpdaterRegistry{
					Name:        "ws1",
					Prefix:      "cr-1.example.com",
					ApiUrl:      "https://cr.example.com",
					Credentials: "pullsecret:d8-delivery/registry-credentials-1",
					Default:     false,
				},
				imageUpdaterRegistry{
					Name:        "ws2",
					Prefix:      "cr-2.example.com",
					ApiUrl:      "https://registry-api.other.com",
					Credentials: "pullsecret:d8-delivery/registry-credentials-2",
					Default:     false,
				},
				imageUpdaterRegistry{
					Name:    "ws3-no-creds",
					Prefix:  "open.example.com",
					ApiUrl:  "https://open.example.com",
					Default: false,
				},
				imageUpdaterRegistry{
					Name:        "ws4-no-repo",
					Prefix:      "cr-4.example.com",
					ApiUrl:      "https://cr-4.example.com",
					Credentials: "pullsecret:d8-delivery/registry-credentials-4",
					Default:     false,
				},
			))

		})

	})

	Context("YAML rendering of Argo CD repo", func() {
		It("renders full struct", func() {
			b, err := yaml.Marshal(argocdHelmOCIRepository{
				Name:     "ws1",
				URL:      "cr-1.example.com/the/path",
				Username: "n-1",
				Password: "pwd-1",
				Project:  "default",
			})

			expected := `
name: ws1
password: pwd-1
project: default
url: cr-1.example.com/the/path
username: n-1
`
			Expect(err).ToNot(HaveOccurred())
			Expect("\n" + string(b)).To(Equal(expected))

		})
		It("omits optional fields", func() {
			b, err := yaml.Marshal(argocdHelmOCIRepository{
				Name:     "ws1",
				URL:      "cr-1.example.com/the/path",
				Username: "",
				Password: "",
				Project:  "default",
			})

			expected := `
name: ws1
project: default
url: cr-1.example.com/the/path
`
			Expect(err).ToNot(HaveOccurred())
			Expect("\n" + string(b)).To(Equal(expected))

		})
	})

	Context("YAML rendering of Argo CD Image Updater registry", func() {
		It("renders full struct", func() {
			b, err := yaml.Marshal(imageUpdaterRegistry{
				Name:        "ws1",
				Prefix:      "cr-1.example.com",
				ApiUrl:      "https://cr.example.com",
				Credentials: "pullsecret:d8-delivery/registry-credentials-1",
				Default:     false,
			})
			expected := `
api_url: https://cr.example.com
credentials: pullsecret:d8-delivery/registry-credentials-1
default: false
name: ws1
prefix: cr-1.example.com
`
			Expect(err).ToNot(HaveOccurred())
			Expect("\n" + string(b)).To(Equal(expected))
		})

		It("omits optional fields", func() {
			b, err := yaml.Marshal(imageUpdaterRegistry{
				Name:    "ws1",
				Prefix:  "cr-1.example.com",
				ApiUrl:  "https://cr.example.com",
				Default: false,
			})
			expected := `
api_url: https://cr.example.com
default: false
name: ws1
prefix: cr-1.example.com
`
			Expect(err).ToNot(HaveOccurred())
			Expect("\n" + string(b)).To(Equal(expected))
		})
	})

})

type mockCredGetter map[string][]byte

func (cg mockCredGetter) Get(context.Context) (map[string][]byte, error) {
	return cg, nil
}
