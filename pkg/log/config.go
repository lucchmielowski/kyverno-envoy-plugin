package log

import (
	"os"
	"strings"
	"sync/atomic"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zapgrpc"
	"google.golang.org/grpc/grpclog"
	"k8s.io/klog/v2"
)

const (
	none          zapcore.Level = 100
	GrpcScopeName string        = "grpc"
)

var logLevelToZap = map[Level]zapcore.Level{
	DebugLevel: zapcore.DebugLevel,
	InfoLevel:  zapcore.InfoLevel,
	WarnLevel:  zapcore.WarnLevel,
	ErrorLevel: zapcore.ErrorLevel,
	FatalLevel: zapcore.FatalLevel,
	NoneLevel:  none,
}

var defaultEncoderConfig = zapcore.EncoderConfig{
	TimeKey:        "time",
	LevelKey:       "level",
	NameKey:        "scope",
	CallerKey:      "caller",
	MessageKey:     "msg",
	StacktraceKey:  "stack",
	LineEnding:     zapcore.DefaultLineEnding,
	EncodeLevel:    zapcore.LowercaseLevelEncoder,
	EncodeCaller:   zapcore.ShortCallerEncoder,
	EncodeDuration: zapcore.StringDurationEncoder,
	EncodeTime:     formatDate,
}

type patchTable struct {
	write       func(ent zapcore.Entry, fields []zapcore.Field) error
	sync        func() error
	exitProcess func(code int)
	errorSink   zapcore.WriteSyncer
	close       func() error
}

var (
	funcs = &atomic.Value{}

	// Control whether all output is JSON or CLI-like. Better than reading zap encoder internals
	useJSON atomic.Value
)

func init() {
	Configure(DefaultOptions())
}

func prepareZap(options *Options) (zapcore.Core, func(string) zapcore.Core, zapcore.WriteSyncer, error) {
	var enc zapcore.Encoder
	encCfg := defaultEncoderConfig

	if options.JSONEncoding {
		enc = zapcore.NewJSONEncoder(encCfg)
		useJSON.Store(true)
	} else {
		enc = zapcore.NewConsoleEncoder(encCfg)
		useJSON.Store(false)
	}

	errSink, closeErrorSink, err := zap.Open(options.ErrorOutputPaths...)
	if err != nil {
		return nil, nil, nil, err
	}

	var outputSink zapcore.WriteSyncer
	if len(options.OutputPaths) > 0 {
		outputSink, _, err = zap.Open(options.OutputPaths...)
		if err != nil {
			closeErrorSink()
			return nil, nil, nil, err
		}
	}

	alwaysOn := zapcore.NewCore(enc, outputSink, zap.NewAtomicLevelAt(zapcore.DebugLevel))
	conditionallyOn := func(scopeName string) zapcore.Core {
		scope := FindScope(scopeName)
		enabler := func(lvl zapcore.Level) bool {
			switch lvl {
			case zapcore.ErrorLevel:
				return scope.ErrorEnabled()
			case zapcore.WarnLevel:
				return scope.WarnEnabled()
			case zapcore.InfoLevel:
				return scope.InfoEnabled()
			}
			return scope.DebugEnabled()
		}
		return zapcore.NewCore(enc, outputSink, zap.LevelEnablerFunc(enabler))
	}
	return alwaysOn,
		conditionallyOn,
		errSink, nil
}

// Optimized date parsing and formatting
func formatDate(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
	t = t.UTC()
	year, month, day := t.Date()
	hour, minute, second := t.Clock()
	micros := t.Nanosecond() / 1000

	buf := make([]byte, 27)

	buf[0] = byte((year/1000)%10) + '0'
	buf[1] = byte((year/100)%10) + '0'
	buf[2] = byte((year/10)%10) + '0'
	buf[3] = byte(year%10) + '0'
	buf[4] = '-'
	buf[5] = byte((month)/10) + '0'
	buf[6] = byte((month)%10) + '0'
	buf[7] = '-'
	buf[8] = byte((day)/10) + '0'
	buf[9] = byte((day)%10) + '0'
	buf[10] = 'T'
	buf[11] = byte((hour)/10) + '0'
	buf[12] = byte((hour)%10) + '0'
	buf[13] = ':'
	buf[14] = byte((minute)/10) + '0'
	buf[15] = byte((minute)%10) + '0'
	buf[16] = ':'
	buf[17] = byte((second)/10) + '0'
	buf[18] = byte((second)%10) + '0'
	buf[19] = '.'
	buf[20] = byte((micros/100000)%10) + '0'
	buf[21] = byte((micros/10000)%10) + '0'
	buf[22] = byte((micros/1000)%10) + '0'
	buf[23] = byte((micros/100)%10) + '0'
	buf[24] = byte((micros/10)%10) + '0'
	buf[25] = byte((micros)%10) + '0'
	buf[26] = 'Z'

	enc.AppendString(string(buf))
}

