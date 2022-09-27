---
title: "The cilium-hubble module: configuration"
---

The module is **automatically** enabled when `cni-cilium` is used.
To disable this module you can add to the `deckhouse` ConfigMap:

```yaml
ciliumHubbleEnabled: "false"
```

## Authentication

[user-authn](/{{ page.lang }}/documentation/v1/modules/150-user-authn/) module provides authentication by default. Also, externalAuthentication can be configured (see below).
If these options are disabled, the module will use basic auth with the auto-generated password.

Use kubectl to see password:

```shell
kubectl -n d8-system exec deploy/deckhouse -- deckhouse-controller module values cilium-hubble -o json | jq '.ciliumHubble.internal.auth.password'
```

Delete secret to re-generate password:

```shell
kubectl -n d8-cni-cilium delete secret/hubble-basic-auth
```
 
**Note:** auth.password parameter is deprecated.

## Parameters

<!-- SCHEMA -->
