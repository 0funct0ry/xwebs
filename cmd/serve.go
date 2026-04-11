package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/0funct0ry/xwebs/internal/handler"
	"github.com/0funct0ry/xwebs/internal/server"
	"github.com/0funct0ry/xwebs/internal/template"
	"github.com/spf13/cobra"
)

var (
	servePort  int
	servePaths []string
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start a WebSocket server",
	Long: `Start a WebSocket server with handler support.
You can specify the listening port and one or more WebSocket paths.
Handlers can be loaded from a configuration file using the --handlers flag.

Example:
  xwebs serve --port 8080 --path /ws
  xwebs serve --handlers echo.yaml --port 9000 --path /api --path /chat`,
	RunE: func(cmd *cobra.Command, args []string) error {
		tmplEngine := template.New(noShellFunc)
		
		var handlers []handler.Handler
		var variables map[string]interface{}

		if handlersFile != "" {
			cfg, err := handler.LoadConfig(handlersFile)
			if err != nil {
				return fmt.Errorf("loading handlers: %w", err)
			}
			handlers = cfg.Handlers
			variables = cfg.Variables

			// Load sandbox settings from handlers config if not explicitly set via flags
			if !cmd.Flags().Changed("sandbox") && cfg.Sandbox {
				sandboxEnabled = cfg.Sandbox
			}
			if !cmd.Flags().Changed("allowlist") && len(cfg.Allowlist) > 0 {
				allowlist = cfg.Allowlist
			}

			if !quiet {
				fmt.Fprintf(os.Stderr, "✓ Loaded %d handlers from %s\n", len(handlers), handlersFile)
			}
		}

		srv := server.New(
			server.WithPort(servePort),
			server.WithPaths(servePaths),
			server.WithHandlers(handlers),
			server.WithVariables(variables),
			server.WithTemplateEngine(tmplEngine),
			server.WithVerbose(verbose),
			server.WithSandbox(sandboxEnabled),
			server.WithAllowlist(allowlist),
		)

		if !quiet {
			fmt.Fprintf(os.Stderr, "Starting xwebs server on :%d\n", servePort)
			fmt.Fprintf(os.Stderr, "Listening on paths: %s\n", strings.Join(servePaths, ", "))
			if handlersFile != "" {
				fmt.Fprintf(os.Stderr, "Handlers loaded from: %s\n", handlersFile)
			}
		}

		return srv.Start(cmd.Context())
	},
}

func init() {
	rootCmd.AddCommand(serveCmd)

	serveCmd.Flags().IntVarP(&servePort, "port", "p", 8080, "port to listen on")
	serveCmd.Flags().StringArrayVar(&servePaths, "path", []string{"/"}, "WebSocket path(s) to listen on")
}
