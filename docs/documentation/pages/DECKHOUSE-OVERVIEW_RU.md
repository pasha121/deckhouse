---
title: Введение в документацию
permalink: ru/deckhouse-overview.html
lang: ru
---

Приветствуем вас на главной странице документации платформы для управления Kubernetes-кластерами Deckhouse. Если вы ещё не попробовали платформу — начните с [Getting started]({{ site.url }}/{{ page.lang }}/gs/), где есть пошаговые инструкции для развёртывания платформы на любой инфраструктуре.

Как быстро найти то, что нужно:
<ul>
<li>Документация разных версий Deckhouse может отличаться. В выпадающем списке справа вверху страницы вы можете выбрать нужную вам версию документации.</li>
<li>Если вы точно знаете, что ищете — воспользуйтесь поиском наверху страницы.</li>
<li>Если вам нужна информация по конкретному модулю — найдите его <a href="revision-comparison.html">в этом списке</a>.</li>
<li>Меню слева для поиска по области применения. Подумайте, к какому разделу логически может относиться то, что вы ищете.
  {% offtopic title="Если сомневаетесь, вот краткое описание разделов..." %}
  <div markdown="1">
  - Deckhouse — глобальные настройки и общая информация по платформе.
  - Кластер Kubernetes — всё, что относится к control plane, а также интеграция с облачными провайдерами, управление узлами, управление сетью и т.д.
  - Доступ к кластеру — инструменты, чтобы зайти в кластер ([openvpn]({{ site.url }}/{{ page.lang }}/documentation/v1/modules/500-openvpn/)) и управлять объектами ([dashboard](modules/500-dashboard/)).
  - Балансировка трафика — [Ingress на базе nginx](modules/402-ingress-nginx/) и возможности [Istio]({{ site.url }}/{{ page.lang }}/documentation/v1/modules/110-istio/).
  - Мониторинг — [Prometheus/Grafana](modules/300-prometheus/), [мониторинг ваших приложений](modules/340-monitoring-custom/), а также [сбор логов](modules/460-log-shipper/).
  - Масштабирование и управление ресурсами — всё что касается управления Pod’ами и масштабирования.
  - Безопасность — [аутентификация](modules/150-user-authn/), [авторизация](modules/140-user-authz/) и [управление сертификатами](modules/101-cert-manager/).
  - Хранилище — [интеграция с Ceph](modules/099-ceph-csi/), работа [с локальным хранилищем](modules/031-local-path-provisioner/) на узлах, а также организация [хранилища на базе Linstor](modules/041-linstor/).
  - Приятные мелочи — [синхронизация времени](modules/470-chrony/) на узлах, автоматическое [копирование секретов]({{ site.url }}/{{ page.lang }}/documentation/v1/modules/600-secret-copier/) по пространствам имен и другие удобства.
  - Поддержка Bare metal — модули для комфортной работы c кластером на железе.
  </div>
  {% endofftopic %}
</li>
</ul>
Не нашли то, что нужно? Попросите помощи в нашем [Telegram-канале]({{ site.social_links[page.lang]['telegram'] }}). Вас там точно не оставят один на один с проблемой.

Если вы используете Enterprise Edition, [напишите нам](mailto:support@deckhouse.io) — мы обязательно поможем.

Знаете, как сделать Deckhouse лучше? Заведите [задачу](https://github.com/deckhouse/deckhouse/issues/), [обсудите](https://github.com/deckhouse/deckhouse/discussions) с нами вашу идею или даже предложите [решение](https://github.com/deckhouse/deckhouse/blob/main/CONTRIBUTING.md).

Хочется чего-то большего? [Присоединяйтесь](https://job.flant.ru/) к команде! Мы всегда рады единомышленникам!
