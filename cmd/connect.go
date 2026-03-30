package cmd

import (
	"fmt"
	"net/http"
	"sort"
	"time"

	"github.com/0funct0ry/xwebs/internal/config"
	"github.com/0funct0ry/xwebs/internal/ws"
	"github.com/spf13/cobra"
)

var (
	subprotocols []string
	insecure     bool
	certFile     string
	keyFile      string
	caFile       string
	pingInterval time.Duration
	pongWait     time.Duration
)

var connectCmd = &cobra.Command{
	Use:   "connect [alias|url]",
	Short: "Connect to a WebSocket endpoint using an alias or URL",
	Long: `Connect to a WebSocket endpoint. You can provide a full URL (starting with ws:// or wss://) 
or a short alias/bookmark defined in your configuration file.

Example:
  xwebs connect prod
  xwebs connect wss://echo.websocket.org --subprotocol v1.xwebs
  xwebs connect secure-server --cert client.crt --key client.key --ca ca.crt`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		target := args[0]
		details, err := config.ResolveConnDetails(target)
		if err != nil {
			return err
		}

		// Flags override configuration
		if cmd.Flags().Changed("insecure") {
			details.Insecure = insecure
		}
		if cmd.Flags().Changed("cert") {
			details.Cert = certFile
		}
		if cmd.Flags().Changed("key") {
			details.Key = keyFile
		}
		if cmd.Flags().Changed("ca") {
			details.CA = caFile
		}
		if cmd.Flags().Changed("proxy") {
			details.Proxy = proxy
		}
		if cmd.Flags().Changed("ping-interval") {
			details.PingInterval = pingInterval
		}
		if cmd.Flags().Changed("pong-wait") {
			details.PongWait = pongWait
		}

		fmt.Printf("Connecting to: %s\n", details.URL)
		if details.Proxy != "" {
			fmt.Printf("Proxy: %s\n", details.Proxy)
		}
		header := make(http.Header)
		if len(details.Headers) > 0 {
			fmt.Println("Headers:")
			// Sort headers for deterministic output
			keys := make([]string, 0, len(details.Headers))
			for k := range details.Headers {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			for _, k := range keys {
				v := details.Headers[k]
				header.Add(k, v)
				fmt.Printf("  %s: %s\n", k, v)
			}
		}

		opts := []ws.DialOption{
			ws.WithHeaders(header),
			ws.WithSubprotocols(subprotocols...),
			ws.WithInsecureSkipVerify(details.Insecure),
			ws.WithPingInterval(details.PingInterval),
			ws.WithPongWait(details.PongWait),
			ws.WithVerbose(verbose),
		}

		if details.Proxy != "" {
			opts = append(opts, ws.WithProxy(details.Proxy))
		}

		if details.CA != "" {
			opts = append(opts, ws.WithCACert(details.CA))
		}
		if details.Cert != "" || details.Key != "" {
			opts = append(opts, ws.WithClientCert(details.Cert, details.Key))
		}

		conn, err := ws.Dial(cmd.Context(), details.URL, opts...)
		if err != nil {
			return fmt.Errorf("connection failed: %w", err)
		}
		defer conn.Close()

		fmt.Println("Handshake successful!")
		if conn.NegotiatedSubprotocol != "" {
			fmt.Printf("Negotiated Subprotocol: %s\n", conn.NegotiatedSubprotocol)
		}

		fmt.Println("\n(Full interactive session logic will be implemented in EPIC 04)")
		fmt.Println("Press Ctrl+C to disconnect...")

		// Wait for connection to close or context to be cancelled
		select {
		case <-conn.Done():
			if err := conn.Err(); err != nil {
				fmt.Printf("\nConnection closed with error: %v\n", err)
			} else {
				fmt.Println("\nConnection closed by server.")
			}
		case <-cmd.Context().Done():
			fmt.Println("\nDisconnecting...")
		}

		return nil
	},
}

func init() {
	connectCmd.Flags().StringSliceVarP(&subprotocols, "subprotocol", "s", []string{}, "suggested subprotocols for negotiation")
	connectCmd.Flags().BoolVarP(&insecure, "insecure", "k", false, "skip TLS certificate verification")
	connectCmd.Flags().StringVar(&certFile, "cert", "", "path to client certificate file (mTLS)")
	connectCmd.Flags().StringVar(&keyFile, "key", "", "path to client key file (mTLS)")
	connectCmd.Flags().StringVar(&caFile, "ca", "", "path to custom CA certificate file")
	connectCmd.Flags().DurationVar(&pingInterval, "ping-interval", 30*time.Second, "interval for automatic ping messages (0 to disable)")
	connectCmd.Flags().DurationVar(&pongWait, "pong-wait", 60*time.Second, "wait time for a pong response")
	rootCmd.AddCommand(connectCmd)
}
