package cmd

import (
	"github.com/spf13/cobra"
)

// ConfigFlags registers configuration-related flags.
func ConfigFlags(cmd *cobra.Command) {
	cmd.Flags().StringVarP(&cfgFile, "config", "c", "", "config file path (default searches ~/.xwebs.yaml then .xwebs.yaml)")
	cmd.Flags().StringVar(&profile, "profile", "", "profile name for configuration")
}

// OutputFlags registers output and logging flags.
func OutputFlags(cmd *cobra.Command) {
	cmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "enable verbose output")
	cmd.Flags().BoolVarP(&quiet, "quiet", "q", false, "suppress all output except errors")
	cmd.Flags().StringVar(&color, "color", "auto", "color output mode: auto, on, off")
	cmd.Flags().StringVar(&logLevel, "log-level", "info", "logging level: debug, info, warn, error")
	cmd.Flags().StringVar(&logFormat, "log-format", "text", "log format: text, json")
}

// HandlerFlags registers flags related to message handlers.
func HandlerFlags(cmd *cobra.Command) {
	cmd.Flags().StringVar(&handlersFile, "handlers", "", "handlers configuration file path (YAML)")
	cmd.Flags().BoolVar(&noShellFunc, "no-shell-func", false, "disable dangerous template functions (shell, env, fileRead, etc.)")
	cmd.Flags().StringArrayVar(&onHandlers, "on", nil, "Define a quick handler (syntax: '<match> :: <action>')")
	cmd.Flags().StringArrayVar(&onMatchHandlers, "on-match", nil, "Define an inline JSON handler (syntax: '{\"match\": \"...\", \"action\": \"...\"}')")
	cmd.Flags().StringVar(&respondTemplate, "respond", "", "Default response template for inline handlers")
	cmd.Flags().BoolVar(&sandboxEnabled, "sandbox", false, "Enable shell command allowlisting for handlers")
	cmd.Flags().StringSliceVar(&allowlist, "allowlist", nil, "Comma-separated list of allowed shell commands")
}

// ConnectionFlags registers flags specific to WebSocket connections.
func ConnectionFlags(cmd *cobra.Command) {
	cmd.Flags().StringVar(&proxy, "proxy", "", "proxy URL (http, https, socks5)")
}
