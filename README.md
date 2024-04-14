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
	Listen(opts ...ListenerOption) (<-chan T, CancelFunc, error)
	IsClosed() bool
	Done() <-chan struct{}
}

func NewBroadcaster[T any](input <-chan T, opts ...BroadcasterOption) Broadcaster[T]
```

### Broadcaster options
```go
WithTimeout(timeout time.Duration)
WithListenerBufferSize(bufSize int)
WithBlocking(blocking bool)
WithIdleTimeout(timeout time.Duration)
```

### Listener options
```go
WithBufferSize(bufSize int)
WithContext(ctx context.Context)
WithTimeoutCallback(func())
```

### Server Sent Events
```go
type Marshaler func(any) ([]byte, error)

func NewEventSource[T any](input <-chan T, eventName string, marshaler Marshaler) EventSource
func NewJsonEventSource[T any](input <-chan T, eventName string) EventSource
func NewTextEventSource(input <-chan string, eventName string) EventSource
func NewTemplateEventSource[T any](input <-chan T, eventName string, t *template.Template, templateName string) EventSource
func BundleEventSources(srcs ...EventSource) EventSource

func NewSSEBroadcaster(src EventSource, opts ...BroadcasterOption) http.Handler
```
* `Marshaler` type is compatible with `json.Marshal` (which is used by default in case `marshaler` is left `nil`).
* `eventName` can be an empty string.

### On-demand broadcasters
```go
type Source[T any] func() (<-chan T, error)
func NewOndemandBroadcaster[T any](src Source[T], opts ...BroadcasterOption) Broadcaster[T]

type OndemandEventSource func() (EventSource, error)
func NewOndemandSSEBroadcaster(src OndemandEventSource, opts ...BroadcasterOption) http.Handler
```

### Converter broadcasters
```go
type Converter[In, Out any] func(In) (Out, bool)

func NewConverterBroadcaster[In, Out any](input <-chan In, convert Converter[In, Out], opts ...BroadcasterOption) Broadcaster[Out]
func NewOndemandConverterBroadcaster[In, Out any](src Source[In], convert Converter[In, Out], opts ...BroadcasterOption) Broadcaster[Out]
```
