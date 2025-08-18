package log

import (
	"fmt"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type Scope struct {
	name        string
	nameToEmit  string
	description string
	callerSkip  int

	outputLevel     *atomic.Value
	stackTraceLevel *atomic.Value
	logCallers      *atomic.Value

	labelKeys []string
	labels    map[string]any
}

var (
	scopes = make(map[string]*Scope)
	lock   sync.RWMutex
)

func RegisterScope(name string, description string) *Scope {
	// Only allow internal callers to set callerSkip
	return registerScope(name, description, 0)
}

func registerScope(name string, description string, callerSkip int) *Scope {
	if strings.ContainsAny(name, ":,.") {
		panic(fmt.Sprintf("scope name %s is invalid, it cannot contain colons, commas, or periods", name))
	}

	lock.Lock()
	defer lock.Unlock()

	s, ok := scopes[name]
	if !ok {
		s = &Scope{
			name:            name,
			description:     description,
			callerSkip:      callerSkip,
			outputLevel:     &atomic.Value{},
			stackTraceLevel: &atomic.Value{},
			logCallers:      &atomic.Value{},
		}
		s.SetOutputLevel(InfoLevel)
		s.SetStackTraceLevel(NoneLevel)
		s.SetLogCallers(false)

		if name != DefaultScopeName {
			s.nameToEmit = name
		}

		scopes[name] = s
	}

	s.labels = make(map[string]any)
	return s
}

func FindScope(scope string) *Scope {
	lock.RLock()
	defer lock.RUnlock()

	s := scopes[scope]
	return s
}

func Scopes() map[string]*Scope {
	lock.RLock()
	defer lock.RUnlock()

	sc := make(map[string]*Scope, len(scopes))
	for k, v := range scopes {
		sc[k] = v
	}
	return sc
}

// Logging methods for each type of log level

func (s *Scope) Fatal(msg any) {
	if s.OutputLevel() >= FatalLevel {
		s.emit(zapcore.FatalLevel, fmt.Sprint(msg))
	}
}

func (s *Scope) Fatalf(format string, args ...any) {
	if s.OutputLevel() >= FatalLevel {
		s.emit(zapcore.FatalLevel, maybeSprintf(format, args))
	}
}

func (s *Scope) FatalEnabled() bool {
	return s.OutputLevel() >= FatalLevel
}

func (s *Scope) Error(msg any) {
	if s.OutputLevel() >= ErrorLevel {
		s.emit(zapcore.ErrorLevel, fmt.Sprint(msg))
	}
}

func (s *Scope) Errorf(format string, args ...any) {
	if s.OutputLevel() >= ErrorLevel {
		s.emit(zapcore.ErrorLevel, maybeSprintf(format, args))
	}
}

func (s *Scope) ErrorEnabled() bool {
	return s.OutputLevel() >= ErrorLevel
}

func (s *Scope) Warn(msg any) {
	if s.OutputLevel() >= WarnLevel {
		s.emit(zapcore.WarnLevel, fmt.Sprint(msg))
	}
}

func (s *Scope) Warnf(format string, args ...any) {
	if s.OutputLevel() >= WarnLevel {
		s.emit(zapcore.WarnLevel, maybeSprintf(format, args))
	}
}

func (s *Scope) WarnEnabled() bool {
	return s.OutputLevel() >= WarnLevel
}

func (s *Scope) Info(msg any) {
	if s.OutputLevel() >= InfoLevel {
		s.emit(zapcore.InfoLevel, fmt.Sprint(msg))
	}
}

func (s *Scope) Infof(format string, args ...any) {
	if s.OutputLevel() >= InfoLevel {
		s.emit(zapcore.InfoLevel, maybeSprintf(format, args))
	}
}

func (s *Scope) InfoEnabled() bool {
	return s.OutputLevel() >= InfoLevel
}

func (s *Scope) Debug(msg any) {
	if s.OutputLevel() >= DebugLevel {
		s.emit(zapcore.DebugLevel, fmt.Sprint(msg))
	}
}

func (s *Scope) Debugf(format string, args ...any) {
	if s.OutputLevel() >= DebugLevel {
		s.emit(zapcore.DebugLevel, maybeSprintf(format, args))
	}
}

func (s *Scope) DebugEnabled() bool {
	return s.OutputLevel() >= DebugLevel
}

// LogWithTime outputs a message with a given timestamp.
func (s *Scope) LogWithTime(level Level, msg string, t time.Time) {
	if s.OutputLevel() >= level {
		s.emitWithTime(logLevelToZap[level], msg, t)
	}
}

// Getters - Setters

func (s *Scope) Name() string {
	return s.name
}

func (s *Scope) Description() string {
	return s.description
}

func (s *Scope) SetOutputLevel(l Level) {
	s.outputLevel.Store(l)
}

func (s *Scope) OutputLevel() Level {
	return s.outputLevel.Load().(Level)
}

func (s *Scope) SetStackTraceLevel(l Level) {
	s.stackTraceLevel.Store(l)
}

func (s *Scope) StackTraceLevel() Level {
	return s.stackTraceLevel.Load().(Level)
}

func (s *Scope) SetLogCallers(logCallers bool) {
	s.logCallers.Store(logCallers)
}

func (s *Scope) LogCallers() bool {
	return s.logCallers.Load().(bool)
}

func (s *Scope) copy() *Scope {
	out := *s
	out.labels = copyStringInterfaceMap(s.labels)
	return &out
}

// WithLabels adds a key-value pairs to the labels in s. The key must be a string, while the value may be any type.
// It returns a copy of s, with the labels added.
// e.g. newScope := oldScope.WithLabels("foo", "bar", "baz", 123, "qux", 0.123)
func (s *Scope) WithLabels(kvlist ...any) *Scope {
	out := s.copy()

	if len(kvlist)%2 != 0 {
		out.labels["WithLabels Error"] = fmt.Sprintf("even number of parameters required, got %d", len(kvlist))
		return out
	}

	for i := 0; i < len(kvlist); i += 2 {
		keyi := kvlist[i]
		key, ok := keyi.(string)
		if !ok {
			out.labels["WithLabels Error"] = fmt.Sprintf("Label name %v must be a string, got %T", keyi, keyi)
			return out
		}
		_, override := out.labels[key]
		out.labels[key] = kvlist[i+1]
		if override {
			// Key already set, just modify the value
			continue
		}
		out.labelKeys = append(out.labelKeys, key)
	}
	return out
}

func (s *Scope) emit(level zapcore.Level, msg string) {
	s.emitWithTime(level, msg, time.Now())
}

func (s *Scope) emitWithTime(level zapcore.Level, msg string, t time.Time) {
	if t.IsZero() {
		t = time.Now()
	}

	e := zapcore.Entry{
		Message:    msg,
		Level:      level,
		Time:       t,
		LoggerName: s.nameToEmit,
	}

	if s.LogCallers() {
		e.Caller = zapcore.NewEntryCaller(runtime.Caller(s.callerSkip + callerSkipOffset))
	}

	if dumpStack(level, s) {
		e.Stack = zap.Stack("").String
	}

	var fields []zapcore.Field
	if useJSON.Load().(bool) {
		fields = make([]zapcore.Field, 0, len(s.labelKeys))
		for _, k := range s.labelKeys {
			v := s.labels[k]
			fields = append(fields, zap.Field{
				Key:       k,
				Interface: v,
				Type:      zapcore.ReflectType,
			})
		}
	} else if len(s.labelKeys) > 0 {
		sb := &strings.Builder{}
		// just for optimization: pre-allocate arbitrary by estimating ~15 char for kv pair
		sb.Grow(len(msg) + 15*len(s.labelKeys))
		sb.WriteString(msg)
		sb.WriteString("\t")
		space := false
		for _, k := range s.labelKeys {
			if space {
				sb.WriteString(" ")
			}
			sb.WriteString(k)
			sb.WriteString("=")
			sb.WriteString(fmt.Sprint(s.labels[k]))
			space = true
		}
		e.Message = sb.String()
	}

	pt := funcs.Load().(patchTable)
	if pt.write != nil {
		if err := pt.write(e, fields); err != nil {
			fmt.Fprintf(pt.errorSink, "%v log write error: %v\n", time.Now(), err)
			_ = pt.errorSink.Sync()
		}
	}
}

func maybeSprintf(format string, args []any) string {
	msg := format
	if len(args) > 0 {
		msg = fmt.Sprintf(format, args...)
	}
	return msg
}

func copyStringInterfaceMap(m map[string]any) map[string]any {
	out := make(map[string]any, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}
