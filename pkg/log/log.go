package log

import (
	"context"
	"errors"
	"flag"
	"io"
	stdlog "log"
	"os"
	"strings"
	"time"

	"github.com/go-logr/logr"
	"github.com/go-logr/zerologr"
	"github.com/rs/zerolog"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	JSONFormat = "json"
	TextFormat = "text"

	LogLevel = 1

	DefaultTime = "default"
	ISO8601     = "iso8601"
	RFC3339     = "rfc3339"
	MILLIS      = "millis"
	NANOS       = "nanos"
	EPOCH       = "epoch"
	RFC3339NANO = "rfc3339nano"
)

var globalLog = log.Log // null log sink when we don't SetLogger

func InitFlags(flags *flag.FlagSet) {
	if flag.CommandLine.Lookup("log_dir") != nil {
		flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	}
	klog.InitFlags(flags)
}

func Setup(logFormat string, loggingTimestampFormat string, level int, disableColor bool) error {
	zerologr.SetMaxV(level)

	var logger zerolog.Logger
	switch logFormat {
	case TextFormat:
		output := zerolog.ConsoleWriter{Out: os.Stderr, NoColor: disableColor}
		output.TimeFormat = resolveTimestampFormat(loggingTimestampFormat)
		logger = zerolog.New(output).With().Timestamp().Caller().Logger()
	case JSONFormat:
		logger = zerolog.New(os.Stderr).With().Timestamp().Logger()
	default:
		return errors.New("unrecognized log format (available: `text`, `json`)")
	}
	globalLog = zerologr.New(&logger)
	klog.SetLogger(globalLog.WithName("klog"))
	log.SetLogger(globalLog)
	return nil
}

func resolveTimestampFormat(format string) string {
	switch format {
	case ISO8601:
		return time.RFC3339
	case RFC3339:
		return time.RFC3339
	case MILLIS:
		return time.StampMilli
	case NANOS:
		return time.StampNano
	case EPOCH:
		return time.UnixDate
	case RFC3339NANO:
		return time.RFC3339Nano
	case DefaultTime:
		return time.RFC3339
	default:
		return time.RFC3339
	}
}

func GlobalLogger() logr.Logger {
	return globalLog
}

func Logger(name string) logr.Logger {
	return globalLog.WithName(name).V(LogLevel)
}

func WithName(name string) logr.Logger {
	return GlobalLogger().WithName(name)
}

func WithValues(kv ...interface{}) logr.Logger {
	return GlobalLogger().WithValues(kv...)
}

func V(level int) logr.Logger {
	return GlobalLogger().V(level)
}

func Info(msg string, kv ...interface{}) {
	GlobalLogger().WithCallDepth(1).Info(msg, kv...)
}

func Error(err error, msg string, kv ...interface{}) {
	GlobalLogger().WithCallDepth(1).Error(err, msg, kv...)
}

func FromContext(ctx context.Context, kv ...interface{}) (logr.Logger, error) {
	logger, err := logr.FromContext(ctx)
	if err != nil {
		return logger, err
	}

	return logger.WithValues(kv...), nil
}

func IntoContext(ctx context.Context, log logr.Logger) context.Context {
	return logr.NewContext(ctx, log)
}

func IntoBackground(log logr.Logger) context.Context {
	return IntoContext(context.Background(), log)
}

func IntoTODO(log logr.Logger) context.Context {
	return IntoContext(context.TODO(), log)
}

type writerAdapter struct {
	io.Writer
	logger logr.Logger
}

func (a *writerAdapter) Write(p []byte) (int, error) {
	a.logger.Info(strings.TrimSuffix(string(p), "\n"))
	return len(p), nil
}

func StdLogger(logger logr.Logger, prefix string) *stdlog.Logger {
	return stdlog.New(&writerAdapter{logger: logger}, prefix, stdlog.LstdFlags)
}
