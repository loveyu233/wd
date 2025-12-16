package wd

import (
	"os"
	"os/signal"
	"syscall"
)

var _ Hook = (*SignalHook)(nil)

// Hook a graceful shutdown hook, default with signals of SIGINT and SIGTERM
type Hook interface {
	// WithSignals add more signals into hook
	WithSignals(signals ...syscall.Signal) Hook

	// Close register shutdown handles
	Close()

	AppendFun(funcs ...func())
}

type SignalHook struct {
	ctx        chan os.Signal
	CloseFuncs []func()
}

func (h *SignalHook) AppendFun(funcs ...func()) {
	h.CloseFuncs = append(h.CloseFuncs, funcs...)
}

var InsGlobalHook Hook

func init() {
	hook := &SignalHook{
		ctx: make(chan os.Signal, 1),
	}

	InsGlobalHook = hook.WithSignals(syscall.SIGINT, syscall.SIGTERM)
}

// WithSignals 用来为钩子追加需要监听的系统信号。
func (h *SignalHook) WithSignals(signals ...syscall.Signal) Hook {
	for _, s := range signals {
		signal.Notify(h.ctx, s)
	}

	return h
}

// Close 用来在收到信号后执行注册的清理函数。
func (h *SignalHook) Close() {
	select {
	case <-h.ctx:
	}
	signal.Stop(h.ctx)

	for _, f := range h.CloseFuncs {
		f()
	}
}
