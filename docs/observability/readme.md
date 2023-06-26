# Observability

## Demo

### Install in one script

Install Prometheus, Grafana and Open Telemetry
```
./docs/demo/deployments/optl-prometheus-grafana/deploy.sh
```

### Install Step by Step

Install Prometheus/Grafana Stack
```
helm repo add prometheus-community https://prometheus-community.github.io/helm-charts
helm repo update
helm install prometheus prometheus-community/kube-prometheus-stack
```

Install cert-manager (required for the Open Telemetry operator)
```
kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.11.0/cert-manager.yaml
```

Install the Open Telemetry operator
```
helm repo add open-telemetry https://open-telemetry.github.io/opentelemetry-helm-charts
helm repo update
helm install opentelemetry-operator open-telemetry/opentelemetry-operator
```

Deploy an Open Telemetry Collector:
```
kubectl apply -f - <<EOF
apiVersion: opentelemetry.io/v1alpha1
kind: OpenTelemetryCollector
metadata:
  name: ot
spec:
  mode: deployment
  ports:
    - name: prometheus
      port: 9464
      targetPort: 9464
      protocol: TCP
  config: |
    receivers:
      jaeger:
        protocols:
          grpc:
      otlp:
        protocols:
          grpc:
          http:

    processors:

    exporters:
      logging:
        verbosity: detailed
      prometheus:
        endpoint: 0.0.0.0:9464
        metric_expiration: 30s

    service:
      pipelines:
        traces:
          receivers: [ jaeger ]
          processors: []
          exporters: [ logging ]
        metrics:
          receivers: [ otlp ]
          exporters: [ prometheus, logging ]
EOF
```

Configure a Prometheus ServiceMonitor
```
kubectl apply -f - <<EOF
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: ot-collector-service-monitor
  labels:
    release: prometheus
spec:
  endpoints:
  - port: prometheus
  selector:
    matchLabels:
      app.kubernetes.io/name: "ot-collector"
EOF
```

### Grafana Dashboard

An example of a configured Grafana Dashboard is accessible here: [dashboard.json](dashboard.json).
The dashboard of this demo can be accessed by exposing the grafana service with `kubectl port-forward svc/prometheus-grafana 9000:80`. The dashboard will then accessible via `localhost:9000` with this username: `admin` and this password: `prom-operator`. Other services can be also exposed:
* Prometheus: `kubectl port-forward svc/prometheus-kube-prometheus-prometheus 9090:9090`
* Open Telemetry Collector: `kubectl port-forward svc/ot-collector 9464:9464`

### Install Meridio

Make sure the `OT_COLLECTOR_ENABLED` environement variable is set to true in the stateless-lb container.
