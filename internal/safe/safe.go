// Package safe runs work that must not take the process down with it.
//
// The Go runtime kills the whole process when any goroutine panics, and the
// HTTP server only recovers panics on the goroutines it starts itself. This
// application starts its own — invoice and reminder notifications, OCR,
// chatbot replies, WhatsApp blasts, expiry sweepers — so a nil map write in one
// best-effort notification would otherwise end the service for every user.
//
// Every such goroutine goes through here, so the cost of a panic in background
// work is one lost job plus a log line with the stack that caused it.
package safe

import (
	"log"
	"runtime/debug"
	"time"
)

// Go runs fn on a new goroutine, recovering and logging any panic it raises.
// label names the work in the log line ("kirim WA invoice"), so a report says
// what was running without having to read the stack.
func Go(label string, fn func()) {
	go Recovered(label, fn)()
}

// Every runs fn on its own goroutine once per interval, for the life of the
// process. It is the shape every in-memory store here uses to evict what has
// expired — rate-limiter buckets, password-reset tokens — so those stores stay
// bounded however long the process runs.
//
// There is no way to stop it: the callers are process-wide singletons, and a
// stop channel nobody closes is worse than none. A panic in one tick is
// recovered and logged, and the next tick still comes.
func Every(label string, interval time.Duration, fn func()) {
	tick := Recovered(label, fn)
	Go(label, func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for range ticker.C {
			tick()
		}
	})
}

// Recovered wraps fn so a panic inside it is logged instead of unwinding into
// the runtime. Use it where something else already owns the goroutine — a
// scheduler task, say — and Go would start a second one behind its back.
func Recovered(label string, fn func()) func() {
	return func() {
		defer func() {
			if v := recover(); v != nil {
				LogPanic(label, v)
			}
		}()
		fn()
	}
}

// LogPanic reports a recovered panic together with the stack that raised it.
//
// Call it from the deferred function that recovered, while the panicking frames
// are still on the stack — otherwise the trace shows where the report was made
// rather than where the panic came from. Callers that only need the log line
// should use Go or Recovered instead of recovering by hand; this is exported
// for the ones that must also act on the failure, such as the HTTP middleware
// that has to answer the request with a 500.
func LogPanic(label string, v any) {
	log.Printf("panic recovered [%s]: %v\n%s", label, v, debug.Stack())
}
