# neo4j controller helm chart

Installs the [neo4j-aura-controller](https://github.com/DoodleScheduling/neo4j-aura-controller).

## Installing the Chart

To install the chart with the release name `neo4j-aura-controller`:

```console
helm upgrade --install neo4j-aura-controller oci://ghcr.io/doodlescheduling/charts/neo4j-aura-controller
```

This command deploys the neo4j-aura-controller with the default configuration. The [configuration](#configuration) section lists the parameters that can be configured during installation.

## Prometheus

The chart comes with a ServiceMonitor/PodMonitor for use with the [Prometheus Operator](https://github.com/coreos/prometheus-operator) which are disabled by default.
If you're not using the Prometheus Operator, you can populate the `podAnnotations` as below:

```yaml
podAnnotations:
  prometheus.io/scrape: "true"
  prometheus.io/port: "metrics"
  prometheus.io/path: "/metrics"
```

## Configuration

See customizing the chart before installing. To see all configurable options with detailed comments, visit the chart's values.yaml, or run the configuration command:

```sh
$ helm show values oci://ghcr.io/doodlescheduling/charts/neo4j-aura-controller
```
