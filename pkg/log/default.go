package log

// Enable logging globally by setting up a global scope

func registerDefaultScopes() (defaults *Scope, grpc *Scope) {
	return registerScope(DefaultScopeName, "Unscoped logging messages.", 1),
		registerScope(GrpcScopeName, "logs from gRPC", 3)
}

var defaultScope, grpcScope = registerDefaultScopes()

func Fatal(fields any) {
	defaultScope.Fatal(fields)
}

func Fatalf(format string, args ...any) {
	defaultScope.Fatalf(format, args...)
}

func FatalEnabled() bool {
	return defaultScope.FatalEnabled()
}

func Error(fields any) {
	defaultScope.Error(fields)
}

func Errorf(format string, args ...any) {
	defaultScope.Errorf(format, args...)
}

func ErrorEnabled() bool {
	return defaultScope.ErrorEnabled()
}

func Warn(fields any) {
	defaultScope.Warn(fields)
}

func Warnf(format string, args ...any) {
	defaultScope.Warnf(format, args...)
}

func WarnEnabled() bool {
	return defaultScope.WarnEnabled()
}

func Info(fields any) {
	defaultScope.Info(fields)
}

func Infof(format string, args ...any) {
	defaultScope.Infof(format, args...)
}

func InfoEnabled() bool {
	return defaultScope.InfoEnabled()
}

func Debug(fields any) {
	defaultScope.Debug(fields)
}

func Debugf(format string, args ...any) {
	defaultScope.Debugf(format, args...)
}

func DebugEnabled() bool {
	return defaultScope.DebugEnabled()
}

func WithLabels(kvlist ...any) *Scope {
	return defaultScope.WithLabels(kvlist...)
}
