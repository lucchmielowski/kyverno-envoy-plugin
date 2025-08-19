package serve

import (
	authzserver "github.com/kyverno/kyverno-envoy-plugin/pkg/commands/serve/authz-server"
	sidecarinjector "github.com/kyverno/kyverno-envoy-plugin/pkg/commands/serve/sidecar-injector"
	validationwebhook "github.com/kyverno/kyverno-envoy-plugin/pkg/commands/serve/validation-webhook"
	"github.com/spf13/cobra"
)

var (
	logLevel        string
	logFormat       string
	disableLogColor bool
)

func Command() *cobra.Command {
	command := &cobra.Command{
		Use:   "serve",
		Short: "Run Kyverno Envoy Plugin servers",
	}

	initLoggingFlags(command)

	command.AddCommand(authzserver.Command())
	command.AddCommand(sidecarinjector.Command())
	command.AddCommand(validationwebhook.Command())

	return command
}

func initLoggingFlags(cmd *cobra.Command) {
	cmd.Flags().StringVar(&logFormat, "log-format", "json", "Format of logging to use, values can be: [json, text]")
	cmd.Flags().BoolVar(&disableLogColor, "disable-log-color", true, "Disable logs color (default: true)")
}
