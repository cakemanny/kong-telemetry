
# Kong Telemetry Demo

This is an example project showing how the OpenTelemetry plugin within
kong can be used to send traces to Jaeger.
It also implements OpenTelemetry instrumentation in an upstream service
and a Kong plugin written in GO

## Using

### Tools
- [kind](https://kind.sigs.k8s.io/) or [minikube](https://minikube.sigs.k8s.io/)
  for running a local kubernetes cluster.
- [tilt](https://tilt.dev/) to build and deploy to the cluster

### Setting Up

Make sure you've started your cluster. If you chose `kind`, that might look
like
```shell
kind create cluster
```

Get everything up and running
```shell
tilt up
```

Give our demo service, dice, a bit of traffic
```shell
curl -i localhost:8000/rolldice
```

Or hit the plugin
```shell
curl -i localhost:8000/plugin
```

Go to http://localhost:16686/ to view the Jaeger UI and you can search for
some traces.

### Updating Kong declarative config

If you want to make changes to `kong.yml` without having to restart Kong,
with `xh` installed, it's very simply to update.

```shell
xh POST localhost:8001/config -- config=@kong.yml
```
<!--
TODO: check if this works with curl

    curl -i -X POST http://localhost:8001/config -d config=@kong.yml
-->

### Cleaning up

Running `tilt down` will have tilt remove everything from the cluster

Remove the cluster
```shell
kind delete cluster  # Or
minikube delete --all  # if you're like that
```

And even the docker images if you won't use them:
```
docker image ls | awk '$1=="dice" || $1=="go-plugins" { print $1 ":" $2 }' | xargs docker image rm
```
