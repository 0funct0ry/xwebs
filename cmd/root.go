package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfgFile   string
	verbose   bool
	quiet     bool
	color     string
	logLevel  string
	logFormat string
	profile   string
)

var validLogLevels = []string{"debug", "info", "warn", "error"}
var validColorModes = []string{"auto", "on", "off"}
var validLogFormats = []string{"text", "json"}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVarP(&cfgFile, "config", "c", "", "config file path (default is .xwebs.yaml in current directory)")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "enable verbose output")
	rootCmd.PersistentFlags().BoolVarP(&quiet, "quiet", "q", false, "suppress all output except errors")
	rootCmd.PersistentFlags().StringVar(&color, "color", "auto", "color output mode: auto, on, off")
	rootCmd.PersistentFlags().StringVar(&logLevel, "log-level", "info", "logging level: debug, info, warn, error")
	rootCmd.PersistentFlags().StringVar(&logFormat, "log-format", "text", "log format: text, json")
	rootCmd.PersistentFlags().StringVar(&profile, "profile", "", "profile name for configuration")

	rootCmd.PersistentFlags().MarkDeprecated("toggle", "this flag is no longer used")

	rootCmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		return validateFlags()
	}
}

func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		viper.AddConfigPath(".")
		viper.SetConfigType("yaml")
		viper.SetConfigName(".xwebs")
	}

	viper.SetEnvPrefix("XWEBS")
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	viper.AutomaticEnv()

	_ = viper.ReadInConfig()

	viper.BindPFlag("config", rootCmd.PersistentFlags().Lookup("config"))
	viper.BindPFlag("verbose", rootCmd.PersistentFlags().Lookup("verbose"))
	viper.BindPFlag("quiet", rootCmd.PersistentFlags().Lookup("quiet"))
	viper.BindPFlag("color", rootCmd.PersistentFlags().Lookup("color"))
	viper.BindPFlag("log-level", rootCmd.PersistentFlags().Lookup("log-level"))
	viper.BindPFlag("log-format", rootCmd.PersistentFlags().Lookup("log-format"))
	viper.BindPFlag("profile", rootCmd.PersistentFlags().Lookup("profile"))
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
		cmd.Help()
	},
}

func Execute() error {
	return rootCmd.Execute()
}

func GetConfig() *viper.Viper {
	return viper.GetViper()
}
