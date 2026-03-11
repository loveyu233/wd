package wd

import (
	"os"
	"os/signal"
	"sync"
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

	Trigger()

	Wait() <-chan struct{}
}

type SignalHook struct {
	ctx        chan os.Signal
	done       chan struct{}
	CloseFuncs []func()
	mu         sync.RWMutex
	once       sync.Once
	runOnce    sync.Once
}

func (h *SignalHook) AppendFun(funcs ...func()) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.CloseFuncs = append(h.CloseFuncs, funcs...)
}

var InsGlobalHook Hook

func init() {
	hook := &SignalHook{
		ctx:  make(chan os.Signal, 1),
		done: make(chan struct{}),
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

func (h *SignalHook) Wait() <-chan struct{} {
	h.once.Do(func() {
		go h.listen()
	})
	return h.done
}

func (h *SignalHook) listen() {
	<-h.ctx
	h.Trigger()
}

func (h *SignalHook) Trigger() {
	h.runOnce.Do(func() {
		signal.Stop(h.ctx)

		h.mu.RLock()
		funcs := append([]func(){}, h.CloseFuncs...)
		h.mu.RUnlock()

		for _, f := range funcs {
			if f == nil {
				continue
			}
			f()
		}
		close(h.done)
	})
}

// Close 用来在收到信号后执行注册的清理函数。
func (h *SignalHook) Close() {
	<-h.Wait()
}
