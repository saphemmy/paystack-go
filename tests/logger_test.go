package paystack_test

import (
	"bytes"
	"fmt"
	"testing"

	paystack "github.com/saphemmy/paystack-go"
)

type recordingLogger struct {
	buf bytes.Buffer
}

func (r *recordingLogger) Printf(format string, args ...interface{}) {
	fmt.Fprintf(&r.buf, format, args...)
}

type recordingLeveled struct {
	debug, info, warn, errs bytes.Buffer
}

func (r *recordingLeveled) Debugf(format string, args ...interface{}) {
	fmt.Fprintf(&r.debug, format, args...)
}
func (r *recordingLeveled) Infof(format string, args ...interface{}) {
	fmt.Fprintf(&r.info, format, args...)
}
func (r *recordingLeveled) Warnf(format string, args ...interface{}) {
	fmt.Fprintf(&r.warn, format, args...)
}
func (r *recordingLeveled) Errorf(format string, args ...interface{}) {
	fmt.Fprintf(&r.errs, format, args...)
}

func TestLogger_InterfaceIsSatisfied(t *testing.T) {
	var _ paystack.Logger = (*recordingLogger)(nil)
	var _ paystack.LeveledLogger = (*recordingLeveled)(nil)
}

func TestLogger_PrintfRoutesArgs(t *testing.T) {
	l := &recordingLogger{}
	l.Printf("hello %s=%d", "x", 7)
	if got := l.buf.String(); got != "hello x=7" {
		t.Fatalf("unexpected log output %q", got)
	}
}

func TestLeveledLogger_AllLevelsRoute(t *testing.T) {
	l := &recordingLeveled{}
	l.Debugf("d%d", 1)
	l.Infof("i%d", 2)
	l.Warnf("w%d", 3)
	l.Errorf("e%d", 4)
	if l.debug.String() != "d1" {
		t.Fatalf("debug=%q", l.debug.String())
	}
	if l.info.String() != "i2" {
		t.Fatalf("info=%q", l.info.String())
	}
	if l.warn.String() != "w3" {
		t.Fatalf("warn=%q", l.warn.String())
	}
	if l.errs.String() != "e4" {
		t.Fatalf("error=%q", l.errs.String())
	}
}
