## Chart-Sentry Diff Report

| Chart | Environment | Status |
|-------|-------------|--------|
| my-app | prod | Changed |
| my-app | staging | Changed |

### my-app/prod
<details><summary>View diff</summary>

```diff
--- my-app/prod (main)
+++ my-app/prod (feat/update-config)

metadata.labels.version  (apps/v1/Deployment/my-app)
  ± value change
    - 0.1.0
    + 0.2.0

spec.replicas  (apps/v1/Deployment/my-app)
  ± value change
    - 3
    + 5

spec.template.spec.containers.my-app.image  (apps/v1/Deployment/my-app)
  ± value change
    - my-app:1.24.0
    + my-app:1.25.0

spec.template.spec.containers.my-app.env  (apps/v1/Deployment/my-app)
  + one list entry added:
    - name: ENABLE_CACHE
      value: "true"

spec.template.spec.containers.my-app.resources.limits.cpu  (apps/v1/Deployment/my-app)
  ± value change
    - 1
    + 2

spec.template.spec.containers.my-app.resources.limits.memory  (apps/v1/Deployment/my-app)
  ± value change
    - 1Gi
    + 2Gi

(root level)  (v1/ConfigMap/my-app-config)
+ one document added:
  ---
  # Source: my-app/templates/configmap.yaml
  apiVersion: v1
  kind: ConfigMap
  metadata:
    name: my-app-config
    labels:
      app: my-app
  data:
    LOG_LEVEL: warn
    CACHE_ENABLED: "true"
```
</details>

### my-app/staging
<details><summary>View diff</summary>

```diff
--- my-app/staging (main)
+++ my-app/staging (feat/update-config)

metadata.labels.version  (apps/v1/Deployment/my-app)
  ± value change
    - 0.1.0
    + 0.2.0

spec.replicas  (apps/v1/Deployment/my-app)
  ± value change
    - 2
    + 3

spec.template.spec.containers.my-app.image  (apps/v1/Deployment/my-app)
  ± value change
    - my-app:1.24.0
    + my-app:1.25.0

spec.template.spec.containers.my-app.env  (apps/v1/Deployment/my-app)
  + one list entry added:
    - name: ENABLE_CACHE
      value: "true"

(root level)  (v1/ConfigMap/my-app-config)
+ one document added:
  ---
  # Source: my-app/templates/configmap.yaml
  apiVersion: v1
  kind: ConfigMap
  metadata:
    name: my-app-config
    labels:
      app: my-app
  data:
    LOG_LEVEL: debug
    CACHE_ENABLED: "true"
```
</details>

