#!/bin/sh
# Ingress --HTTP/1.1--> Go --HTTP/1.1--> envoy --HTTP/2--> java-demo
# While below command is running, restart the java-demo deployment:
#   You might see 503 errors in oha depending on the DNS refresh rate of the cluster.
#   GKE with kube-dns, for example, stores answers with a TTL of 30s (regardless of how often
#   kube-dns is queried. So, a dns_refresh interval of 3s in Envoy will not help here.)
#   If the backend service has a shutdown time (between being in Terminating state and actually
#   shutting down) of less than 30s, you will see 503 errors, because kube-dns answers still
#   contain the terminating hosts and Envoy failing to connect to them.
LOAD_BALANCER_IP="$(kubectl get services -n ingress-nginx ingress-nginx-controller -o jsonpath='{.status.loadBalancer.ingress[0].ip}')"
oha -c 100 -z 10m -m POST -d '{}' -H "Content-Type: application/json" -H "Host: go-demo" "http://$LOAD_BALANCER_IP/envoy/java-demo/sleep?min=50&max=200&h2&dns"
