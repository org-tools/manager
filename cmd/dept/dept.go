package dept

import (
	"fmt"

	orgmanager "github.com/hduhelp/org-manager"
	"github.com/hduhelp/org-manager/cmd/base"
	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"
)

func init() {
	Cmd.AddCommand(infoCmd, createCmd, linkCmd, listCmd)
}

var Cmd = &cobra.Command{
	Use:   "dept",
	Short: "dept management",
	Run: func(cmd *cobra.Command, args []string) {
		target := base.SelectTarget()
		nowDepartment := target.GetRootDepartment()
		fmt.Println(orgmanager.ExternalIdentityOfDepartment(target, nowDepartment))
		for _, v := range nowDepartment.GetUsers() {
			fmt.Println(orgmanager.ExternalIdentityOfUser(target, v), v.GetName())
		}
		for {
			depts := nowDepartment.GetChildDepartments()
			if len(depts) == 0 {
				return
			}
			deptsName := make([]string, 0)
			for _, v := range depts {
				deptsName = append(deptsName, v.GetName())
			}
			prompt := promptui.Select{
				Label: "Select Department",
				Items: deptsName,
			}
			_, deptName, err := prompt.Run()
			cobra.CheckErr(err)
			for _, v := range depts {
				if v.GetName() == deptName {
					nowDepartment = v
				}
			}
			for _, v := range nowDepartment.GetUsers() {
				fmt.Println(orgmanager.ExternalIdentityOfUser(target, v), v.GetName())
			}
			fmt.Println(orgmanager.ExternalIdentityOfDepartment(target, nowDepartment))
		}
	},
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
		fmt.Println(dept.GetID(), dept.GetName())
		fmt.Println(orgmanager.ExternalIdentityOfDepartment(target, dept))

		if entryCenter, ok := target.(orgmanager.EntryCenter); ok {
			dept, err := entryCenter.LookupEntryDepartmentByExternalIdentity(extID)
			cobra.CheckErr(err)
			for _, extID := range dept.GetExternalIdentities() {
				target, err := extID.GetTarget()
				cobra.CheckErr(err)
				linkedDept, err := target.LookupEntryDepartmentByInternalExternalIdentity(extID)
				cobra.CheckErr(err)
				fmt.Println(linkedDept.GetName(), orgmanager.ExternalIdentityOfDepartment(target, linkedDept))
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
		fmt.Println(parentDept.GetName())
		_, err = parentDept.CreateChildDepartment(&orgmanager.Department{
			Name: base.InputStringWithHint("Name"),
		})
		cobra.CheckErr(err)
	},
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "list child depts",
	Run: func(cmd *cobra.Command, args []string) {
		var department orgmanager.DepartmentableEntry
		var target orgmanager.Target
		if len(args) == 0 {
			target = base.SelectTarget()
			department = target.GetRootDepartment()
		} else {
			extID, err := orgmanager.ExternalIdentityParseString(args[0])
			cobra.CheckErr(err)
			target, err = extID.GetTarget()
			cobra.CheckErr(err)
			department, err = target.LookupEntryDepartmentByInternalExternalIdentity(extID)
			cobra.CheckErr(err)
		}
		fmt.Println("now department", department.GetName(), orgmanager.ExternalIdentityOfDepartment(target, department))
		for _, child := range department.GetChildDepartments() {
			fmt.Println(child.GetName(), orgmanager.ExternalIdentityOfDepartment(target, child))
		}
	},
}

func getDepartmentFromExtIDString(extIDString string) orgmanager.Departmentable {
	extID, err := orgmanager.ExternalIdentityParseString(extIDString)
	cobra.CheckErr(err)
	target, err := extID.GetTarget()
	cobra.CheckErr(err)
	department, err := target.LookupEntryDepartmentByInternalExternalIdentity(extID)
	cobra.CheckErr(err)
	return department
}
