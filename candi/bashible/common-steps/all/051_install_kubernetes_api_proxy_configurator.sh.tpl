bb-event-on 'bb-sync-file-changed' '_on_kubernetes_api_proxy_configurator_changed'
_on_kubernetes_api_proxy_configurator_changed() {
  if systemctl is-enabled --quiet kubernetes-api-proxy 2>/dev/null ; then
    systemctl restart kubernetes-api-proxy-configurator
  fi
}

bb-sync-file /var/lib/bashible/kubernetes-api-proxy-configurator.sh - << "EOF"
#!/bin/bash

function kubectl_exec() {
  attempt=0
  until kubectl --request-timeout 20s --kubeconfig=/etc/kubernetes/kubelet.conf ${@}; do
    attempt=$(( attempt + 1 ))
    if [ "$attempt" -gt "2" ]; then
      exit 1
    fi
  done
}

# Read from command args
apiserver_endpoints=$@
apiserver_backup_endpoints=""

if [ -z "$apiserver_endpoints" ] ; then
  self_node_addresses=""
  if self_node=$(kubectl_exec get node $HOSTNAME -o json); then
    self_node_addresses="$(echo "$self_node" | jq '.status.addresses[] | .address' -r)"
  fi

  if eps=$(kubectl_exec -n default get endpoints kubernetes -o json) ; then
    for ep in $(echo "$eps" | jq '.subsets[] | (.ports[0].port | tostring) as $port | .addresses[] | .ip + ":" +  $port' -r | sort) ; do
      ip_regex=$(echo $ep | cut -d: -f1 | sed 's/\./\\./g')

      if echo "$self_node_addresses" | grep $ip_regex > /dev/null ; then
        apiserver_endpoints="$apiserver_endpoints $ep"
      else
        # If endpoint is not local treat it as a backup
        apiserver_backup_endpoints="$apiserver_backup_endpoints $ep"
      fi
    done

    # If there are no local enpoints use remote normally
    if [ -z "$apiserver_endpoints" ] ; then
      apiserver_endpoints="$apiserver_backup_endpoints"
      apiserver_backup_endpoints=""
    fi
  fi
fi

# If there are no endpoints, try to get endpoint from locally running apiserver
if [ -z "$apiserver_endpoints" ] && [ -f /etc/kubernetes/manifests/kube-apiserver.yaml ] ; then
  if local_apiserver_endpoint="$(netstat -tlpn | grep -E -o '[0-9\.]+:6443')" ; then
    apiserver_endpoints="$local_apiserver_endpoint"
  fi

  secure_port="$(ps -e -o command | grep kube-apiserver | grep -E -o '\-\-secure-port=[0-9]+' | cut -d= -f2)"
  if [ "x$secure_port" == "x" ] ; then
    secure_port="6443"
  fi

  local_apiserver_endpoint="$(ps -e -o command | grep kube-apiserver | grep -E -o '\-\-advertise-address=[0-9\.]+' | cut -d= -f2)"
  if [ "x$local_apiserver_endpoint" != "x" ] ; then
    apiserver_endpoints="$apiserver_endpoints $local_apiserver_endpoint:$secure_port"
  fi

  local_apiserver_endpoint="$(ps -e -o command | grep kube-apiserver | grep -E -o '\-\-bind-address=[0-9\.]+' | cut -d= -f2)"
  if [ "x$local_apiserver_endpoint" != "x" ] ; then
    apiserver_endpoints="$apiserver_endpoints $local_apiserver_endpoint:$secure_port"
  fi
fi

# Fail, if there are no endpoints
if [ -z "$apiserver_endpoints" ] ; then
  exit 1
fi

# Generate nginx config (to the temporary location)
cat > /etc/kubernetes/kubernetes-api-proxy/nginx_new.conf << END
{{- if eq .bundle "ubuntu-lts" }}
user www-data;
{{- else if eq .bundle "centos-7" }}
user nginx;
{{- end }}

pid /var/run/kubernetes-api-proxy.pid;
error_log stderr notice;

worker_processes 2;
worker_rlimit_nofile 130048;
worker_shutdown_timeout 10s;

events {
  multi_accept on;
  use epoll;
  worker_connections 16384;
}

stream {
  upstream kubernetes {
    least_conn;
$(
for ep in $apiserver_endpoints ; do
  echo -e "    server $ep;"
done
for ep in $apiserver_backup_endpoints ; do
  echo -e "    server $ep backup;"
done
)
  }
  server {
    listen 127.0.0.1:6445;
    proxy_pass kubernetes;
    # Configurator uses 24h proxy_timeout in case of long running jobs like kubectl exec or kubectl logs
    # After time out, nginx will force a client to reconnect
    proxy_timeout 24h;
    proxy_connect_timeout 1s;
  }
}
END

if [ ! -f /etc/kubernetes/kubernetes-api-proxy/nginx.conf ] ; then
  echo "[INFO] setting up new nginx.conf"
  mv /etc/kubernetes/kubernetes-api-proxy/nginx_new.conf /etc/kubernetes/kubernetes-api-proxy/nginx.conf
  systemctl restart kubernetes-api-proxy
elif
  ! diff -u /etc/kubernetes/kubernetes-api-proxy/nginx.conf /etc/kubernetes/kubernetes-api-proxy/nginx_new.conf &&
  nginx -t -c /etc/kubernetes/kubernetes-api-proxy/nginx_new.conf
then
  echo "[INFO] nginx.conf changed!"
  mv /etc/kubernetes/kubernetes-api-proxy/nginx_new.conf /etc/kubernetes/kubernetes-api-proxy/nginx.conf
  systemctl reload kubernetes-api-proxy
fi
EOF

chmod +x /var/lib/bashible/kubernetes-api-proxy-configurator.sh
