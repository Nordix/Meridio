#! /bin/bash

helm repo add prometheus-community https://prometheus-community.github.io/helm-charts
helm repo add open-telemetry https://open-telemetry.github.io/opentelemetry-helm-charts
helm repo update

helm install prometheus prometheus-community/kube-prometheus-stack

kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.11.0/cert-manager.yaml

sleep 5

while kubectl get pods --no-headers | awk '$3' | grep -v "Running" > /dev/null; do sleep 1; done

sleep 15

helm install opentelemetry-operator open-telemetry/opentelemetry-operator

sleep 15

while kubectl get pods --no-headers | awk '$3' | grep -v "Running" > /dev/null; do sleep 1; done

sleep 10

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