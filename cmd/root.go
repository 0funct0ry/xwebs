package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfgFile         string
	handlersFile    string
	verbose         bool
	quiet           bool
	color           string
	logLevel        string
	logFormat       string
	profile         string
	proxy           string
	noShellFunc     bool
	onHandlers      []string
	onMatchHandlers []string
	respondTemplate string
	sandboxEnabled  bool
	allowlist       []string
)

var validLogLevels = []string{"debug", "info", "warn", "error"}
var validColorModes = []string{"auto", "on", "off"}
var validLogFormats = []string{"text", "json"}

func init() {
	cobra.OnInitialize(initConfig)

	// Persistent flags removed in favor of command-specific flags
	_ = rootCmd.PersistentFlags().MarkDeprecated("toggle", "this flag is no longer used")

	rootCmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		return validateFlags()
	}
}

func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		home, _ := os.UserHomeDir()

		viper.AddConfigPath(".")
		viper.AddConfigPath(home)
		viper.SetConfigName(".xwebs")
		viper.SetConfigType("yaml")
	}

	viper.SetEnvPrefix("XWEBS")
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		var notFound viper.ConfigFileNotFoundError
		if !errors.As(err, &notFound) {
			fmt.Fprintf(os.Stderr, "Error reading config file: %v\n", err)
		}
	}

	// Apply named profile if specified
	if profile != "" {
		profileKey := fmt.Sprintf("profiles.%s", profile)
		if !viper.IsSet(profileKey) {
			fmt.Fprintf(os.Stderr, "Error: profile '%s' not found in configuration\n", profile)
			os.Exit(1)
		}

		profileConfig := viper.GetStringMap(profileKey)
		if err := viper.MergeConfigMap(profileConfig); err != nil {
			fmt.Fprintf(os.Stderr, "Error merging profile '%s': %v\n", profile, err)
			os.Exit(1)
		}
	}

	if viper.ConfigFileUsed() != "" && !quiet {
		fmt.Fprintf(os.Stderr, "Using config file: %s\n", viper.ConfigFileUsed())
	}

	// Find the active command to bind its local flags to Viper
	activeCmd, _, _ := rootCmd.Find(os.Args[1:])
	if activeCmd != nil {
		_ = viper.BindPFlags(activeCmd.Flags())

		// Sync global variables from Viper and update flag defaults for help text
		syncFlag := func(name string, ptr interface{}) {
			f := activeCmd.Flags().Lookup(name)
			if f == nil {
				return
			}
			switch v := ptr.(type) {
			case *string:
				*v = viper.GetString(name)
				f.DefValue = *v
			case *bool:
				*v = viper.GetBool(name)
				if *v {
					f.DefValue = "true"
				} else {
					f.DefValue = "false"
				}
			}
		}

		syncFlag("config", &cfgFile)
		syncFlag("verbose", &verbose)
		syncFlag("quiet", &quiet)
		syncFlag("color", &color)
		syncFlag("log-level", &logLevel)
		syncFlag("log-format", &logFormat)
		syncFlag("profile", &profile)
		syncFlag("proxy", &proxy)
		syncFlag("handlers", &handlersFile)
		syncFlag("no-shell-func", &noShellFunc)
		syncFlag("respond", &respondTemplate)
		syncFlag("sandbox", &sandboxEnabled)

		// String slices need manual syncing from Viper if not set via flags
		if !activeCmd.Flags().Changed("on") && viper.IsSet("on") {
			onHandlers = viper.GetStringSlice("on")
		}
		if !activeCmd.Flags().Changed("on-match") && viper.IsSet("on-match") {
			onMatchHandlers = viper.GetStringSlice("on-match")
		}
		if !activeCmd.Flags().Changed("allowlist") && viper.IsSet("allowlist") {
			allowlist = viper.GetStringSlice("allowlist")
		}
	}
}

func validateFlags() error {
	logLevel = strings.ToLower(logLevel)
	if !contains(validLogLevels, logLevel) {
		return fmt.Errorf("invalid log-level: %s (valid: %s)", logLevel, strings.Join(validLogLevels, ", "))
	}

	color = strings.ToLower(color)
	if !contains(validColorModes, color) {
		return fmt.Errorf("invalid color: %s (valid: %s)", color, strings.Join(validColorModes, ", "))
	}

	logFormat = strings.ToLower(logFormat)
	if !contains(validLogFormats, logFormat) {
		return fmt.Errorf("invalid log-format: %s (valid: %s)", logFormat, strings.Join(validLogFormats, ", "))
	}

	return nil
}

func contains(slice []string, item string) bool {
	for _, v := range slice {
		if v == item {
			return true
		}
	}
	return false
}

var rootCmd = &cobra.Command{
	Use:   "xwebs",
	Short: "A WebSocket CLI tool for developers",
	Long: `xwebs is a powerful WebSocket development tool with shell integration,
Go templates, and an optional React web UI. It supports client mode, server mode,
relay, broadcast, mock, replay, bench, and diff capabilities.

For more information, visit: https://github.com/0funct0ry/xwebs`,
	SilenceUsage:          false,
	SilenceErrors:         false,
	DisableFlagsInUseLine: false,
	Run: func(cmd *cobra.Command, args []string) {
		if profile != "" || verbose || quiet || logLevel != "info" {
			fmt.Fprintf(os.Stderr, "Active Configuration:\n")
			if profile != "" {
				fmt.Fprintf(os.Stderr, "  - Profile:    %s\n", profile)
			}
			fmt.Fprintf(os.Stderr, "  - Log Level:  %s\n", logLevel)
			fmt.Fprintf(os.Stderr, "  - Verbose:    %v\n", verbose)
			if quiet {
				fmt.Fprintf(os.Stderr, "  - Quiet:      %v\n", quiet)
			}
			if proxy != "" {
				fmt.Fprintf(os.Stderr, "  - Proxy:      %s\n", proxy)
			}
			if noShellFunc {
				fmt.Fprintf(os.Stderr, "  - Sandboxed:  %v\n", noShellFunc)
			}
			if handlersFile != "" {
				fmt.Fprintf(os.Stderr, "  - Handlers:   %s\n", handlersFile)
			}
			if len(onHandlers) > 0 {
				fmt.Fprintf(os.Stderr, "  - Inline:     %d handlers\n", len(onHandlers)+len(onMatchHandlers))
			}
			fmt.Fprintf(os.Stderr, "\n")
		}
		_ = cmd.Help()
	},
}

func Execute() error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	return rootCmd.ExecuteContext(ctx)
}

func GetConfig() *viper.Viper {
	return viper.GetViper()
}
