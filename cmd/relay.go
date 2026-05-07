package cmd

import (
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/0funct0ry/xwebs/internal/config"
	"github.com/0funct0ry/xwebs/internal/relay"
	"github.com/0funct0ry/xwebs/internal/ws"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	relayAddr         string
	relayPort         int
	relayUpstream     string
	relayPaths        []string
	relayTLS          bool
	relayCert         string
	relayKey          string
	relayCA           string
	relayInsecure     bool
	relayHeaders      []string
	relaySubprotocols []string
)

var relayCmd = &cobra.Command{
	Use:   "relay",
	Short: "Start a WebSocket relay (MITM proxy)",
	Long: `Start a WebSocket relay that acts as a MITM proxy between a local client and an upstream server.
This allows for inspection and (eventually) transformation of WebSocket traffic.

Example:
  xwebs relay --port 9090 --upstream wss://api.example.com/ws
  xwebs relay -p 9090 -u prod-alias --path /ws --verbose
`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Sync flags with viper
		if cmd.Flags().Changed("addr") {
			viper.Set("relay.addr", relayAddr)
		} else if viper.IsSet("relay.addr") {
			relayAddr = viper.GetString("relay.addr")
		}

		if cmd.Flags().Changed("port") {
			viper.Set("relay.port", relayPort)
		} else if viper.IsSet("relay.port") {
			relayPort = viper.GetInt("relay.port")
		}

		if cmd.Flags().Changed("upstream") {
			viper.Set("relay.upstream", relayUpstream)
		} else if viper.IsSet("relay.upstream") {
			relayUpstream = viper.GetString("relay.upstream")
		}

		if relayUpstream == "" {
			return fmt.Errorf("upstream URL or alias is required (use --upstream)")
		}

		// Resolve upstream connection details
		details, err := config.ResolveConnDetails(relayUpstream)
		if err != nil {
			return fmt.Errorf("resolving upstream %q: %w", relayUpstream, err)
		}

		// Flags override configuration for upstream
		if cmd.Flags().Changed("insecure") {
			details.Insecure = relayInsecure
		}
		if cmd.Flags().Changed("ca") {
			details.CA = relayCA
		}

		// Construct dial options for upstream
		header := make(http.Header)
		if len(details.Headers) > 0 {
			for k, v := range details.Headers {
				header.Add(k, v)
			}
		}
		for _, h := range relayHeaders {
			parts := strings.SplitN(h, ":", 2)
			if len(parts) == 2 {
				header.Add(strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1]))
			}
		}

		dialOpts := []ws.DialOption{
			ws.WithHeaders(header),
			ws.WithSubprotocols(relaySubprotocols...),
			ws.WithInsecureSkipVerify(details.Insecure),
			ws.WithVerbose(verbose),
		}
		if details.Proxy != "" {
			dialOpts = append(dialOpts, ws.WithProxy(details.Proxy))
		}
		if details.CA != "" {
			dialOpts = append(dialOpts, ws.WithCACert(details.CA))
		}
		if details.Cert != "" || details.Key != "" {
			dialOpts = append(dialOpts, ws.WithClientCert(details.Cert, details.Key))
		}

		// Initialize relay
		opts := &relay.Options{
			BindAddr:    relayAddr,
			Port:        relayPort,
			UpstreamURL: details.URL,
			Paths:       relayPaths,
			Verbose:     verbose,
			Quiet:       quiet,
			TLSEnabled:  relayTLS,
			CertFile:    relayCert,
			KeyFile:     relayKey,
			DialOpts:    dialOpts,
		}

		r := relay.New(opts)

		if !quiet {
			protocol := "ws"
			if relayTLS {
				protocol = "wss"
			}
			fmt.Fprintf(os.Stderr, "✓ xwebs relay starting on %s:%d (%s) -> %s\n", relayAddr, relayPort, protocol, details.URL)
			if len(relayPaths) == 1 {
				fmt.Fprintf(os.Stderr, "✓ Listening on path: %s\n", relayPaths[0])
			} else {
				fmt.Fprintf(os.Stderr, "✓ Listening on paths: %s\n", strings.Join(relayPaths, ", "))
			}
			fmt.Fprintln(os.Stderr, "--------------------------------------------------")
		}

		return r.Start(cmd.Context())
	},
}

func init() {
	rootCmd.AddCommand(relayCmd)

	relayCmd.Flags().StringVarP(&relayAddr, "addr", "a", "localhost", "address to listen on")
	relayCmd.Flags().IntVarP(&relayPort, "port", "p", 8080, "port to listen on")
	relayCmd.Flags().StringVarP(&relayUpstream, "upstream", "u", "", "upstream WebSocket URL or alias (required)")
	relayCmd.Flags().StringSliceVar(&relayPaths, "path", []string{"/"}, "WebSocket path(s) to listen on (repeatable)")

	relayCmd.Flags().BoolVar(&relayTLS, "tls", false, "enable TLS for the relay listener")
	relayCmd.Flags().StringVar(&relayCert, "cert", "", "path to TLS certificate file")
	relayCmd.Flags().StringVar(&relayKey, "key", "", "path to TLS private key file")
	relayCmd.Flags().StringVar(&relayCA, "ca", "", "path to CA certificate file for upstream verification")
	relayCmd.Flags().BoolVar(&relayInsecure, "insecure", false, "skip TLS verification for upstream")

	relayCmd.Flags().StringSliceVar(&relayHeaders, "header", nil, "HTTP headers to add to upstream request (Key:Value)")
	relayCmd.Flags().StringSliceVar(&relaySubprotocols, "subprotocol", nil, "WebSocket subprotocols to request from upstream")

	ConfigFlags(relayCmd)
	OutputFlags(relayCmd)
}
