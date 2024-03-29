# github.com/razzie/broadcaster

## Summary
The scope of this library is to take an input Go channel (producer) and broadcast the objects sent over it to an arbitrary amount of listeners (consumers).

Listeners are channels of the same type as the input channel. They can be opened and closed at any time and these operations block until the listener is opened/closed.

An incoming object from the input channel is always sent to all current listeners before the next object is consumed. If there are no listeners, the incoming object is dropped, unless the broadcaster is set to blocking mode. It is possible to configure a timeout so blocking listeners won't clog the broadcaster.

Closing the input channel closes all listeners and subsequent calls to ``Listen()`` return ``ErrBroadcasterClosed`` error.

### Server Sent Events
As an extra feature, the library supports the creation of a http.Handler that broadcasts incoming objects from one or more input channels (event sources).

## API reference
### Broadcaster interface + instantiation
```go
type Broadcaster[T any] interface {
	Listen(opts ...ListenerOption) (ch <-chan T, cancel func(), err error)
	IsClosed() bool
}

func NewBroadcaster[T any](input <-chan T, opts ...BroadcasterOption) Broadcaster[T]
```

### Broadcaster options
```go
WithLogger(logger *slog.Logger)
WithTimeout(timeout time.Duration)
WithListenerBufferSize(bufSize int)
WithBlocking(blocking bool)
```

### Listener options
```go
WithBufferSize(bufSize int)
WithContext(ctx context.Context)
```

### Server Sent Events
```go
func NewEventSource[T any](input <-chan T, eventName string, marshaler Marshaler) EventSource

func BundleEventSources(srcs ...EventSource) EventSource

func NewSSEBroadcaster(src EventSource, opts ...BroadcasterOption) http.Handler
```

`Marshaler` type is compatible with `json.Marshal` (which is used by default in case `marshaler` is left `nil`).

`eventName` can be an empty string.
