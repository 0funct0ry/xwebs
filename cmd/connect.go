package cmd

import (
	"fmt"
	"sort"

	"github.com/spf13/cobra"
	"github.com/0funct0ry/xwebs/internal/config"
)

var connectCmd = &cobra.Command{
	Use:   "connect [alias|url]",
	Short: "Connect to a WebSocket endpoint using an alias or URL",
	Long: `Connect to a WebSocket endpoint. You can provide a full URL (starting with ws:// or wss://) 
or a short alias/bookmark defined in your configuration file.

Example:
  xwebs connect prod
  xwebs connect wss://echo.websocket.org`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		target := args[0]
		url, headers, err := config.ResolveConnDetails(target)
		if err != nil {
			return err
		}

		fmt.Printf("Connecting to: %s\n", url)
		if len(headers) > 0 {
			fmt.Println("Headers:")
			// Sort headers for deterministic output
			keys := make([]string, 0, len(headers))
			for k := range headers {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			for _, k := range keys {
				fmt.Printf("  %s: %s\n", k, headers[k])
			}
		}

		// Actual connection logic is part of EPIC 04
		fmt.Println("\n(Connection logic will be implemented in EPIC 04)")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(connectCmd)
}
