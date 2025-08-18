package root

import (
	"github.com/kyverno/kyverno-envoy-plugin/pkg/commands/serve"
	"github.com/kyverno/kyverno-envoy-plugin/pkg/log"
	"github.com/spf13/cobra"
)

var (
	LoggingOptions = defaultLogOptions()
)

func Command() *cobra.Command {
	root := &cobra.Command{
		Use:   "kyverno-envoy-plugin",
		Short: "kyverno-envoy-plugin is a plugin for Envoy",
	}

	// Attach the Istio logging options to the command.
	LoggingOptions.AttachCobraFlags(root)
	root.AddCommand(serve.Command())
	return root
}

func defaultLogOptions() *log.Options {
	o := log.DefaultOptions()

	o.SetDefaultOutputLevel("all", log.WarnLevel)
	// These scopes are too noisy even at warning level
	o.SetDefaultOutputLevel("validation", log.ErrorLevel)
	o.SetDefaultOutputLevel("processing", log.ErrorLevel)
	o.SetDefaultOutputLevel("kube", log.ErrorLevel)
	return o
}
