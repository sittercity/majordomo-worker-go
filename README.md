## Majordomo Worker [![Build Status](https://travis-ci.org/sittercity/majordomo-worker-go.svg?branch=master)](https://travis-ci.org/sittercity/majordomo-worker-go)

Majordomo worker implementation. Written in Go.

This package is not intended to be used on its own. It only handles the Majordomo protocol as a worker.
You must write worker-specific logic based upon the usage information below in order for your work
to be processed.

[Majordomo protocol RFC](http://rfc.zeromq.org/spec:7)

## Usage

The Majordomo worker requires a 'worker action' that matches the following interface:

```go
type WorkerAction interface {
	Call([][]byte) [][]byte
}
```

You *must* provide an action for the worker to perform. You can have the action do whatever you want. You are responsible for handling all input and output. This package will handle all communication to and from the majordomo broker.

In addition, the Majordomo worker requires a logger that conforms to the [GoKit Logger](https://github.com/go-kit/kit/tree/master/log) interface.

To create a worker:

```go
workerConfig := majordomo_worker.WorkerConfig{
  BrokerAddress: "tcp://broker-address", // See below for rules governing broker addresses
  ServiceName: "service-name", // Unique, abstract service name for your client/worker pair
  HeartbeatInMillis: 1000*time.Millisecond, // time to wait between heartbeats
  ReconnectInMillis: 1000*time.Millisecond, // time to sleep before reconnecting
  PollingInterval: 500*time.Millisecond, // polling interval. This is how often we check the ZeroMQ socket
  MaxHeartbeatLiveness: 50, // max 'aliveness' count. This is the number of times we try to poll before deciding that the broker is dead if we haven't heard anything
  Action: action, // an 'action' that matches the interface above
}

logger := ...<create your logger that matches the GoKit Logger interface>...

worker := majordomo_worker.NewWorker(logger, workerConfig)
```

You can then call the following:

```go
worker.Receive()
```

This will run until it encounters an unrecoverable error (i.e. ZeroMQ issue) or until a 'shutdown' is called on the worker.

You are responsible for managing any interrupts and calling the 'Shutdown()' method as appropriate. Example:

```go
  w := worker.NewWorker(...)

  // Handle shutdown signals
  intCh := make(chan os.Signal, 1)
  signal.Notify(intCh, syscall.SIGINT, syscall.SIGTERM)
  go func() {
    <-intCh
    w.Shutdown()
  }()

  for {
    w.Receive()
  }
```

In the above example if the process receives a SIGTERM or SIGINT it will initiate a shutdown that will gracefully close the worker after the current request work is completed.

### Broker addresses

It is possible to pass multiple broker addresses for workers to use. You *must* use the following format:

```
tcp://broker-address1,tcp://broker-address2
```

## Test

Right now tests are a little unoptimized. Tests could take up to 20 seconds due to various
heartbeating and reconnection settings. This is a target area for improvement.

We also are not at 100% coverage. PRs welcome!

```sh
make test
```

## Contributing

Simply open a PR on your own fork to add the functionality you desire. As long as you have new tests to cover your new work then we'll be happy!
