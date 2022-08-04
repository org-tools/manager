package main

import (
	"os"

	"github.com/org-tools/manager/cmd/dept"
	"github.com/org-tools/manager/cmd/monitor"
	"github.com/org-tools/manager/cmd/user"
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
	rootCmd.AddCommand(dept.Cmd, user.Cmd, monitor.Cmd)
}
