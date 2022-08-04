package base

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/manifoldco/promptui"
	orgmanager "github.com/org-tools/org-manager"
	"github.com/samber/lo"
	"github.com/spf13/cobra"
)

func SelectTarget(exc ...string) (orgmanager.Target, string) {
	targets := lo.Keys(orgmanager.Targets)
	targets = lo.Filter(targets, func(v string, i int) bool {
		return !lo.Contains(exc, targets[i])
	})
	prompt := promptui.Select{
		Label: "Select Target",
		Items: targets,
	}
	_, target, err := prompt.Run()
	cobra.CheckErr(err)
	return orgmanager.Targets[target], target
}

func InputStringWithHint(hint string) string {
	fmt.Printf("%s: ", hint)
	return InputString()
}

func InputString() string {
	reader := bufio.NewReader(os.Stdin)
	name, err := reader.ReadString('\n')
	cobra.CheckErr(err)
	return strings.TrimRight(name, "\n")
}
