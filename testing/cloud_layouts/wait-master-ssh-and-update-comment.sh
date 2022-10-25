#!/bin/bash

log_file="$1"
comment_url="$2"
connection_str_out_file="$3"

if [ -z "$log_file" ]; then
  echo "Log file is required"
  exit 1
fi

if [ -z "$comment_url" ]; then
  echo "Comment url is required"
  exit 1
fi

if [ -z "$TOKEN_GITHUB_BOT" ]; then
  echo "Token env is required"
  exit 1
fi

if [ -z "$connection_str_out_file" ]; then
  echo "Connection string output file is required"
  exit 1
fi

master_ip=""
master_user=""
result_body=""

function get_comment(){
  local response_file="$1"
  local exit_code
  local http_code
  http_code="$(curl \
    --output "$response_file" \
    --write-out "%{http_code}" \
    -H "Accept: application/vnd.github+json" \
    -H "Authorization: Bearer $TOKEN_GITHUB_BOT" \
    "$comment_url"
  )"
  exit_code="$?"

  echo "Getting response (code: $http_code):"
  cat "$response_file"

  if [[ "$exit_code" != 0 ]]; then
    echo "Incorrect response code $exit_code"
    return 1
  fi

  if [[ "$http_code" != "200" ]]; then
    echo "Incorrect response code $http_code"
    return 1
  fi

  local connection_str="${master_user}@${master_ip}"
  local connection_str_body="${PROVIDER}-${LAYOUT}-${CRI}-${KUBERNETES_VERSION} - Connection string: \`ssh ${connection_str}\`"
  local bbody
  if ! bbody="$(cat "$response_file" | jq -crM --arg a "$connection_str_body" '{body: (.body + "\r\n\r\n" + $a + "\r\n")}')"; then
    return 1
  fi

  result_body="$bbody"
  echo "Result body: $result_body"
}

function update_comment(){
  local http_body="$1"
  local response_file=$(mktemp)
  local exit_code
  local http_code

  http_code="$(curl \
    -v --output "$response_file" \
    --write-out "%{http_code}" \
    -X PATCH \
    -H "Accept: application/vnd.github+json" \
    -H "Authorization: Bearer $TOKEN_GITHUB_BOT" \
    -d "$http_body" \
    "$comment_url"
  )"
  exit_code="$?"

  echo "Response update output:"
  cat "$response_file"
  rm -f "$response_file"

  if [ "$exit_code" == 0 ]; then
    if [ "$http_code" == "200" ]; then
        return 0
    fi
  fi

  echo "Comment not updated, http code: $http_code"

  return 1
}

function wait_master_host_connection_string() {
  local ip
  if ! ip="$(grep -Po '(?<=master_ip_address_for_ssh = ).+$' "$log_file")"; then
    echo "Master ip not found"
    return 1
  fi

  #https://stackoverflow.com/posts/36760050/revisions
  # we need to verify ip because string ca fsynced partially
  if ! echo "$ip" | grep -Po '((25[0-5]|(2[0-4]|1\d|[1-9]|)\d)\.?\b){4}'; then
    echo "$ip is not ip"
    return 1
  fi

  master_ip=$ip
  echo "IP found $master_ip"

  local user
  if ! user="$(grep -Po '(?<=master_user_name_for_ssh = ).+$' "$log_file")"; then
    echo "User not found"
    return 1
  fi

  if [ -z "$user" ]; then
    echo "User is empty"
    return 1
  fi

  master_user="$user"
  echo "User was found: $master_user"

  # got ip and user
  return 0
}

# wait master ip and user. 10 minutes 60 cycles wit 10 second sleep
sleep_second=0
for (( i=1; i<=60; i++ )); do
  # yep sleep before
  sleep $sleep_second
  sleep_second=10

  if wait_master_host_connection_string; then
    break
  fi
done

if [[ "$master_ip" == "" || "$master_user" == "" ]]; then
  echo "Timeout waiting master ip and master user"
  exit 1
fi

connection_str="${master_user}@${master_ip}"
echo -n "$connection_str" > "$connection_str_out_file"

# get body
sleep_second=0
for (( i=1; i<=5; i++ )); do
  sleep "$sleep_second"
  sleep_second=5

  response_file="$(mktemp)"
  if get_comment "$response_file"; then
    rm -f "$response_file"
    break
  fi

  rm -f "$response_file"
  echo "Next attempt to getting comment in 5 seconds"
done

if [ -z "$result_body" ]; then
  echo "Timeout waiting comment body"
  exit 1
fi

# update comment
sleep_second=0
for (( i=1; i<=5; i++ )); do
  sleep "$sleep_second"
  sleep_second=5

  if update_comment "$result_body" ; then
    echo "Comment was updated"
    exit 0
  fi
done

echo "Timeout waiting comment updating"
exit 1
