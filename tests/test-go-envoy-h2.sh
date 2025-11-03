#!/bin/sh
# Ingress --HTTP/1.1--> Go --HTTP/1.1--> envoy --HTTP/2--> java-demo
# While below command is running, restart the java-demo deployment.
# You MIGHT see 503 errors due to Tomcat sending the final GOAWAY quickly after the first graceful GOAWAY.
# See: https://github.com/apache/tomcat/pull/917
# With this patch, you WILL NOT see any 503 errors in oha, because Tomcat leaves Envoy enough time to
# not race new requests with the final GOAWAY.
LOAD_BALANCER_IP="$(kubectl get services -n ingress-nginx ingress-nginx-controller -o jsonpath='{.status.loadBalancer.ingress[0].ip}')"
oha -c 100 -z 10m -m POST -d '{}' -H "Content-Type: application/json" -H "Host: go-demo" "http://$LOAD_BALANCER_IP/envoy/java-demo/sleep?min=50&max=200&h2"
