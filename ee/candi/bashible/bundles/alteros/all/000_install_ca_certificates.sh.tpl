# Copyright 2022 Flant JSC
# Licensed under the Deckhouse Platform Enterprise Edition (EE) license. See https://github.com/deckhouse/deckhouse/blob/main/ee/LICENSE.

bb-yum-install ca-certificates
# hack to avoid problems with certs in alpine busybox for kube-apiserver
if [[ ! -e /etc/ssl/certs/ca-certificates.crt ]]; then
  ln -s /etc/ssl/certs/ca-bundle.crt /etc/ssl/certs/ca-certificates.crt
fi

{{- if .registry.ca }}
bb-event-on 'registry-ca-changed' '_update_ca_certificates'
function _update_ca_certificates() {
  update-ca-trust
}

bb-sync-file /etc/pki/ca-trust/source/anchors/registry-ca.crt - registry-ca-changed << "EOF"
{{ .registry.ca }}
EOF
{{- end }}
