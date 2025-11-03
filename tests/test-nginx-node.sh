#!/bin/sh
# Ingress --HTTP/1.1--> Nginx --HTTP/1.1--> node-demo
# While below command is running, restart the node-demo deployment:
#   You should NOT see any 502 errors from Nginx, because the Node application
#   is actively draining connections for all requests once graceful shutdown
#   is initiated by returning 'Connection: close' to all received requests, making
#   Nginx (or rather kube-proxy/netfilter) eventually route to other node-demo pods.
#   This is configurable in the node-demo deployment using the container's ENABLE_DRAIN
#   environment variable. If this is not set or set to anything other than "true",
#   the node-demo application will not drain connections and you will see 502 errors.
LOAD_BALANCER_IP="$(kubectl get services -n ingress-nginx ingress-nginx-controller -o jsonpath='{.status.loadBalancer.ingress[0].ip}')"
oha -c 200 -z 10m -m POST -d '{}' -H "Content-Type: application/json" -H "Host: nginx" "http://$LOAD_BALANCER_IP/node-demo/sleep?min=50&max=300"
