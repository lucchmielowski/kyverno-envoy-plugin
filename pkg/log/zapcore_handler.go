package log

import "go.uber.org/zap/zapcore"

// callerSkipOffset is how many callers to pop off the stack to determine the caller function locality, used for
// adding file/line number to log output.
const callerSkipOffset = 3

var toLevel = map[zapcore.Level]Level{
	zapcore.FatalLevel: FatalLevel,
	zapcore.ErrorLevel: ErrorLevel,
	zapcore.WarnLevel:  WarnLevel,
	zapcore.InfoLevel:  InfoLevel,
	zapcore.DebugLevel: DebugLevel,
}

func dumpStack(level zapcore.Level, scope *Scope) bool {
	treshold := toLevel[level]
	if scope != defaultScope {
		treshold = ErrorLevel
		switch level {
		case zapcore.FatalLevel:
			treshold = FatalLevel

		}
	}
	return scope.StackTraceLevel() >= treshold
}
