package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/0funct0ry/xwebs/internal/build"
	"github.com/spf13/cobra"
)

var (
	shortVersion bool
	jsonOutput   bool
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version information",
	Long:  "Print the version information including git commit hash, build date, and go version.",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		info := build.GetBuildInfo()

		if shortVersion {
			fmt.Println(info.Version)
			return
		}

		if jsonOutput {
			encoder := json.NewEncoder(os.Stdout)
			encoder.SetIndent("", "  ")
			if err := encoder.Encode(info); err != nil {
				fmt.Fprintf(os.Stderr, "Error encoding version to JSON: %v\n", err)
				os.Exit(1)
			}
			return
		}

		fmt.Println(info.String())
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
	versionCmd.Flags().BoolVarP(&shortVersion, "short", "s", false, "print only the version number")
	versionCmd.Flags().BoolVarP(&jsonOutput, "json", "j", false, "print output in JSON format")
}
