# porter

Serves requested data on ports specified in environment variables of the form `SERVE_{PORT,TLS_PORT,SCTP_PORT}_[NNNN]`. eg:

- `SERVE_PORT_9001` - serve TCP connections on port 9001
- `SERVE_TLS_PORT_9002` - serve TLS-encrypted TCP connections on port 9002
- `SERVE_SCTP_PORT_9003` - serve SCTP connections on port 9003

The included `localhost.crt` is a PEM-encoded TLS cert with SAN IPs `127.0.0.1` and `[::1]`,
expiring in January 2084, generated from `src/crypto/tls`:

```console
    go run generate_cert.go  --rsa-bits 2048 --host 127.0.0.1,::1,example.com --ca --start-date "Jan 1 00:00:00 1970" --duration=1000000h
```

To use a different cert/key, mount them into the pod and set the `CERT_FILE` and `KEY_FILE`
environment variables to the desired paths.

Usage:

```console
    kubectl exec test-agnhost -- /agnhost porter
```
