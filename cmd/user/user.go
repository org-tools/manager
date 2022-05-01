package user

import (
	"fmt"

	orgmanager "github.com/hduhelp/org-manager"
	"github.com/spf13/cobra"
)

func init() {
	Cmd.AddCommand(linkCmd, infoCmd)
}

var Cmd = &cobra.Command{
	Use:   "user",
	Short: "user management",
	Run: func(cmd *cobra.Command, args []string) {

	},
}

var infoCmd = &cobra.Command{
	Use:   "info",
	Short: "show user info with extID",
	Run: func(cmd *cobra.Command, args []string) {
		extID, err := orgmanager.ExternalIdentityParseString(args[0])
		cobra.CheckErr(err)
		target, err := orgmanager.GetTargetByPlatformAndSlug(extID.GetPlatform(), extID.GetTargetSlug())
		cobra.CheckErr(err)
		user, err := target.LookupEntryUserByInternalExternalIdentity(extID)
		cobra.CheckErr(err)
		fmt.Println(user.UserId(), user.UserName())
		fmt.Println(orgmanager.ExternalIdentityOfUser(target, user))

		if entryCenter, ok := target.(orgmanager.EntryCenter); ok {
			user, err := entryCenter.LookupEntryUserByExternalIdentity(extID)
			cobra.CheckErr(err)
			fmt.Println(err)
			fmt.Println(user)
			fmt.Println(user.GetExternalIdentities())
		}
	},
}

var linkCmd = &cobra.Command{
	Use:   "link",
	Short: "link user form to",
	Run: func(cmd *cobra.Command, args []string) {
		extIDNeedLink, err := orgmanager.ExternalIdentityParseString(args[0])
		cobra.CheckErr(err)
		target, err := orgmanager.GetTargetByPlatformAndSlug(extIDNeedLink.GetPlatform(), extIDNeedLink.GetTargetSlug())
		cobra.CheckErr(err)
		extIDLinkTo, err := orgmanager.ExternalIdentityParseString(args[1])
		cobra.CheckErr(err)
		targetShouldBeEntryCenter, err := orgmanager.GetTargetByPlatformAndSlug(extIDLinkTo.GetPlatform(), extIDLinkTo.GetTargetSlug())
		cobra.CheckErr(err)
		_, err = target.LookupEntryUserByInternalExternalIdentity(extIDNeedLink)
		cobra.CheckErr(err)

		if entryCenter, ok := targetShouldBeEntryCenter.(orgmanager.EntryCenter); ok {
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
		userExtIDStoreable := user.(orgmanager.EntryExtIDStoreable)
		alreadyExtIDs := userExtIDStoreable.GetExternalIdentities()
		fmt.Println(alreadyExtIDs)
		err = userExtIDStoreable.SetExternalIdentities(append(alreadyExtIDs, extIDNeedLink))
		cobra.CheckErr(err)
	},
}
