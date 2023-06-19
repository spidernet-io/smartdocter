[![codecov](https://codecov.io/github/fortio/log/branch/main/graph/badge.svg?token=LONYZDFQ7C)](https://codecov.io/github/fortio/log)

# Log

Fortio's log is simple logger built on top of go's default one with
additional opinionated levels similar to glog but simpler to use and configure.

It's been used for many years for Fortio's org Fortio project and more (under fortio.org/fortio/log package) but split out recently for standalone use, with the "flag polution" limited (as a library it doesn't include the flags, you configure it using apis).

```golang
// On a cli tool (avoids file name and line numbers, stack traces on log.Fatalf etc)
log.SetDefaultsForClientTools()
log.LoggerStaticFlagSetup() // adds -loglevel flag to configure
// Or on a server type, import fortio.org/dflag, then:
dflag.LoggerFlagSetup()

// Then, printf style leveled logging:
log.Debugf(...) // Debug level
log.LogVf(...)  // Verbose level
log.Infof(...)  // Info/default level
log.Warnf(...)  // Warning level
log.Errf(...)   // Error level
log.Critf(...)  // Critical level (always logged even if level is set to max)
log.Fatalf(...) // Fatal level - program will panic/exit

// for http servers there is also
// access log type including user-agent, forwarded ip/proto (behind load balancer case),
// TLS crypto used and CN of peer certificate if any.
log.LogRequest(r, "some info")
```

See the `Config` object for options like whether to include line number and file name of caller or not etc

New since 1.4 server logging (as used in [fortio.org/scli](https://pkg.go.dev/fortio.org/scli#ServerMain) for instance) is now structured (json), client logging (as setup by [fortio.org/cli](https://pkg.go.dev/fortio.org/scli#ServerMain) remains as before.

One can also revert server to not be JSON through config.

In JSON mode the output looks like this
```json
{"ts":1683504169239557,"level":"info","file":"logger.go","line":221,"msg":"Log level is now 1 Verbose (was 2 Info"}
```
Which can be converted to JSONEntry but is also a fixed, optimized format (ie ts is always first etc)

The timestamp `ts` is in microseconds since epoch (golang UnixMicro())

Optional additional `KeyValue` pairs can be added to the base structure using the new `log.S` or passed to `log.LogRequest`.
