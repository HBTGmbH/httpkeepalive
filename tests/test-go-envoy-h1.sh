#!/bin/sh
# Ingress --HTTP/1.1--> Go --HTTP/1.1--> envoy --HTTP/1.1--> java-demo
# While below command is running, restart the envoy deployment:
#   You should NOT see any 503 errors in oha.
# When restarting the 'java-demo' deployment, you SHOULD see 503 errors, though:
#   This is because the Tomcat web server will only wait for currently active
#   HTTP requests to finish and then shut down immediately, including closing any
#   keep-alive connections. This closing of keep-alive connections races with 
#   requests a client (Envoy in this case) is just about to send over it.
#   This then results in TCP RST packets emitted by the backend server (Tomcat).
LOAD_BALANCER_IP="$(kubectl get services -n ingress-nginx ingress-nginx-controller -o jsonpath='{.status.loadBalancer.ingress[0].ip}')"
oha -c 100 -z 10m -m POST -d '{}' -H "Content-Type: application/json" -H "Host: go-demo" "http://$LOAD_BALANCER_IP/envoy/java-demo/sleep?min=50&max=200"

