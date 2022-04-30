package main

import (
	"os"

	"github.com/hduhelp/org-manager/cmd/dept"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "org-manager",
	Short: "org manager of multi-platform",
}

func main() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.Flags().BoolP("target", "t", false, "Custom the target")
	rootCmd.AddCommand(dept.Cmd)
}
