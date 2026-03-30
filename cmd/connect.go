package cmd

import (
	"fmt"
	"net/http"
	"time"

	"github.com/0funct0ry/xwebs/internal/config"
	"github.com/0funct0ry/xwebs/internal/ws"
	"github.com/spf13/cobra"
)

var (
	pingInterval      time.Duration
	pongWait          time.Duration
	reconnect         bool
	reconnectBackoff  time.Duration
	reconnectMax      time.Duration
	reconnectAttempts int
	maxMessageSize    int64
	compress          bool
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
		if cmd.Flags().Changed("reconnect") {
			details.Reconnect = reconnect
		}
		if cmd.Flags().Changed("reconnect-backoff") {
			details.ReconnectBackoff = reconnectBackoff
		}
		if cmd.Flags().Changed("reconnect-max") {
			details.ReconnectMax = reconnectMax
		}
		if cmd.Flags().Changed("reconnect-attempts") {
			details.ReconnectAttempts = reconnectAttempts
		}
		if cmd.Flags().Changed("max-message-size") {
			details.MaxMessageSize = maxMessageSize
		}
		if cmd.Flags().Changed("compress") {
			details.Compress = compress
		}

		header := make(http.Header)
		if len(details.Headers) > 0 {
			for k, v := range details.Headers {
				header.Add(k, v)
			}
		}

		opts := []ws.DialOption{
			ws.WithHeaders(header),
			ws.WithSubprotocols(subprotocols...),
			ws.WithInsecureSkipVerify(details.Insecure),
			ws.WithPingInterval(details.PingInterval),
			ws.WithPongWait(details.PongWait),
			ws.WithVerbose(verbose),
			ws.WithReconnect(details.Reconnect),
			ws.WithReconnectBackoff(details.ReconnectBackoff),
			ws.WithReconnectMax(details.ReconnectMax),
			ws.WithReconnectAttempts(details.ReconnectAttempts),
			ws.WithMaxMessageSize(details.MaxMessageSize),
			ws.WithCompression(details.Compress),
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

		reconnectCount := 0
		for {
			fmt.Printf("Connecting to: %s\n", details.URL)
			if details.Proxy != "" && reconnectCount == 0 {
				fmt.Printf("Proxy: %s\n", details.Proxy)
			}

			conn, err := ws.Dial(cmd.Context(), details.URL, opts...)
			if err != nil {
				if !details.Reconnect {
					return fmt.Errorf("connection failed: %w", err)
				}

				if details.ReconnectAttempts > 0 && reconnectCount >= details.ReconnectAttempts {
					return fmt.Errorf("connection failed after %d attempts: %w", reconnectCount, err)
				}

				backoff := ws.ExponentialBackoff(details.ReconnectBackoff, details.ReconnectMax, reconnectCount)
				fmt.Printf("Connection failed: %v. Retrying in %v... (attempt %d)\n", err, backoff, reconnectCount+1)

				select {
				case <-time.After(backoff):
					reconnectCount++
					continue
				case <-cmd.Context().Done():
					return nil
				}
			}

			reconnectCount = 0 // Reset on successful connection
			fmt.Println("Handshake successful!")
			if conn.NegotiatedSubprotocol != "" {
				fmt.Printf("Negotiated Subprotocol: %s\n", conn.NegotiatedSubprotocol)
			}
			if conn.IsCompressionEnabled() {
				fmt.Println("Compression: enabled (permessage-deflate)")
			} else if conn.CompressionRequested() {
				fmt.Println("Compression: requested but not negotiated by server")
			}

			fmt.Println("\n(Full interactive session logic will be implemented in EPIC 04)")
			fmt.Println("Press Ctrl+C to disconnect...")

			// Wait for connection to close or context to be cancelled
			var closedUnexpectedly bool
			select {
			case <-conn.Done():
				if err := conn.Err(); err != nil {
					fmt.Printf("\nConnection lost: %v\n", err)
					closedUnexpectedly = true
				} else {
					fmt.Println("\nConnection closed by server.")
					closedUnexpectedly = false
				}
			case <-cmd.Context().Done():
				fmt.Println("\nDisconnecting...")
				conn.Close()
				return nil
			}

			if details.Reconnect && closedUnexpectedly {
				backoff := ws.ExponentialBackoff(details.ReconnectBackoff, details.ReconnectMax, 0)
				fmt.Printf("Attempting to reconnect in %v...\n", backoff)
				select {
				case <-time.After(backoff):
					continue
				case <-cmd.Context().Done():
					return nil
				}
			}
			break
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
	connectCmd.Flags().BoolVar(&reconnect, "reconnect", false, "enable automatic reconnection")
	connectCmd.Flags().DurationVar(&reconnectBackoff, "reconnect-backoff", 1*time.Second, "initial backoff duration for reconnection")
	connectCmd.Flags().DurationVar(&reconnectMax, "reconnect-max", 30*time.Second, "maximum backoff duration for reconnection")
	connectCmd.Flags().IntVar(&reconnectAttempts, "reconnect-attempts", 0, "maximum number of reconnection attempts (0 for unlimited)")
	connectCmd.Flags().Int64Var(&maxMessageSize, "max-message-size", 0, "maximum message size in bytes (0 for unlimited)")
	connectCmd.Flags().BoolVar(&compress, "compress", false, "enable per-message-deflate compression")
	rootCmd.AddCommand(connectCmd)
}

var (
	subprotocols []string
	insecure     bool
	certFile     string
	keyFile      string
	caFile       string
)
