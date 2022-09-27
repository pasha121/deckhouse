---
title: "Модуль cilium-hubble: настройки"
---

Модуль включается **автоматически** если включен `cni-cilium` модуль.
Для выключения, необходимо в ConfigMap `deckhouse` добавить:

```yaml
ciliumHubbleEnabled: "false"
```

## Аутентификация

По умолчанию используется модуль [user-authn](/{{ page.lang }}/documentation/v1/modules/150-user-authn/). Также можно настроить аутентификацию через `externalAuthentication` (см. ниже).
Если эти варианты отключены, то модуль включит basic auth со сгенерированным паролем.

Посмотреть сгенерированный пароль можно командой:

```shell
kubectl -n d8-system exec deploy/deckhouse -- deckhouse-controller module values cilium-hubble -o json | jq '.ciliumHubble.internal.auth.password'
```

Чтобы сгенерировать новый пароль, нужно удалить секрет:

```shell
kubectl -n d8-cni-cilium delete secret/hubble-basic-auth
```

**Внимание:** параметр `auth.password` больше не поддерживается.

## Параметры

<!-- SCHEMA -->
