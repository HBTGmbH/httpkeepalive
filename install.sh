#!/bin/sh

: "${IMAGE_REPO_PREFIX:?IMAGE_REPO_PREFIX must be set (non-empty)}"
helm upgrade --install --namespace ingress-nginx --create-namespace -f ingress-settings.yaml --repo https://kubernetes.github.io/ingress-nginx ingress-nginx ingress-nginx
envsubst '$IMAGE_REPO_PREFIX' < go-demo/deployment.yaml | kubectl apply -f -
envsubst '$IMAGE_REPO_PREFIX' < java-demo/deployment.yaml | kubectl apply -f -
envsubst '$IMAGE_REPO_PREFIX' < node-demo/deployment.yaml | kubectl apply -f -
kubectl apply -f nginx/deployment.yaml
kubectl apply -f envoy/deployment.yaml
kubectl apply -f varnish/deployment.yaml
