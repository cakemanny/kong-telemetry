
# Kong Telemetry Demo

This is an example project showing how the OpenTelemetry plugin within
kong can be used to send traces to Elastic APM

## Using

Get everything up and running
```console
$ docker compose up -d --wait
```

Give our demo service, dice, a bit of traffic
```console
$ curl localhost:80/rolldice
```

Go to http://localhost:5601/app/apm/services
Login as `elastic` using `elastopass` as the password


## Cleaning up
```console
$ docker compose down -v --remove-orphans
```

## Yet to try

Add a small go plugin like the one in
[bettermarks/kong\_8531](https://github.com/bettermarks/kong_8531/tree/simplified-goplugin-repro-3-1-0)
but one that also has some otel instrumentation.
