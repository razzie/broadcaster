package broadcaster

import (
	"context"
	"time"
)

var (
	defaultBroadcasterOptions = broadcasterOptions{
		timeout: -1,
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