// processLevels breaks down the argument string into a set of scope + level and applies
// the result to a scope. This can be overriden globally
func processLevels(allScopes map[string]*Scope, arg string, setter func(*Scope, Level)) error {
	levels := strings.Split(arg, ",")
	for _, sl := range levels {
		s, l, err := convertScopedLevel(sl)
		if err != nil {
			return err
		}

		if scope, ok := allScopes[s]; ok {
			setter(scope, l)
		} else if s == OverrideScopeName {
			// override replaces everything
			for _, scope := range allScopes {
				setter(scope, l)
			}
		}
	}

	return nil
}

func udpateScopes(options *Options) error {
	allScopes := Scopes()

	levels := options.defaultOutputLevels
	if levels == "" {
		levels = options.outputLevels
	} else if options.outputLevels != "" {
		levels = levels + "," + options.outputLevels
	}

	// Update output level of all listed scopes
	if err := processLevels(allScopes, levels, func(s *Scope, l Level) { s.SetOutputLevel(l) }); err != nil {
		return err
	}

	// update stack tracing levels of all listed scopes
	if err := processLevels(allScopes, options.stackTraceLevels, func(s *Scope, l Level) { s.SetStackTraceLevel(l) }); err != nil {
		return err
	}

	sc := strings.Split(options.logCallers, ",")
	for _, s := range sc {
		if s == "" {
			continue
		}

		if s == OverrideScopeName {
			// ignore everything else and apply the override value
			for _, scope := range allScopes {
				scope.SetLogCallers(true)
			}
		}

		if scope, ok := allScopes[s]; ok {
			scope.SetLogCallers(true)
		}
	}

	// If gRPC logging is enabled
	if grpcScope.OutputLevel() != NoneLevel {
		options.logGRPC = true
	}

	return nil
}

func Configure(options *Options) error {
	if err := udpateScopes(options); err != nil {
		return err
	}

	baseLogger, logBuilder, errSink, err := prepareZap(options)
	if err != nil {
		return err
	}
	defaultLogger := logBuilder(DefaultScopeName)
	allLoggers := []*zapcore.Core{&baseLogger, &defaultLogger}

	var grpcLogger zapcore.Core
	if options.logGRPC {
		grpcLogger = logBuilder(GrpcScopeName)
		allLoggers = append(allLoggers, &grpcLogger)
	}

	closeFns := make([]func() error, 0)

	for _, ext := range options.extensions {
		for _, logger := range allLoggers {
			newLogger, closeFn, err := ext(*logger)
			if err != nil {
				return err
			}
			*logger = newLogger
			closeFns = append(closeFns, closeFn)
		}
	}

	pt := patchTable{
		write: func(ent zapcore.Entry, fields []zapcore.Field) error {
			err := baseLogger.Write(ent, fields)
			if ent.Level == zapcore.FatalLevel {
				funcs.Load().(patchTable).exitProcess(1)
			}

			return err
		},
		sync:        baseLogger.Sync,
		exitProcess: os.Exit,
		errorSink:   errSink,
		close: func() error {
			// best effort sync
			baseLogger.Sync() // nolint: errcheck
			for _, f := range closeFns {
				if err := f(); err != nil {
					return err
				}
			}
			return nil
		},
	}
	funcs.Store(pt)

	opts := []zap.Option{
		zap.ErrorOutput(errSink),
		zap.AddCallerSkip(1),
	}

	if defaultScope.LogCallers() {
		opts = append(opts, zap.AddCaller())
	}

	l := defaultScope.StackTraceLevel()
	if l != NoneLevel {
		opts = append(opts, zap.AddStacktrace(logLevelToZap[l]))
	}

	defaultZapLogger := zap.New(defaultLogger, opts...)

	// capture global zap logging and force if through the logger
	_ = zap.ReplaceGlobals(defaultZapLogger)

	// capture standard golang "log" package output and for it through the logger
	_ = zap.RedirectStdLog(defaultZapLogger)

	// capture gRPC logging
	if options.logGRPC {
		grpclog.SetLoggerV2(zapgrpc.NewLogger(zap.New(grpcLogger, opts...).WithOptions(zap.AddCallerSkip(3))))
	}

	configureKlog.Do(func() {
		klog.SetLogger(NewLogrAdapter(KlogScope))
	})

	if klogVerbose() {
		KlogScope.SetOutputLevel(DebugLevel)
	}

	return nil
}

// Sync flushes any buffered log entries.
// Processes should normally take care to call Sync before exiting.
func Sync() error {
	return funcs.Load().(patchTable).sync()
}

// Close implements io.Closer.
func Close() error {
	return funcs.Load().(patchTable).close()
}
