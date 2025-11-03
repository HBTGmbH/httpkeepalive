# What

This repository implements a set of HTTP services using different runtimes to showcase their behaviour regarding
keepalive connections when:
- doing rolling updates (old pods terminating and new ones starting)
- sending requests at the boundary of keep-alive intervals

# Why

To demonstrate that HTTP/1.1 persistent connections (aka. keep-alive) can lead to traffic disruption when not
appropriately configuring the client/server or not implementing graceful shutdown correctly.

# How

Use a load generation tool, like oha, to send requests through a set of services deployed in a Kubernetes cluster.
The services are implemented using different runtimes (Go, Java, Node.js, Nginx, Varnish, Envoy) to showcase their
behaviour regarding keep-alive connections.

In order to route traffic through a set of selected services, first the `Host` request header determines which ingress
service the request hits first. This can be either `envoy`, `nginx`, `varnish` or `go-demo`.
The path part of the URL denotes the set of services to route the traffic through. For example, the following will first
hit envoy, then nginx and lastly node-demo with the /sleep endpoint of node-demo:
`curl -X POST -H 'Host: envoy' http://YOUR_SERVER/nginx/node-demo/sleep`. And the following with go to Varnish, then
nginx and then go-demo: `curl -X POST -H 'Host: varnish' http://YOUR_SERVER/nginx/go-demo/sleep`.