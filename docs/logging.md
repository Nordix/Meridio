# Meridio - logging

Meridio uses structured logging implemented by
[go-logr/logr](https://github.com/go-logr/logr). Structured logging
means that printf-like formatted messages are not used, instead
key/object pairs are passed to the log functions.

```go
import "github.com/nordix/meridio/pkg/log"

	var config Config
	err := envconfig.Process("ipam", &config)
	if err != nil {
		panic(err)  // We can't log since we have no logger yet
	}
	logger := log.New("Meridio-ipam", config.LogLevel)
	logger.Info("Configuration read", "config", config)
```

When executed this will produce (formatted with
[jq](https://stedolan.github.io/jq/));

```json
{
  "severity": "info",
  "timestamp": "2022-08-31T09:04:03.482+00:00",
  "service_id": "Meridio-ipam",
  "message": "Configuration read",
  "version": "1.0.0",
  "extra_data": {
    "config": {
      "Port": 7777,
      "Datasource": "/run/ipam/data/registry.db",
      "TrenchName": "red",
      "NSPService": "meridio-nsp-red:7778",
      "PrefixIPv4": "172.16.0.0/16",
      "ConduitPrefixLengthIPv4": 20,
      "NodePrefixLengthIPv4": 24,
      "PrefixIPv6": "fd00::172.16.0.0/112",
      "ConduitPrefixLengthIPv6": 116,
      "NodePrefixLengthIPv6": 120,
      "IPFamily": "dualstack",
      "LogLevel": "DEBUG"
    }
  }
}
```

Structured logs can be scanned with [jq](https://stedolan.github.io/jq/).

```
kubectl logs -n red meridio-load-balancer-6dbbb9556f-f5cc4 -c load-balancer \
  | grep '^{' | jq 'select(.extra_data.class == "SimpleNetworkService")'
kubectl logs -n red meridio-load-balancer-6dbbb9556f-f5cc4 -c load-balancer \
  | grep '^{' | jq 'select(.extra_data.class == "SimpleNetworkService")|select(.message == "updateVips")'

```

## Logger from context

A logger should be created in `main()` and be used for logging
everywhere. The logger is not passed in every call but a
[go context](https://pkg.go.dev/context) should. Functions should
use the logger from the context;

```go
// In main();
ctx = logr.NewContext(ctx, logger)
// In a function;
logger = log.FromContextOrGlobal(ctx)
```

Functions really should always have a context as first parameter but
they might not. A global logger is provided;

```
log.Logger.Info("Configuration read", "config", config)
```

The global logger is set by the *first* call to `log.New`. A global logger
named "Meridio" on INFO level is pre-installed before `log.New` is called.



## Log levels

Severity `debug`, `info`, `error` and `critical` are used (not
`warning`). The `Info()` call can have different "verbosity", set with the
`V(n)` method;

```go
logger.Info("This is a normal info message")
logger.V(1).Info("This is a debug message")
logger.V(2).Info("This is a trace message")
```

There is no defined "trace" level in output so both trace and debug
messages will have severity "debug". The level filtering is still valid
though, trace messages are suppressed unless TRACE level is set.

The `Fatal()` function logs on `critical` level.

### Costly parameter computations

**This is important!**

Consider;

```go
logger.V(2).Info("Gathered data", "collected", verySlowFunction())
```

The `verySlowFunction()` will *always* be executed, even if not on
`trace` level. A few of these may have a severe impact on
performance but you may *really* want them for trace. Luckily there is
a [trick](https://github.com/go-logr/logr/issues/149);

```
 if loggerV := logger.V(2); loggerV.Enabled() {
   loggerV.Info("Gathered data", "collected", verySlowFunction())
 }
```

Now the `verySlowFunction()` is *only* executed when trace level is set.


## Fatal

```go
import "github.com/nordix/meridio/pkg/log"
	logger := log.New("Meridio-ipam", config.LogLevel)
	log.Fatal(logger, "Can't read crucial data", "error", err)
```

The logger is a pure `logr.Logger` logger so there is no `Fatal()`
method. However we want to print a termination message using the same
formatting as other log items so the `logger` is passed as a parameter.

Example output;
```json
{
  "severity": "critical",
  "timestamp": "2022-08-31T13:42:29.345+02:00",
  "service_id": "Meridio-test",
  "message": "Can't read crucial data",
  "version": "1.1.0",
  "extra_data": {
    "error": "Not found"
  }
}
```


## Design patterns

Patterns must evolve slowly to get really good so these are mere
ideas. It is very easy to get carried away and impose some
over-structured logging that floods the logs with useless data.


### Class logger

A logger used in a type (Class) can be decorated with `class` and
`instance` records;

```go
type someHandler struct {
	ctx    context.Context
	logger logr.Logger
	Adr    *net.TCPAddr // (optional; capitalized to make it visible)
}

func newHandler(ctx context.Context, addr string) *someHandler {
	logger := log.FromContextOrGlobal(ctx).WithValues("class", "someHandler")
	adr, err := net.ResolveTCPAddr("tcp", addr)
	if err != nil {
		log.Fatal(logger, "ResolveTCPAddr", "error", err)
	}
	h := &someHandler{
		ctx:    ctx,
		logger: logger.WithValues("instance", adr),
		Adr:    adr,
	}
	h.logger.Info("Created", "object", h)
	return h
}

func (h *someHandler) connect() error {
	logger := h.logger.WithValues("func", "connect")
	logger.Info("Called")
	return nil
}
```

The `class` is the name of the type and `instance` can be anything
that identifies an instance. The instance field must be
capitalized if you want it visible.

The example shows a `func` entry to identify a function. This should
*not* be used as a common pattern but may be handy in some cases.

