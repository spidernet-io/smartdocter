# serve-hostname

This is a small util app to serve your hostname on TCP and/or UDP. Useful for testing.

The subcommand can accept the following flags:

- `tcp` (default: `false`): Serve raw over TCP.
- `udp` (default: `false`): Serve raw over UDP.
- `http` (default: `true`): Serve HTTP.
- `close` (default: `false`): Close connection per each HTTP request.
- `port` (default: `9376`): The port number to listen to.

Keep in mind that `--http` cannot be given at the same time as `--tcp` or `--udp`.

Usage:

```console
    kubectl exec test-agnhost -- /agnhost serve-hostname [--tcp] [--udp] [--http] [--close] [--port <port>]
```
