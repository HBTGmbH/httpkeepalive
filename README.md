# What

This repository implements a set of HTTP services using different runtimes to showcase their behaviour regarding keepalive connections when:
- doing rolling updates (old pods terminating and new ones starting)
- sending requests at the boundary of keep-alive intervals

# Why

To demonstrate that HTTP/1.1 persistent connections (aka. keep-alive) can lead to traffic disruption when not appropriately configuring the client/server or not implementing graceful shutdown correctly.

# How

Either:
- run any of the provided `tests/test-*.sh` scripts, which will query Kubernetes for the external ingress load balancer IP and run `oha` for a few minutes, during which you do a rolling update / restart of the called services
- instead of the provided test scripts or instead of `oha` you can use any HTTP load balancing tool. You just need to make sure to test requests using the POST method (as most HTTP clients will do transparent retries for network errors when doing GET)

In order to route traffic through a set of selected services, first the `Host` request header determines which ingress service the request hits first. This can be either `envoy`, `nginx`, `varnish` or `go-demo`.
The path part of the URL denotes the set of services to route the traffic through. For example, the following will first hit envoy, then nginx and lastly node-demo with the /sleep endpoint of node-demo: `curl -X POST -H 'Host: envoy' http://YOUR_SERVER/nginx/node-demo/sleep`. And the following with go to Varnish, then nginx and then go-demo: `curl -X POST -H 'Host: varnish' http://YOUR_SERVER/nginx/go-demo/sleep`.

