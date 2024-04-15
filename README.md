# github.com/razzie/broadcaster

## Summary
The scope of this library is to take an input Go channel (producer) and broadcast the objects sent over it to an arbitrary amount of listeners (consumers).

Listeners are channels of the same type as the input channel. They can be opened and closed at any time and these operations block until the listener is opened/closed.

An incoming object from the input channel is always sent to all current listeners before the next object is consumed. If there are no listeners, the incoming object is dropped, unless the broadcaster is set to blocking mode. It is possible to configure a timeout so blocking listeners won't clog the broadcaster.

Closing the input channel closes all listeners and subsequent calls to ``Listen()`` return ``ErrBroadcasterClosed`` error.

### Server Sent Events
As an extra feature, the library supports the creation of a http.Handler that broadcasts incoming objects from one or more input channels (event sources).

## Feature table
| Instantiation                            | Input type         | On-demand | Multi-source | Converter | SSE |
| ---------------------------------------- | ------------------ | --------- | ------------ | --------- | --- |
| NewBroadcaster[T]                        | <-chan T           | no        | no           | no        | no  |
| NewConverterBroadcaster[In, Out]         | <-chan In          | no        | no           | yes       | no  |
| NewOndemandBroadcaster[T]                | Source[T]          | yes       | no           | no        | no  |
| NewOndemandConverterBroadcaster[In, Out] | Source[In]         | yes       | no           | yes       | no  |
| NewMultisourceBroadcaster[K, T]          | MultiSource[K, T]  | yes       | yes          | no        | no  |
| NewMultiConverterBroadcaster[K, In, Out] | MultiSource[K, In] | yes       | yes          | yes       | no  |
| NewSSEBroadcaster                        | EventSource        | no        | no           | no        | yes |

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

### Multi-source broadcasters
```go
type MultiSource[K comparable, T any] func(K) (<-chan T, CancelFunc, error)

type MultiBroadcaster[K comparable, T any] interface {
	Listen(key K, opts ...ListenerOption) (<-chan T, CancelFunc, error)
}

func NewMultiBroadcaster[K comparable, T any](src MultiSource[K, T], opts ...BroadcasterOption) MultiBroadcaster[K, T]
```

### Converter broadcasters
```go
type Converter[In, Out any] func(In) (Out, bool)

func NewConverterBroadcaster[In, Out any](input <-chan In, convert Converter[In, Out], opts ...BroadcasterOption) Broadcaster[Out]
func NewOndemandConverterBroadcaster[In, Out any](src Source[In], convert Converter[In, Out], opts ...BroadcasterOption) Broadcaster[Out]
func NewMultiConverterBroadcaster[K comparable, In, Out any](src MultiSource[K, In], conv Converter[In, Out], opts ...BroadcasterOption) MultiBroadcaster[K, Out]
```
