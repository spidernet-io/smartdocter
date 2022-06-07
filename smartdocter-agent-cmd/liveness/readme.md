# liveness

Starts a simple server that is alive for 10 seconds, then reports unhealthy for the rest
of its (hopefully) short existence.

Usage:

```console
    kubectl exec test-agnhost -- /agnhost liveness
```
