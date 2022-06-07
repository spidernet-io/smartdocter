# grpc-health-checking

Started the gRPC health checking server. The health checking response can be
controlled with the time delay or via http control server.

- `--delay-unhealthy-sec` - the delay to change status to NOT_SERVING.
  Endpoint reporting SERVING for `delay-unhealthy-sec` (`-1` by default)
  seconds and then NOT_SERVING. Negative value indicates always SERVING. Use `0` to
  start endpoint as NOT_SERVING.
- `--port` (default: `5000`) can be used to override the gRPC port number.
- `--http-port` (default: `8080`) can be used to override the http control server port number.
- `--service` (default: ``) can be used used to specify which service this endpoint will respond to.

Usage:

```console
    kubectl exec test-agnhost -- /agnhost grpc-health-checking \
      [--delay-unhealthy-sec 5] [--service ""] \
      [--port 5000] [--http-port 8080]

    kubectl exec test-agnhost -- curl http://localhost:8080/make-not-serving
```
