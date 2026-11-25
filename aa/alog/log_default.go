package alog

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/aarioai/aago/aa/acontext"
	"github.com/aarioai/aago/pkg/utils"

	"github.com/aarioai/aago/aa/ae"
)

type xlog struct {
	Level ErrorLevel
}

func NewDefaultLog(level ErrorLevel) LogInterface {
	return &xlog{Level: level}
}
func packMessage(ctx context.Context, level ErrorLevel, caller, msg string) string {
	traceInfo := acontext.TraceInfo(ctx)
	b := strings.Builder{}
	b.Grow(15 + len(traceInfo))

	if level != LevelErrAll {
		b.WriteString("[")
		b.WriteString(level.Name())
		b.WriteString("] ")
	}

	b.WriteString(caller)
	b.WriteByte(' ')
	b.WriteString(msg)

	b.WriteString(traceInfo)
	return b.String()
}

func (l *xlog) print(ctx context.Context, level ErrorLevel, caller string, msg string) {
	if level > l.Level {
		return
	}
	log.Println(packMessage(ctx, level, caller, msg))
}

// log drops disabled levels before paying for the caller lookup.
// utils.Caller skips aago frames, so the extra frame here does not change the reported caller.
func (l *xlog) log(ctx context.Context, level ErrorLevel, msg string) {
	if level > l.Level {
		return
	}
	l.print(ctx, level, utils.Caller(2), msg)
}
func (l *xlog) New(prefix string, f func(context.Context, string, ...any), suffix ...string) func(context.Context, string, ...any) {
	var s string
	if len(suffix) > 0 {
		s = " " + suffix[0]
	}
	if prefix != "" {
		prefix += " "
	}
	return func(ctx context.Context, msg string, args ...any) {
		f(ctx, prefix+msg+s, args...)
	}
}
func (l *xlog) Debug(ctx context.Context, msg string) {
	l.log(ctx, LevelDebug, msg)
}
func (l *xlog) Debugf(ctx context.Context, msg string, args ...any) {
	if LevelDebug > l.Level {
		return
	}
	l.log(ctx, LevelDebug, fmt.Sprintf(msg, args...))
}

func (l *xlog) Info(ctx context.Context, msg string) {
	l.log(ctx, LevelInfo, msg)
}

func (l *xlog) Infof(ctx context.Context, format string, args ...any) {
	if LevelInfo > l.Level {
		return
	}
	l.log(ctx, LevelInfo, fmt.Sprintf(format, args...))
}
func (l *xlog) Notice(ctx context.Context, msg string) {
	l.log(ctx, LevelNotice, msg)
}
func (l *xlog) Noticef(ctx context.Context, format string, args ...any) {
	if LevelNotice > l.Level {
		return
	}
	l.log(ctx, LevelNotice, fmt.Sprintf(format, args...))
}
func (l *xlog) Warn(ctx context.Context, format string) {
	l.log(ctx, LevelWarn, format)
}
func (l *xlog) Warnf(ctx context.Context, format string, args ...any) {
	if LevelWarn > l.Level {
		return
	}
	l.log(ctx, LevelWarn, fmt.Sprintf(format, args...))
}

func (l *xlog) Error(ctx context.Context, msg string) {
	l.log(ctx, LevelError, msg)
}
func (l *xlog) Errorf(ctx context.Context, format string, args ...any) {
	if LevelError > l.Level {
		return
	}
	l.log(ctx, LevelError, fmt.Sprintf(format, args...))
}
func (l *xlog) E(ctx context.Context, e *ae.Error, msg ...any) {
	if e == nil || LevelError > l.Level {
		return
	}
	s := e.String()
	if len(msg) > 0 {
		s = fmt.Sprint(msg...)
	}
	l.print(ctx, LevelError, e.Caller, s)
}

func (l *xlog) Fatal(ctx context.Context, msg string) {
	l.log(ctx, LevelFatal, msg)
}

func (l *xlog) Fatalf(ctx context.Context, format string, args ...any) {
	if LevelFatal > l.Level {
		return
	}
	l.log(ctx, LevelFatal, fmt.Sprintf(format, args...))
}

func (l *xlog) Alert(ctx context.Context, msg string) {
	l.log(ctx, LevelAlert, msg)
}

func (l *xlog) Alertf(ctx context.Context, format string, args ...any) {
	if LevelAlert > l.Level {
		return
	}
	l.log(ctx, LevelAlert, fmt.Sprintf(format, args...))
}

func (l *xlog) Emerg(ctx context.Context, msg string) {
	l.log(ctx, LevelEmerg, msg)
}

func (l *xlog) Emergf(ctx context.Context, format string, args ...any) {
	if LevelEmerg > l.Level {
		return
	}
	l.log(ctx, LevelEmerg, fmt.Sprintf(format, args...))
}

func (l *xlog) Print(ctx context.Context, msg string) {
	log.Println(packMessage(ctx, LevelErrAll, utils.Caller(1), msg))
}

func (l *xlog) Printf(ctx context.Context, format string, args ...any) {
	log.Println(packMessage(ctx, LevelErrAll, utils.Caller(1), fmt.Sprintf(format, args...)))
}

func (l *xlog) Trace(ctx context.Context) {
	traceInfo := acontext.TraceInfo(ctx)
	log.Printf("[TRACE]%s %s\n", traceInfo, utils.Caller(1))
}
