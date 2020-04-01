This directory holds config that can be used by [skaffold](https://skaffold.dev/) for quick iteration on the code while deploying to a k8s cluster.

## General workflow

Running in the dev mode while waiting for manual input is enabled by running the following in the root of the project - this example will work with minikube

```console
skaffold dev -f skaffold/skaffold.yaml --trigger manual --kube-context=local -n dev
```
