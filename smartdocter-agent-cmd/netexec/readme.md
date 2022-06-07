# netexec

Starts a HTTP(S) server on given port with the following endpoints:

- `/`: Returns the request's timestamp.
- `/clientip`: Returns the request's IP address.
- `/dial`: Creates a given number of requests to the given host and port using the given protocol,
  and returns a JSON with the fields `responses` (successful request responses) and `errors` (
  failed request responses). Returns `200 OK` status code if the last request succeeded,
  `417 Expectation Failed` if it did not, or `400 Bad Request` if any of the endpoint's parameters
  is invalid. The endpoint's parameters are:
  - `host`: The host that will be dialed.
  - `port`: The port that will be dialed.
  - `request`: The HTTP endpoint or data to be sent through UDP. If not specified, it will result
      in a `400 Bad Request` status code being returned.
  - `protocol`: The protocol which will be used when making the request. Default value: `http`.
      Acceptable values: `http`, `udp`, `sctp`.
  - `tries`: The number of times the request will be performed. Default value: `1`.
- `/echo`: Returns the given `msg` (`/echo?msg=echoed_msg`), with the optional status `code`.
- `/exit`: Closes the server with the given code and graceful shutdown. The endpoint's parameters
  are:
  - `code`: The exit code for the process. Default value: 0. Allows an integer [0-127].
  - `timeout`: The amount of time to wait for connections to close before shutting down.
      Acceptable values are golang durations. If 0 the process will exit immediately without
      shutdown.
  - `wait`: The amount of time to wait before starting shutdown. Acceptable values are
      golang durations. If 0 the process will start shutdown immediately.
- `/healthz`: Returns `200 OK` if the server is ready, `412 Status Precondition Failed`
  otherwise. The server is considered not ready if the UDP server did not start yet or
  it exited.
- `/hostname`: Returns the server's hostname.
- `/hostName`: Returns the server's hostname.
- `/redirect`: Returns a redirect response to the given `location`, with the optional status `code`
  (`/redirect?location=/echo%3Fmsg=foobar&code=307`).
- `/shell`: Executes the given `shellCommand` or `cmd` (`/shell?cmd=some-command`) and
  returns a JSON containing the fields `output` (command's output) and `error` (command's
  error message). Returns `200 OK` if the command succeeded, `417 Expectation Failed` if not.
- `/shutdown`: Closes the server with the exit code 0.
- `/upload`: Accepts a file to be uploaded, writing it in the `/uploads` folder on the host.
  Returns a JSON with the fields `output` (containing the file's name on the server) and
  `error` containing any potential server side errors.

If `--tls-cert-file` is added (ideally in conjunction with `--tls-private-key-file`, the HTTP server
will be upgraded to HTTPS. The image has default, `localhost`-based cert/privkey files at
`/localhost.crt` and `/localhost.key` (see: [`porter` subcommand](#porter))

If `--http-override` is set, the HTTP(S) server will always serve the override path & options,
ignoring the request URL.

It will also start a UDP server on the indicated UDP port that responds to the following commands:

- `hostname`: Returns the server's hostname
- `echo <msg>`: Returns the given `<msg>`
- `clientip`: Returns the request's IP address

The UDP server can be disabled by setting `--udp-port -1`.

Additionally, if (and only if) `--sctp-port` is passed, it will start an SCTP server on that port,
responding to the same commands as the UDP server.

Usage:

```console
    kubectl exec test-agnhost -- /agnhost netexec [--http-port <http-port>] [--udp-port <udp-port>] [--sctp-port <sctp-port>] [--tls-cert-file <cert-file>] [--tls-private-key-file <privkey-file>]
```
