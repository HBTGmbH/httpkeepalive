#!/bin/sh
# Ingress --HTTP/1.1--> Go
# While below command is running, restart the go-demo deployment:
#   You should NOT see any 5xx errors in oha.
#   This is because the Go application has a graceful shutdown,
#   where it will still accept incoming connections and incoming requests on
#   existing connections, but respond with 'Connection: close' for requests,
#   which eventually leads the client to not initiate new connections anymore,
#   because the Ingress controller has knowledge of terminating pods.
LOAD_BALANCER_IP="$(kubectl get services -n ingress-nginx ingress-nginx-controller -o jsonpath='{.status.loadBalancer.ingress[0].ip}')"
oha -c 100 -z 10m -m POST -d '{}' -H "Content-Type: application/json" -H "Host: go-demo" "http://$LOAD_BALANCER_IP/sleep?min=50ms&max=200ms"
