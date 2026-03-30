package cmd

import (
	"fmt"
	"net/http"
	"sort"

	"github.com/0funct0ry/xwebs/internal/config"
	"github.com/0funct0ry/xwebs/internal/ws"
	"github.com/spf13/cobra"
)

var (
	subprotocols []string
	insecure     bool
)

var connectCmd = &cobra.Command{
	Use:   "connect [alias|url]",
	Short: "Connect to a WebSocket endpoint using an alias or URL",
	Long: `Connect to a WebSocket endpoint. You can provide a full URL (starting with ws:// or wss://) 
or a short alias/bookmark defined in your configuration file.

Example:
  xwebs connect prod
  xwebs connect wss://echo.websocket.org --subprotocol v1.xwebs`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		target := args[0]
		url, headers, err := config.ResolveConnDetails(target)
		if err != nil {
			return err
		}

		fmt.Printf("Connecting to: %s\n", url)
		header := make(http.Header)
		if len(headers) > 0 {
			fmt.Println("Headers:")
			// Sort headers for deterministic output
			keys := make([]string, 0, len(headers))
			for k, v := range headers {
				keys = append(keys, k)
				header.Add(k, v)
			}
			sort.Strings(keys)
			for _, k := range keys {
				fmt.Printf("  %s: %s\n", k, headers[k])
			}
		}

		opts := []ws.DialOption{
			ws.WithHeaders(header),
			ws.WithSubprotocols(subprotocols...),
			ws.WithInsecureSkipVerify(insecure),
		}

		conn, err := ws.Dial(cmd.Context(), url, opts...)
		if err != nil {
			return fmt.Errorf("connection failed: %w", err)
		}
		defer conn.Close()

		fmt.Println("Handshake successful!")
		if conn.NegotiatedSubprotocol != "" {
			fmt.Printf("Negotiated Subprotocol: %s\n", conn.NegotiatedSubprotocol)
		}

		fmt.Println("\n(Full interactive session logic will be implemented in EPIC 04)")
		return nil
	},
}

func init() {
	connectCmd.Flags().StringSliceVarP(&subprotocols, "subprotocol", "s", []string{}, "suggested subprotocols for negotiation")
	connectCmd.Flags().BoolVarP(&insecure, "insecure", "k", false, "skip TLS certificate verification")
	rootCmd.AddCommand(connectCmd)
}
