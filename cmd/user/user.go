package user

import (
	"fmt"

	"github.com/org-tools/manager"
	"github.com/org-tools/manager/cmd/base"
	"github.com/spf13/cobra"
)

func init() {
	Cmd.AddCommand(linkCmd, infoCmd, createCmd, listCmd, syncCmd)
}

var Cmd = &cobra.Command{
	Use:   "user",
	Short: "user management",
}

var infoCmd = &cobra.Command{
	Use:   "info",
	Short: "show user info with extID",
	Run: func(cmd *cobra.Command, args []string) {
		extID, err := manager.ExternalIdentityParseString(args[0])
		cobra.CheckErr(err)
		if extID.GetEntryType() != manager.EntryTypeUser {
			fmt.Println("extID not type user")
			return
		}
		target, err := manager.GetTargetByPlatformAndSlug(extID.GetPlatform(), extID.GetTargetSlug())
		cobra.CheckErr(err)
		user, err := target.LookupEntryUserByInternalExternalIdentity(extID)
		cobra.CheckErr(err)
		fmt.Println(user.GetID(), user.GetName())
		fmt.Println(manager.ExternalIdentityOfUser(target, user))

		if entryCenter, ok := target.(manager.EntryCenter); ok {
			user, err := entryCenter.LookupEntryUserByExternalIdentity(extID)
			cobra.CheckErr(err)
			for _, extID := range user.GetExternalIdentities() {
				target, err := extID.GetTarget()
				cobra.CheckErr(err)
				linkedUser, err := target.LookupEntryUserByInternalExternalIdentity(extID)
				cobra.CheckErr(err)
				fmt.Println(linkedUser.GetName(), manager.ExternalIdentityOfUser(target, linkedUser))
			}
		}
	},
}

var linkCmd = &cobra.Command{
	Use:   "link",
	Short: "link user form to",
	Run: func(cmd *cobra.Command, args []string) {
		extIDNeedLink, err := manager.ExternalIdentityParseString(args[0])
		cobra.CheckErr(err)
		if extIDNeedLink.GetEntryType() != manager.EntryTypeDept {
			fmt.Println("extIDNeedLink not type user")
			return
		}
		target, err := manager.GetTargetByPlatformAndSlug(extIDNeedLink.GetPlatform(), extIDNeedLink.GetTargetSlug())
		cobra.CheckErr(err)
		extIDLinkTo, err := manager.ExternalIdentityParseString(args[1])
		cobra.CheckErr(err)
		if extIDLinkTo.GetEntryType() != manager.EntryTypeDept {
			fmt.Println("extIDLinkTo not type user")
			return
		}
		targetShouldBeEntryCenter, err := manager.GetTargetByPlatformAndSlug(extIDLinkTo.GetPlatform(), extIDLinkTo.GetTargetSlug())
		cobra.CheckErr(err)
		_, err = target.LookupEntryUserByInternalExternalIdentity(extIDNeedLink)
		cobra.CheckErr(err)

		if entryCenter, ok := targetShouldBeEntryCenter.(manager.EntryCenter); ok {
			user, err := entryCenter.LookupEntryUserByExternalIdentity(extIDNeedLink)
			if err != nil {
				fmt.Println(err)
			}
			if user != nil && err == nil {
				fmt.Println("already linked")
				return
			}
		}

		user, err := targetShouldBeEntryCenter.LookupEntryUserByInternalExternalIdentity(extIDLinkTo)
		cobra.CheckErr(err)
		userExtIDStoreable := user.(manager.EntryExtIDStoreable)
		alreadyExtIDs := userExtIDStoreable.GetExternalIdentities()
		fmt.Println(alreadyExtIDs)
		err = userExtIDStoreable.SetExternalIdentities(append(alreadyExtIDs, extIDNeedLink))
		cobra.CheckErr(err)
	},
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "list users",
	Run: func(cmd *cobra.Command, args []string) {
		target, _ := base.SelectTarget()
		users, err := target.GetAllUsers()
		cobra.CheckErr(err)
		for _, user := range users {
			fmt.Println(user.GetName(), manager.ExternalIdentityOfUser(target, user))
		}
	},
}

var createCmd = &cobra.Command{
	Use:   "create",
	Short: "create user",
	Run: func(cmd *cobra.Command, args []string) {
		target, _ := base.SelectTarget()
		user, err := target.(manager.UserWriteable).CreateUser(manager.User{
			Name:  base.InputStringWithHint("Name"),
			Email: base.InputStringWithHint("Email"),
		})
		cobra.CheckErr(err)
		fmt.Println(user.GetName(), manager.ExternalIdentityOfUser(target, user))
	},
}

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "sync users",
	Run: func(cmd *cobra.Command, args []string) {
		source, key := base.SelectTarget()
		targetShouldBeUserWriteable, _ := base.SelectTarget(key)
		if source == targetShouldBeUserWriteable {
			fmt.Println("target is same as source")
			return
		}
		userWriteable, ok := targetShouldBeUserWriteable.(manager.UserWriteable)
		if !ok {
			fmt.Println("target should be UserWriteable")
			return
		}
		users, err := source.GetAllUsers()
		cobra.CheckErr(err)
		fmt.Println("Total", len(users))
		users = manager.Uniq(users)
		fmt.Println("Uniq", len(users))
		for _, user := range users {
			get, err := userWriteable.LookupUser(user)
			if err != nil {
				fmt.Println(err)
			}
			if get != nil {
				fmt.Println("merge", get.GetName(), user.GetName())
				if mergable, ok := get.(manager.UserableCanMerge); ok {
					err = mergable.Merge(user)
					if err != nil {
						fmt.Println(err)
					}
				}
				continue
			}
			_, err = userWriteable.CreateUser(user)
			if err != nil {
				fmt.Println(err)
			}
		}
	},
}
