# Observability

## Demo

### Install in one script

Install Prometheus and Grafana
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

Configure a Prometheus PodMonitor
```yaml
kubectl apply -f - <<EOF
apiVersion: monitoring.coreos.com/v1
kind: PodMonitor
metadata:
  name: stateless-lb-frontend-attractor-a-1-service-monitor
  labels:
    release: prometheus
spec:
  podMetricsEndpoints:
  - port: metrics
    interval: 5s
    scheme: "https"
    tlsConfig:
        insecureSkipVerify: true
  namespaceSelector:
    matchNames: 
    - red
  selector:
    matchLabels:
      app-type: stateless-lb-frontend
EOF
```

### Grafana Dashboard

An example of a configured Grafana Dashboard is accessible here: [dashboard.json](dashboard.json).
The dashboard of this demo can be accessed by exposing the grafana service with `kubectl port-forward svc/prometheus-grafana 9000:80`. The dashboard will then accessible via `localhost:9000` with this username: `admin` and this password: `prom-operator`. Other services can be also exposed:
* Prometheus: `kubectl port-forward svc/prometheus-kube-prometheus-prometheus 9090:9090`

### Install Meridio

Make sure the `NSM_METRICS_ENABLED` environement variable is set to true in the stateless-lb container.
