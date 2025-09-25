# neo4j-aura-controller

[![release](https://img.shields.io/github/release/DoodleScheduling/neo4j-aura-controller/all.svg)](https://github.com/DoodleScheduling/neo4j-aura-controller/releases)
[![release](https://github.com/DoodleScheduling/neo4j-aura-controller/actions/workflows/release.yaml/badge.svg)](https://github.com/DoodleScheduling/neo4j-aura-controller/actions/workflows/release.yaml)
[![report](https://goreportcard.com/badge/github.com/DoodleScheduling/neo4j-aura-controller)](https://goreportcard.com/report/github.com/DoodleScheduling/neo4j-aura-controller)
[![OpenSSF Scorecard](https://api.securityscorecards.dev/projects/github.com/DoodleScheduling/neo4j-aura-controller/badge)](https://api.securityscorecards.dev/projects/github.com/DoodleScheduling/neo4j-aura-controller)
[![Coverage Status](https://coveralls.io/repos/github/DoodleScheduling/neo4j-aura-controller/badge.svg?branch=master)](https://coveralls.io/github/DoodleScheduling/neo4j-aura-controller?branch=master)
[![license](https://img.shields.io/github/license/DoodleScheduling/neo4j-aura-controller.svg)](https://github.com/DoodleScheduling/neo4j-aura-controller/blob/master/LICENSE)

## Deploy an instance

```yaml
apiVersion: neo4j.infra.doodle.com/v1beta1
kind: AuraInstance
metadata:
  name: my-instance
spec:
  cloudProvider: gcp
  memory: 4GB
  region: eu-central-1
  tier: free-db
  neo4jVersion: "5"
  secret:
    name: neo4j-project-admin
---
apiVersion: v1
data:
  clientID: c2VjcmV0=
  clientSecret: c2VjcmV0=
kind: Secret
metadata:
  name: neo4j-project-admin
type: Opaque
```

## Observe reconciliation

Each resource reports various conditions in `.status.condtions` which will give the necessary insight about the 
current state of the resource.

```yaml
status:
  conditions:
  - lastTransitionTime: "2023-11-30T12:01:52Z"
    message: random cloud error
    observedGeneration: 32
    reason: ReconciliationFailed
    status: "False"
    type: Ready
  - lastTransitionTime: "2023-12-11T14:03:31Z"
    message: selector matches at least one running pod
    observedGeneration: 3
    reason: PodsRunning
    status: "False"
    type: ScaledToZero
```

## Installation

### Helm

Please see [chart/neo4j-aura-controller](https://github.com/DoodleScheduling/neo4j-aura-controller/tree/master/chart/neo4j-aura-controller) for the helm chart docs.

### Manifests/kustomize

Alternatively you may get the bundled manifests in each release to deploy it using kustomize or use them directly.

## Configuration
The controller can be configured using cmd args:
```
--concurrent int                            The number of concurrent reconciles. (default 4)
--enable-leader-election                    Enable leader election for controller manager. Enabling this will ensure there is only one active controller manager.
--graceful-shutdown-timeout duration        The duration given to the reconciler to finish before forcibly stopping. (default 10m0s)
--health-addr string                        The address the health endpoint binds to. (default ":9557")
--insecure-kubeconfig-exec                  Allow use of the user.exec section in kubeconfigs provided for remote apply.
--insecure-kubeconfig-tls                   Allow that kubeconfigs provided for remote apply can disable TLS verification.
--kube-api-burst int                        The maximum burst queries-per-second of requests sent to the Kubernetes API. (default 300)
--kube-api-qps float32                      The maximum queries-per-second of requests sent to the Kubernetes API. (default 50)
--leader-election-lease-duration duration   Interval at which non-leader candidates will wait to force acquire leadership (duration string). (default 35s)
--leader-election-release-on-cancel         Defines if the leader should step down voluntarily on controller manager shutdown. (default true)
--leader-election-renew-deadline duration   Duration that the leading controller manager will retry refreshing leadership before giving up (duration string). (default 30s)
--leader-election-retry-period duration     Duration the LeaderElector clients should wait between tries of actions (duration string). (default 5s)
--log-encoding string                       Log encoding format. Can be 'json' or 'console'. (default "json")
--log-level string                          Log verbosity level. Can be one of 'trace', 'debug', 'info', 'error'. (default "info")
--max-retry-delay duration                  The maximum amount of time for which an object being reconciled will have to wait before a retry. (default 15m0s)
--metrics-addr string                       The address the metric endpoint binds to. (default ":9556")
--min-retry-delay duration                  The minimum amount of time for which an object being reconciled will have to wait before a retry. (default 750ms)
--watch-all-namespaces                      Watch for resources in all namespaces, if set to false it will only watch the runtime namespace. (default true)
--watch-label-selector string               Watch for resources with matching labels e.g. 'sharding.fluxcd.io/shard=shard1'.
```
