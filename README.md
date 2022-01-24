# influxdb-operator
Kubernetes operator to manage following influxdb2 resources:
* Organizations
* Access tokens
* Buckets

The operator has four custom resource definitions associated with it
that allow for defining configuration parameters in addition to the
above three resources.

```bash
kubectl get customresourcedefinitions.apiextensions.k8s.io | grep influxdb.kubetrail.io
buckets.influxdb.kubetrail.io               2022-01-24T01:01:50Z
configs.influxdb.kubetrail.io               2022-01-24T01:01:50Z
organizations.influxdb.kubetrail.io         2022-01-24T01:01:50Z
tokens.influxdb.kubetrail.io                2022-01-24T01:01:50Z
```

The details of these resources is described later in this readme, however,
the main point to capture is that the `configs.influxdb.kubetrail.io` resource
defines parameters required for the `influxdb2` API client such as `influxdb2` connection
address and access-token secret name.

Other custom resources work with `Config` object to operate on respective
resources. `Config` object can thus be thought of as analogous to a configmap
except that being a separate CR allows us to define RBAC specifically for
it without affecting client's access permissions on native configmaps resources.

## installation
first download the code, build container image and push
to your container registry.
> please make sure go toolchain and docker are installed
> at relatively newer versions and also update the
> IMG value to point to your registry
```bash
export IMG=docker.io/your-account-name/influxdb-operator:0.0.1
make generate
make manifests
make docker-build
make docker-push
```
once the container image is available in your registry you can
deploy the controller.
> please make sure you have cert-manager and prometheus running
> on your cluster

install `cert-manager`
```bash
kubectl apply --validate=false -f https://github.com/jetstack/cert-manager/releases/download/v1.6.1/cert-manager.yaml
```

install `prometheus` after creating namespace for it and making sure
your `helm` repos are updated
```bash
helm repo add prometheus-community https://prometheus-community.github.io/helm-charts
helm repo update
helm --namespace=prometheus-system upgrade --install \
                prometheus prometheus-community/kube-prometheus-stack \
                --set=grafana.enabled=false \
                --version=27.0.1
```

install `CRD's` and the controller
```bash
make install
make deploy
```

Make sure everything is running properly:
```bash
kubectl --namespace=influxdb-operator-system get pods,svc,configmaps,secrets,servicemonitors
NAME                                                       READY   STATUS    RESTARTS   AGE
pod/influxdb-operator-controller-manager-89d8ccb67-kzrnr   2/2     Running   0          123m

NAME                                                           TYPE        CLUSTER-IP      EXTERNAL-IP   PORT(S)    AGE
service/influxdb-operator-controller-manager-metrics-service   ClusterIP   00.00.000.000   <none>        8443/TCP   136m
service/influxdb-operator-webhook-service                      ClusterIP   00.00.000.000   <none>        443/TCP    136m

NAME                                         DATA   AGE
configmap/a5b1cc23.kubetrail.io              0      136m
configmap/influxdb-operator-manager-config   1      136m
configmap/kube-root-ca.crt                   1      136m

NAME                                                      TYPE                                  DATA   AGE
secret/artifact-registry-key                              kubernetes.io/dockerconfigjson        1      136m
secret/default-token-4vzx2                                kubernetes.io/service-account-token   3      136m
secret/influxdb-operator-controller-manager-token-shvss   kubernetes.io/service-account-token   3      136m
secret/webhook-server-cert                                kubernetes.io/tls                     3      136m

NAME                                                                                        AGE
servicemonitor.monitoring.coreos.com/influxdb-operator-controller-manager-metrics-monitor   136m
```

## create organization, access token and bucket
Once CRD's are installed and operator pod is up and running as shown above, 
`influxdb2` resources can be created. To illustrate the flow, here is a 
an exmaple, which creates an organization, an access token in that organization
and lastly a bucket in that organization with that access token. There are
several cross-references of the resource names in order to ensure that 
RBAC can be effectively assigned.

```yaml
# resources below create influxdb resources and they are
# designed in this particular way for RBAC management

# Config CR defines how to reach influxdb service running
# in a namespace along with api token. In other words,
# Config CR defines things to instantiate influxdb2 api client.

# Organization CR and Token CR create respective resources,
# and they can do so via influxdb2 client instantiated via
# Config CR that can be defined in the spec. This allows
# RBAC to be permissive for these resources and restrictive
# for Bucket CR

# Bucket CR, on the other hand, does not have an ability to
# reference a Config CR and always assumes presence of a
# Config CR named default.

# config below is used by the organization CR to create new organizations
apiVersion: influxdb.kubetrail.io/v1beta1
kind: Config
metadata:
  name: config-for-org-crud               # (1) this can be referenced in organization cr
spec:
  orgName: influxdata
  tokenSecretName: influxdata-admin-token
  tokenSecretNamespace: influxdb2-system
---
apiVersion: influxdb.kubetrail.io/v1beta1
kind: Organization
metadata:
  name: sample-organization               # (2) org name to create
spec:
  configName: config-for-org-crud         # (1) reference config created earlier to access admin token
---
# config below is used by the token CR to create new tokens
apiVersion: influxdb.kubetrail.io/v1beta1
kind: Config
metadata:
  name: config-for-token-crud            # (3) config name to reference in token creation cr
spec:
  orgName: sample-organization           # (2) reference to org name created above
  tokenSecretName: influxdata-admin-token
  tokenSecretNamespace: influxdb2-system
---
apiVersion: influxdb.kubetrail.io/v1beta1
kind: Token
metadata:
  name: organization-admin-token
spec:
  configName: config-for-token-crud       # (3) refer to the config created above to access admin token
  secretName: organization-admin-token    # (4) secret name to write token to in self ns
  permissions:
    - permissionType: write
      resourceType: orgs
    - permissionType: read
      resourceType: orgs
    - permissionType: write
      resourceType: buckets
    - permissionType: read
      resourceType: buckets
---
# config below is used to create new buckets
apiVersion: influxdb.kubetrail.io/v1beta1
kind: Config
metadata:
  name: default                              # (5) bucket cr ALWAYS works via default config
spec:
  orgName: sample-organization               # (2) organization to operate in
  tokenSecretName: organization-admin-token  # (4) secret name created above
---
apiVersion: influxdb.kubetrail.io/v1beta1
kind: Bucket
metadata:
  name: sample-bucket                        # (6) name of bucket to create... always via default config (5)
spec:
  secondsTtl: 0 # or >= 3600
  description: bucket to store sensor data
```

The manifest shown above can be applied in a namespace resulting in
creation of following resources:
```bash
kubectl --namespace=influxdb-sample get \
    organizations.influxdb.kubetrail.io,tokens.influxdb.kubetrail.io,buckets.influxdb.kubetrail.io
NAME                                                     STATUS   AGE
organization.influxdb.kubetrail.io/sample-organization   ready    130m

NAME                                                   STATUS   AGE
token.influxdb.kubetrail.io/organization-admin-token   ready    130m

NAME                                         STATUS   AGE
bucket.influxdb.kubetrail.io/sample-bucket   ready    130m
```
