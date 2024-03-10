package broadcaster

import (
	"context"
	"log/slog"
	"time"
)

var (
	defaultBroadcasterOptions = broadcasterOptions{
		logger:  slog.Default(),
		timeout: -1,
	}
)

type broadcasterOptions struct {
	logger     *slog.Logger
	timeout    time.Duration
	lisBufSize int
	blocking   bool
}

type BroadcasterOption func(*broadcasterOptions)

func WithLogger(logger *slog.Logger) BroadcasterOption {
	return func(bo *broadcasterOptions) {
		bo.logger = logger
	}
}

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

type listenerOptions struct {
	bufSize int
	ctx     context.Context
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
