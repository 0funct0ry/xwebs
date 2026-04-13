package cmd

import (
	"encoding/json"
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
	serveTLS   bool
	serveCert  string
	serveKey   string
	serveAPI   bool
	serveMetrics bool
	allowedOrigins []string
	allowIPs       []string
	denyIPs        []string
	rateLimit      string
	serveUI        bool
)

var serveCmd = &cobra.Command{
	Use:     "serve",
	Aliases: []string{"s", "srv"},
	Short:   "Start a WebSocket server",
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

		// Add inline handlers from CLI flags
		for i, hStr := range onHandlers {
			h, err := handler.ParseInlineHandler(hStr, respondTemplate, i+1)
			if err != nil {
				return fmt.Errorf("invalid inline handler %d: %w", i+1, err)
			}
			handlers = append(handlers, h)
		}

		for i, hJSON := range onMatchHandlers {
			var h handler.Handler
			if err := json.Unmarshal([]byte(hJSON), &h); err != nil {
				return fmt.Errorf("invalid inline JSON handler %d: %w", i+1, err)
			}
			if h.Name == "" {
				h.Name = fmt.Sprintf("inline-json-%d", i+1)
			}
			if respondTemplate != "" && h.Respond == "" {
				h.Respond = respondTemplate
			}
			handlers = append(handlers, h)
		}

		if len(handlers) > 0 && !quiet && (len(onHandlers) > 0 || len(onMatchHandlers) > 0) {
			fmt.Fprintf(os.Stderr, "✓ Added %d inline handlers\n", len(onHandlers)+len(onMatchHandlers))
		}

		// Normalize paths: ensure they start with /
		for i, p := range servePaths {
			if !strings.HasPrefix(p, "/") {
				servePaths[i] = "/" + p
			}
		}

		if serveTLS {
			if serveCert == "" || serveKey == "" {
				return fmt.Errorf("--cert and --key must be provided when --tls is enabled")
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
			server.WithTLS(serveTLS),
			server.WithCertFile(serveCert),
			server.WithKeyFile(serveKey),
			server.WithAPI(serveAPI),
			server.WithMetrics(serveMetrics),
			server.WithAllowedOrigins(allowedOrigins),
			server.WithAllowIPs(allowIPs),
			server.WithDenyIPs(denyIPs),
			server.WithRateLimit(rateLimit),
			server.WithUI(serveUI),
		)

		if !quiet {
			protocol := "ws"
			if serveTLS {
				protocol = "wss"
			}
			fmt.Fprintf(os.Stderr, "✓ xwebs server starting on :%d (%s)\n", servePort, protocol)
			if len(servePaths) == 1 {
				fmt.Fprintf(os.Stderr, "✓ Listening on path: %s\n", servePaths[0])
			} else {
				fmt.Fprintf(os.Stderr, "✓ Listening on paths: %s\n", strings.Join(servePaths, ", "))
			}
			if handlersFile != "" {
				fmt.Fprintf(os.Stderr, "✓ Handlers loaded from: %s\n", handlersFile)
			}
			fmt.Fprintln(os.Stderr, "--------------------------------------------------")
		}

		return srv.Start(cmd.Context())
	},
}

func init() {
	rootCmd.AddCommand(serveCmd)

	serveCmd.Flags().IntVarP(&servePort, "port", "p", 8080, "port to listen on")
	serveCmd.Flags().StringArrayVar(&servePaths, "path", []string{"/"}, "WebSocket path(s) to listen on")
	serveCmd.Flags().BoolVar(&serveTLS, "tls", false, "enable TLS (wss://)")
	serveCmd.Flags().StringVar(&serveCert, "cert", "", "path to certificate file")
	serveCmd.Flags().StringVar(&serveKey, "key", "", "path to private key file")
	serveCmd.Flags().BoolVar(&serveAPI, "api", true, "enable REST API")
	serveCmd.Flags().BoolVar(&serveMetrics, "metrics", false, "enable Prometheus metrics")
	serveCmd.Flags().StringArrayVar(&allowedOrigins, "allowed-origins", nil, "allowed origins for WebSocket connections")
	serveCmd.Flags().StringArrayVar(&allowIPs, "allow-ip", nil, "allowed IP addresses or CIDR ranges")
	serveCmd.Flags().StringArrayVar(&denyIPs, "deny-ip", nil, "denied IP addresses or CIDR ranges")
	serveCmd.Flags().StringVar(&rateLimit, "rate-limit", "", "rate limit (e.g., '10/s' per-client, or '10/s,100/s' for per-client,global)")
	serveCmd.Flags().BoolVar(&serveUI, "ui", false, "enable web UI")
}
