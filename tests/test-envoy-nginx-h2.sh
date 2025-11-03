#!/bin/sh
# Ingress --HTTP/1.1--> Envoy --HTTP/2--> nginx --HTTP/1.1--> java-demo
# While below command is running, restart the nginx deployment:
#   You MIGHT see 503 errors due to Nginx sending only a single final GOAWAY with the last seen stream id
#   and does _not_ do the double-GOAWAY graceful shutdown of the session/connection using a first GOAWAY
#   with max stream id (2^31-1) followed by a final GOAWAY with the actual last seen stream id.
#   Instead, nginx only sends a single GOAWAY and then immediately rejects any requests/streams following that.
#   This then creates a race condition between the client (Envoy) sending a new stream/request right as the server (nginx)
#   is sending its single/final GOAWAY.
#   Nginx is not using double-GOAWAY because:
#     - https://trac.nginx.org/nginx/ticket/2224
#     - https://issues.chromium.org/issues/40661777
LOAD_BALANCER_IP="$(kubectl get services -n ingress-nginx ingress-nginx-controller -o jsonpath='{.status.loadBalancer.ingress[0].ip}')"
oha -c 100 -z 10m -m POST -d '{}' -H "Content-Type: application/json" -H "Host: envoy" "http://$LOAD_BALANCER_IP/nginx/java-demo/sleep?min=50&max=200&h2"

