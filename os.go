package main

import (
	"os"
	"syscall"
)

// SignalHandler represents a func that can handle a signal
type SignalHandler func(s os.Signal)

// TermSignalHandler returns a SignalHandler that is executed only on a term signal
func TermSignalHandler(f func()) SignalHandler {
	return func(s os.Signal) {
		if isTermSignal(s) {
			f()
		}
	}
}

func isTermSignal(s os.Signal) bool {
	return s == syscall.SIGKILL || s == syscall.SIGINT || s == syscall.SIGQUIT || s == syscall.SIGTERM
}

// LoggerSignalHandler returns a SignalHandler that logs the signal
func LoggerSignalHandler(l SeverityLogger, ignoredSignals ...os.Signal) SignalHandler {
	ss := make(map[os.Signal]bool)
	for _, s := range ignoredSignals {
		ss[s] = true
	}
	return func(s os.Signal) {
		if _, ok := ss[s]; ok {
			return
		}
		l.Debugf("jlskit: received signal %s", s)
	}
}
