package safe

import (
	"bytes"
	"log"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// logCapture collects what the standard logger writes during one test. It
// signals every write, because a panic on another goroutine is reported after
// that goroutine's own defers have run — waiting for the work to finish is not
// enough to have seen the report.
type logCapture struct {
	mu      sync.Mutex
	buf     bytes.Buffer
	written chan struct{}
}

func (c *logCapture) Write(p []byte) (int, error) {
	c.mu.Lock()
	n, err := c.buf.Write(p)
	c.mu.Unlock()
	select {
	case c.written <- struct{}{}:
	default:
	}
	return n, err
}

// text returns everything logged so far.
func (c *logCapture) text() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.buf.String()
}

// awaitLine blocks until something is logged, so a test can read text() without
// racing the goroutine that produces it.
func (c *logCapture) awaitLine(t *testing.T) string {
	t.Helper()
	select {
	case <-c.written:
	case <-time.After(2 * time.Second):
		t.Fatal("nothing was logged within 2s")
	}
	return c.text()
}

// captureLog redirects the standard logger for the duration of one test.
func captureLog(t *testing.T) *logCapture {
	t.Helper()
	c := &logCapture{written: make(chan struct{}, 1)}
	prevWriter, prevFlags := log.Writer(), log.Flags()
	log.SetOutput(c)
	log.SetFlags(0)
	t.Cleanup(func() {
		log.SetOutput(prevWriter)
		log.SetFlags(prevFlags)
	})
	return c
}

func TestGoSurvivesPanic(t *testing.T) {
	c := captureLog(t)

	Go("kirim WA invoice", func() { panic("boom") })
	logged := c.awaitLine(t)

	if !strings.Contains(logged, "kirim WA invoice") {
		t.Errorf("log = %q, want the label of the work that panicked", logged)
	}
	if !strings.Contains(logged, "boom") {
		t.Errorf("log = %q, want the panic value", logged)
	}
	if !strings.Contains(logged, "safe_test.go") {
		t.Errorf("log = %q, want a stack trace pointing at the panicking call", logged)
	}
}

func TestGoRunsWorkAndLogsNothingWhenItSucceeds(t *testing.T) {
	c := captureLog(t)

	done := make(chan struct{})
	Go("pekerjaan sehat", func() { close(done) })
	<-done

	if logged := c.text(); logged != "" {
		t.Errorf("log = %q, want nothing for work that did not panic", logged)
	}
}

func TestRecoveredContainsThePanicInTheCallersGoroutine(t *testing.T) {
	c := captureLog(t)

	ran := false
	// No goroutine of its own: the wrapper must swallow the panic right here,
	// which is what lets a scheduler task keep the recovery without the
	// scheduler losing track of a second goroutine.
	Recovered("job penjadwal", func() {
		ran = true
		panic("job meledak")
	})()

	if !ran {
		t.Fatal("wrapped function did not run")
	}
	logged := c.text()
	if !strings.Contains(logged, "job penjadwal") || !strings.Contains(logged, "job meledak") {
		t.Errorf("log = %q, want the label and the panic value", logged)
	}
}

func TestRecoveredPassesThroughANormalReturn(t *testing.T) {
	calls := 0
	wrapped := Recovered("pekerjaan sehat", func() { calls++ })
	wrapped()
	wrapped()
	if calls != 2 {
		t.Errorf("calls = %d, want 2", calls)
	}
}

func TestEveryKeepsTickingThroughAPanic(t *testing.T) {
	c := captureLog(t)

	var ticks atomic.Int32
	Every("penyapu uji", time.Millisecond, func() {
		if ticks.Add(1) == 1 {
			panic("tick pertama meledak")
		}
	})

	deadline := time.Now().Add(2 * time.Second)
	for ticks.Load() < 3 {
		if time.Now().After(deadline) {
			t.Fatalf("ticks = %d after 2s, want the loop to survive the panic and keep running", ticks.Load())
		}
		time.Sleep(time.Millisecond)
	}
	if logged := c.text(); !strings.Contains(logged, "tick pertama meledak") {
		t.Errorf("log = %q, want the panicking tick reported", logged)
	}
}

// A nil panic value is the one case recover() cannot tell apart from "no
// panic"; Go 1.21+ hands it over as a *runtime.PanicNilError, so it must still
// be reported rather than silently swallowed.
func TestGoReportsANilPanic(t *testing.T) {
	c := captureLog(t)

	Go("panic nil", func() { panic(nil) })
	logged := c.awaitLine(t)

	if !strings.Contains(logged, "panic nil") {
		t.Errorf("log = %q, want the panic reported", logged)
	}
}
