package dept

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	orgmanager "github.com/hduhelp/org-manager"
	"github.com/manifoldco/promptui"
	"github.com/samber/lo"
	"github.com/spf13/cobra"
)

func init() {
	Cmd.AddCommand(infoCmd, createCmd, linkCmd, listCmd)
}

var Cmd = &cobra.Command{
	Use:   "dept",
	Short: "dept management",
	Run: func(cmd *cobra.Command, args []string) {
		targets := lo.Keys(orgmanager.Targets)
		prompt := promptui.Select{
			Label: "Select Target",
			Items: targets,
		}

		_, target, err := prompt.Run()

		if err != nil {
			fmt.Printf("Prompt failed %v\n", err)
			return
		}
		nowDepartment := orgmanager.Targets[target].RootDepartment()
		fmt.Println(orgmanager.ExternalIdentityOfDepartment(orgmanager.Targets[target], nowDepartment))
		for _, v := range nowDepartment.Users() {
			fmt.Println(orgmanager.ExternalIdentityOfUser(orgmanager.Targets[target], v), v.GetUserName())
		}
		for {
			depts := nowDepartment.SubDepartments()
			if len(depts) == 0 {
				return
			}
			deptsName := make([]string, 0)
			for _, v := range depts {
				deptsName = append(deptsName, v.Name())
			}
			prompt = promptui.Select{
				Label: "Select Department",
				Items: deptsName,
			}
			var deptName string
			_, deptName, err = prompt.Run()
			for _, v := range depts {
				if v.Name() == deptName {
					nowDepartment = v
				}
			}
			for _, v := range nowDepartment.Users() {
				fmt.Println(orgmanager.ExternalIdentityOfUser(orgmanager.Targets[target], v), v.GetUserName())
			}
			fmt.Println(orgmanager.ExternalIdentityOfDepartment(orgmanager.Targets[target], nowDepartment))
		}
	},
}

func selectTarget() orgmanager.Target {
	targets := lo.Keys(orgmanager.Targets)
	prompt := promptui.Select{
		Label: "Select Target",
		Items: targets,
	}
	_, target, err := prompt.Run()
	cobra.CheckErr(err)
	return orgmanager.Targets[target]
}

var infoCmd = &cobra.Command{
	Use:   "info",
	Short: "show dept info with extID",
	Run: func(cmd *cobra.Command, args []string) {
		extID, err := orgmanager.ExternalIdentityParseString(args[0])
		cobra.CheckErr(err)
		if extID.GetEntryType() != orgmanager.EntryTypeDept {
			fmt.Println("extID not type dept")
			return
		}
		target, err := orgmanager.GetTargetByPlatformAndSlug(extID.GetPlatform(), extID.GetTargetSlug())
		cobra.CheckErr(err)
		dept, err := target.LookupEntryDepartmentByInternalExternalIdentity(extID)
		cobra.CheckErr(err)
		fmt.Println(dept.DepartmentID(), dept.Name())
		fmt.Println(orgmanager.ExternalIdentityOfDepartment(target, dept))

		if entryCenter, ok := target.(orgmanager.EntryCenter); ok {
			dept, err := entryCenter.LookupEntryDepartmentByExternalIdentity(extID)
			cobra.CheckErr(err)
			for _, extID := range dept.GetExternalIdentities() {
				target, err := extID.GetTarget()
				cobra.CheckErr(err)
				linkedDept, err := target.LookupEntryDepartmentByInternalExternalIdentity(extID)
				cobra.CheckErr(err)
				fmt.Println(linkedDept.Name(), orgmanager.ExternalIdentityOfDepartment(target, linkedDept))
			}
		}
	},
}

