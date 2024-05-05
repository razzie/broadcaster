package broadcaster

import (
	"context"
	"io"
	"net/http"
	"time"
)

var (
	defaultBroadcasterOptions = broadcasterOptions{
		timeout: -1,
	}

	defaultSSEListenerOptions = sseListenerOptions{
		client: http.DefaultClient,
		method: "GET",
	}
)

type broadcasterOptions struct {
	timeout     time.Duration
	lisBufSize  int
	blocking    bool
	idleTimeout time.Duration
}

type BroadcasterOption func(*broadcasterOptions)

func WithTimeout(timeout time.Duration) BroadcasterOption {
	return func(bo *broadcasterOptions) {
		bo.timeout = timeout
	}
}

func WithListenerBufferSize(bufSize int) BroadcasterOption {
	return func(bo *broadcasterOptions) {
		bo.lisBufSize = bufSize
	}
}

func WithBlocking(blocking bool) BroadcasterOption {
	return func(bo *broadcasterOptions) {
		bo.blocking = blocking
	}
}

func WithIdleTimeout(timeout time.Duration) BroadcasterOption {
	return func(bo *broadcasterOptions) {
		bo.idleTimeout = timeout
	}
}

type listenerOptions struct {
	ctx       context.Context
	onTimeout func()
	bufSize   int
}

type ListenerOption func(*listenerOptions)

func WithBufferSize(bufSize int) ListenerOption {
	return func(lo *listenerOptions) {
		lo.bufSize = bufSize
	}
}

func WithContext(ctx context.Context) ListenerOption {
	return func(lo *listenerOptions) {
		lo.ctx = ctx
	}
}

func WithTimeoutCallback(onTimeout func()) ListenerOption {
	return func(lo *listenerOptions) {
		lo.onTimeout = onTimeout
	}
}

type sseListenerOptions struct {
	client          *http.Client
	method          string
	body            io.Reader
	bodyContentType string
	header          http.Header
	bufSize         int
}

type SSEListenerOption func(*sseListenerOptions)

func WithClient(client *http.Client) SSEListenerOption {
	return func(slo *sseListenerOptions) {
		slo.client = client
	}
}

func WithMethod(method string) SSEListenerOption {
	return func(slo *sseListenerOptions) {
		slo.method = method
	}
}

func WithBody(body io.Reader, contentType string) SSEListenerOption {
	return func(slo *sseListenerOptions) {
		slo.body = body
		slo.bodyContentType = contentType
	}
}

func WithHeader(key, value0 string, values ...string) SSEListenerOption {
	return func(slo *sseListenerOptions) {
		if slo.header == nil {
			slo.header = make(http.Header)
		}
		slo.header.Set(key, value0)
		for _, value := range values {
			slo.header.Add(key, value)
		}
	}
}

func WithEventsBufferSize(bufSize int) SSEListenerOption {
	return func(slo *sseListenerOptions) {
		slo.bufSize = bufSize
	}
}
