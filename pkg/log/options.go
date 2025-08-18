package log

import (
	"fmt"
	"sort"
	"strings"

	"github.com/spf13/cobra"
	"go.uber.org/zap/zapcore"
)

const (
	DefaultScopeName       = "default"
	OverrideScopeName      = "all"
	defaultOutputLevel     = InfoLevel
	defaultStackTraceLevel = NoneLevel
	defaultOutputPath      = "stdout"
	defaultErrorOutputPath = "stderr"
)

type Level int

const (
	NoneLevel Level = iota
	FatalLevel
	ErrorLevel
	WarnLevel
	InfoLevel
	DebugLevel
)

var levelToString = map[Level]string{
	NoneLevel:  "none",
	FatalLevel: "fatal",
	ErrorLevel: "error",
	WarnLevel:  "warn",
	InfoLevel:  "info",
	DebugLevel: "debug",
}

var stringToLevel = map[string]Level{
	"none":  NoneLevel,
	"fatal": FatalLevel,
	"error": ErrorLevel,
	"warn":  WarnLevel,
	"info":  InfoLevel,
	"debug": DebugLevel,
}

func StringToLevel(level string) Level {
	return stringToLevel[level]
}

func LevelToString(level Level) string {
	return levelToString[level]
}

type Options struct {
	OutputPaths []string

	ErrorOutputPaths []string
	JSONEncoding     bool
	logGRPC          bool

	outputLevels        string
	defaultOutputLevels string
	logCallers          string
	stackTraceLevels    string

	extensions []Extension
}

func DefaultOptions() *Options {
	return &Options{
		OutputPaths:         []string{defaultOutputPath},
		ErrorOutputPaths:    []string{defaultErrorOutputPath},
		defaultOutputLevels: DefaultScopeName + ":" + levelToString[defaultOutputLevel] + ",grpc:none",
		stackTraceLevels:    DefaultScopeName + ":" + levelToString[defaultStackTraceLevel],
		logGRPC:             false,
	}
}

func (o *Options) SetDefaultOutputLevel(scope string, level Level) {
	sl := scope + ":" + levelToString[level]
	levels := strings.Split(o.defaultOutputLevels, ",")
	if scope == DefaultScopeName {
		o.defaultOutputLevels = sl
	} else {
		for i, l := range levels {
			if strings.HasPrefix(l, scope+":") {
				levels[i] = sl
				o.defaultOutputLevels = strings.Join(levels, ",")
				return
			}
		}
		o.defaultOutputLevels += "," + sl
	}

	prefix := scope + ":"
	for i, ol := range levels {
		if strings.HasPrefix(ol, prefix) {
			levels[i] = sl
			o.defaultOutputLevels = strings.Join(levels, ",")
			return
		}
	}

	levels = append(levels, sl)
	o.defaultOutputLevels = strings.Join(levels, ",")
}

func convertScopedLevel(sl string) (string, Level, error) {
	var s string
	var l string

	pieces := strings.Split(sl, ":")
	if len(pieces) == 1 {
		s = DefaultScopeName
		l = pieces[0]
	} else if len(pieces) == 2 {
		s = pieces[0]
		l = pieces[1]
	} else {
		return "", NoneLevel, fmt.Errorf("invalid output level format: %s", sl)
	}
	level, ok := stringToLevel[l]
	if !ok {
		return "", NoneLevel, fmt.Errorf("invalid output level: %s", l)
	}

	return s, level, nil
}

func (o *Options) AttachCobraFlags(cmd *cobra.Command) {
	o.AttatchFlags(
		cmd.PersistentFlags().StringArrayVar,
		cmd.PersistentFlags().StringVar,
		cmd.PersistentFlags().IntVar,
		cmd.PersistentFlags().BoolVar)
}

// AttatchFlags attaches the logging flags to the given flag set.
func (o *Options) AttatchFlags(
	stringArrayVar func(p *[]string, name string, value []string, usage string),
	stringVar func(p *string, name string, value string, usage string),
	_ func(p *int, name string, value int, usage string),
	boolVar func(p *bool, name string, value bool, usage string),
) {
	stringArrayVar(&o.OutputPaths, "log_target", o.OutputPaths,
		"The set of paths where to output the log. This can be any path as well as special values: stdout or stderr.")

	boolVar(&o.JSONEncoding, "log_json", o.JSONEncoding,
		"Whether to output the log in JSON format. If false, the log will be output in a human-readable format.")

	levelListString := fmt.Sprintf("[%s, %s, %s, %s, %s, %s]",
		levelToString[NoneLevel],
		levelToString[FatalLevel],
		levelToString[ErrorLevel],
		levelToString[WarnLevel],
		levelToString[InfoLevel],
		levelToString[DebugLevel])

	allScopes := Scopes()
	if len(allScopes) > 1 {
		keys := make([]string, 0, len(allScopes))
		for name := range allScopes {
			keys = append(keys, name)
		}
		keys = append(keys, OverrideScopeName)
		sort.Strings(keys)
		s := strings.Join(keys, ",")

		stringVar(&o.outputLevels, "log_level", o.outputLevels,
			fmt.Sprintf("Comma-separated minimum per-scope logging level of messages to output, in the form of "+
				"<scope>:<level>,<scope>:<level>,... where scope can be one of [%s] and level can be one of %s",
				s, levelListString))

		stringVar(&o.stackTraceLevels, "stack_trace_level", o.stackTraceLevels,
			fmt.Sprintf("Comma-separated minimum per-scope logging level at which stack traces are captured, in the form of "+
				"<scope>:<level>,<scope:level>,... where scope can be one of [%s] and level can be one of %s",
				s, levelListString))

		stringVar(&o.logCallers, "log_caller", o.logCallers,
			fmt.Sprintf("Comma-separated list of scopes for which to include caller information, scopes can be any of [%s]", s))
	} else {
		stringVar(&o.outputLevels, "log_output_level", o.outputLevels,
			fmt.Sprintf("The minimum logging level of messages to output,  can be one of %s", levelListString))

		stringVar(&o.stackTraceLevels, "log_stacktrace_level", o.stackTraceLevels,
			fmt.Sprintf("The minimum logging level at which stack traces are captured, can be one of %s",
				levelListString))
	}
}

// Extension provides an extension mechanism for logs.
// This is essentially like https://pkg.go.dev/golang.org/x/exp/slog#Handler.
// This interface should be considered unstable; we will likely swap it for slog in the future and not expose zap internals.
// Returns a modified Core interface, and a Close() function.
type Extension func(c zapcore.Core) (zapcore.Core, func() error, error)

func (o *Options) WithExtension(e Extension) *Options {
	o.extensions = append(o.extensions, e)
	return o
}