var linkCmd = &cobra.Command{
	Use:   "link",
	Short: "link dept form to",
	Run: func(cmd *cobra.Command, args []string) {
		extIDNeedLink, err := orgmanager.ExternalIdentityParseString(args[0])
		cobra.CheckErr(err)
		if extIDNeedLink.GetEntryType() != orgmanager.EntryTypeDept {
			fmt.Println("extIDNeedLink not type dept")
			return
		}
		target, err := orgmanager.GetTargetByPlatformAndSlug(extIDNeedLink.GetPlatform(), extIDNeedLink.GetTargetSlug())
		cobra.CheckErr(err)
		extIDLinkTo, err := orgmanager.ExternalIdentityParseString(args[1])
		if extIDLinkTo.GetEntryType() != orgmanager.EntryTypeDept {
			fmt.Println("extIDLinkTo not type dept")
			return
		}
		cobra.CheckErr(err)
		targetShouldBeEntryCenter, err := orgmanager.GetTargetByPlatformAndSlug(extIDLinkTo.GetPlatform(), extIDLinkTo.GetTargetSlug())
		cobra.CheckErr(err)
		_, err = target.LookupEntryDepartmentByInternalExternalIdentity(extIDNeedLink)
		cobra.CheckErr(err)

		if entryCenter, ok := targetShouldBeEntryCenter.(orgmanager.EntryCenter); ok {
			dept, err := entryCenter.LookupEntryDepartmentByExternalIdentity(extIDNeedLink)
			if err != nil {
				fmt.Println(err)
			}
			if dept != nil && err == nil {
				fmt.Println("already linked")
				return
			}
		} else {
			fmt.Println(targetShouldBeEntryCenter.GetPlatform(), "should be EntryCenter")
			return
		}

		dept, err := targetShouldBeEntryCenter.LookupEntryDepartmentByInternalExternalIdentity(extIDLinkTo)
		cobra.CheckErr(err)
		deptExtIDStoreable := dept.(orgmanager.EntryExtIDStoreable)
		alreadyExtIDs := deptExtIDStoreable.GetExternalIdentities()
		fmt.Println(alreadyExtIDs)
		err = deptExtIDStoreable.SetExternalIdentities(append(alreadyExtIDs, extIDNeedLink))
		cobra.CheckErr(err)
	},
}

var createCmd = &cobra.Command{
	Use:   "create",
	Short: "create dept",
	Run: func(cmd *cobra.Command, args []string) {
		extID, err := orgmanager.ExternalIdentityParseString(args[0])
		cobra.CheckErr(err)
		target, err := extID.GetTarget()
		cobra.CheckErr(err)
		parentDept, err := target.LookupEntryDepartmentByInternalExternalIdentity(extID)
		cobra.CheckErr(err)
		fmt.Println(parentDept.Name())
		w := parentDept.(orgmanager.UnionDepartmentWriter)
		reader := bufio.NewReader(os.Stdin)
		name, _ := reader.ReadString('\n')
		name = strings.TrimRight(name, "\n")
		_, err = w.CreateSubDepartment(orgmanager.DepartmentCreateOptions{
			Name: name,
		})
		cobra.CheckErr(err)
	},
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "list child depts",
	Run: func(cmd *cobra.Command, args []string) {
		var department orgmanager.UnionDepartment
		var target orgmanager.Target
		if len(args) == 0 {
			target = selectTarget()
			department = target.RootDepartment()
		} else {
			extID, err := orgmanager.ExternalIdentityParseString(args[0])
			cobra.CheckErr(err)
			target, err = extID.GetTarget()
			cobra.CheckErr(err)
			department, err = target.LookupEntryDepartmentByInternalExternalIdentity(extID)
			cobra.CheckErr(err)
		}
		fmt.Println("now department", department.Name(), orgmanager.ExternalIdentityOfDepartment(target, department))
		for _, child := range department.SubDepartments() {
			fmt.Println(child.Name(), orgmanager.ExternalIdentityOfDepartment(target, child))
		}
	},
}

func getDepartmentFromExtIDString(extIDString string) orgmanager.UnionDepartment {
	extID, err := orgmanager.ExternalIdentityParseString(extIDString)
	cobra.CheckErr(err)
	target, err := extID.GetTarget()
	cobra.CheckErr(err)
	department, err := target.LookupEntryDepartmentByInternalExternalIdentity(extID)
	cobra.CheckErr(err)
	return department
}
