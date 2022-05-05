package orgmanager

import (
	"errors"
	"fmt"
	"strings"

	azidentity "github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	abstractions "github.com/microsoft/kiota-abstractions-go"
	azurego "github.com/microsoft/kiota-authentication-azure-go"
	msgraphsdk "github.com/microsoftgraph/msgraph-sdk-go"
	"github.com/microsoftgraph/msgraph-sdk-go/groups"
	groupsitem "github.com/microsoftgraph/msgraph-sdk-go/groups/item"
	"github.com/microsoftgraph/msgraph-sdk-go/groups/item/members"
	"github.com/microsoftgraph/msgraph-sdk-go/groups/item/owners"
	"github.com/microsoftgraph/msgraph-sdk-go/models"
	"github.com/microsoftgraph/msgraph-sdk-go/models/odataerrors"
	"github.com/microsoftgraph/msgraph-sdk-go/users"
	usersitem "github.com/microsoftgraph/msgraph-sdk-go/users/item"
	"github.com/samber/lo"
	"github.com/sethvargo/go-password/password"

	"google.golang.org/protobuf/proto"
)

type azureAD struct {
	client  *msgraphsdk.GraphServiceClient
	adapter abstractions.RequestAdapter
	config  *azureADConfig
}

var defaultAzureADUserSelect = []string{
	"businessPhones",
	"displayName",
	"givenName",
	"id",
	"jobTitle",
	"mail",
	"mobilePhone",
	"officeLocation",
	"preferredLanguage",
	"surname",
	"userPrincipalName",
	"otherMails",
}

func (a azureAD) GetTargetSlug() string {
	return a.config.Slug
}

func (a azureAD) GetPlatform() string {
	return a.config.Platform
}

type azureADConfig struct {
	Platform     string
	Slug         string
	TenantID     string
	ClientID     string
	ClientSecret string
	RootGroupID  string
	EmailDomain  string
}

func (a *azureAD) InitFormUnmarshaler(unmarshaler func(any) error) (Target, error) {
	if err := unmarshaler(&a.config); err != nil {
		return nil, err
	}
	cred, err := azidentity.NewClientSecretCredential(a.config.TenantID, a.config.ClientID, a.config.ClientSecret, nil)
	if err != nil {
		return nil, fmt.Errorf("Error creating credentials: %v\n", err)
	}

	auth, err := azurego.NewAzureIdentityAuthenticationProviderWithScopes(cred, []string{"https://graph.microsoft.com/.default"})
	if err != nil {
		return nil, fmt.Errorf("Error authentication provider: %v\n", err)
	}
	a.adapter, err = msgraphsdk.NewGraphRequestAdapter(auth)
	if err != nil {
		return nil, fmt.Errorf("Error creating adapter: %v\n", err)
	}
	a.client = msgraphsdk.NewGraphServiceClient(a.adapter)
	return a, nil
}

func (d *azureAD) GetRootDepartment() DepartmentableEntry {
	rootGroup, _ := d.client.GroupsById(d.config.RootGroupID).Get(nil)
	return &azureADGroup{
		azureAD: d,
		raw:     rootGroup,
	}
}

func (d *azureAD) GetAllUsers() (users []UserableEntry, err error) {
	resp, err := d.client.Users().Get(nil)
	if err != nil {
		return nil, err
	}
	for _, v := range resp.GetValue() {
		users = append(users, &azureADUser{
			azureAD: d,
			raw:     v,
		})
	}
	return users, err
}

func (d *azureAD) LookupEntryByExternalIdentity(extID ExternalIdentity) (Entry, error) {
	switch extID.GetEntryType() {
	case EntryTypeUser:
		return d.lookupAzureADUserByExternalIdentity(extID)
	case EntryTypeDept:
		return d.lookupAzureADGroupByExternalIdentity(extID)
	default:
		return nil, errors.New("unsupported entry type")
	}
}

func (d *azureAD) LookupEntryUserByExternalIdentity(extID ExternalIdentity) (UserEntryExtIDStoreable, error) {
	return d.lookupAzureADUserByExternalIdentity(extID)
}

func (d *azureAD) lookupAzureADUserByExternalIdentity(extID ExternalIdentity) (*azureADUser, error) {
	if extID.GetTargetSlug() == d.config.Slug && extID.GetPlatform() == d.config.Platform {
		return d.lookupAzureADUserByInternalExternalIdentity(extID)
	}
	requestParameters := &users.UsersRequestBuilderGetQueryParameters{
		Select: defaultAzureADUserSelect,
		Filter: proto.String(fmt.Sprintf("otherMails/any(id:id eq '%s')", extID)),
	}
	resp, err := d.client.Users().Get(&users.UsersRequestBuilderGetOptions{
		QueryParameters: requestParameters,
	})
	if err != nil {
		return nil, err
	}
	if len(resp.GetValue()) != 1 {
		return nil, errors.New("cannot identitify user")
	}
	return &azureADUser{
		azureAD: d,
		raw:     resp.GetValue()[0],
	}, nil
}

func (d *azureAD) LookupEntryDepartmentByExternalIdentity(extID ExternalIdentity) (DepartmentEntryExtIDStoreable, error) {
	return d.lookupAzureADGroupByExternalIdentity(extID)
}

func (d *azureAD) lookupAzureADGroupByExternalIdentity(extID ExternalIdentity) (*azureADGroup, error) {
	if extID.GetTargetSlug() == d.config.Slug && extID.GetPlatform() == d.config.Platform {
		return d.lookupAzureADGroupByExternalIdentity(extID)
	}
	requestParameters := &groups.GroupsRequestBuilderGetQueryParameters{
		Search: proto.String(fmt.Sprintf(`"description:%s"`, extID)),
	}
	resp, err := d.client.Groups().Get(&groups.GroupsRequestBuilderGetOptions{
		QueryParameters: requestParameters,
		Headers:         map[string]string{"ConsistencyLevel": "eventual"},
	})
	if err != nil {
		return nil, err
	}
	if len(resp.GetValue()) != 1 {
		return nil, errors.New("cannot identitify group")
	}
	return &azureADGroup{
		azureAD: d,
		raw:     resp.GetValue()[0],
	}, nil
}

func (d *azureAD) LookupEntryUserByInternalExternalIdentity(internalExtID ExternalIdentity) (UserableEntry, error) {
	return d.lookupAzureADUserByInternalExternalIdentity(internalExtID)
}

func (d *azureAD) lookupAzureADUserByInternalExternalIdentity(internalExtID ExternalIdentity) (*azureADUser, error) {
	user, err := d.client.UsersById(internalExtID.GetEntryID()).Get(&usersitem.UserItemRequestBuilderGetOptions{
		QueryParameters: &usersitem.UserItemRequestBuilderGetQueryParameters{
			Select: defaultAzureADUserSelect,
		},
	})
	if err != nil {
		return nil, err
	}
	return &azureADUser{azureAD: d, raw: user}, nil
}

func (d *azureAD) LookupEntryDepartmentByInternalExternalIdentity(internalExtID ExternalIdentity) (DepartmentableEntry, error) {
	return d.lookupAzureADGroupByInternalExternalIdentity(internalExtID)
}

func (d *azureAD) lookupAzureADGroupByInternalExternalIdentity(internalExtID ExternalIdentity) (*azureADGroup, error) {
	group, err := d.client.GroupsById(internalExtID.GetEntryID()).Get(nil)
	if err != nil {
		return nil, err
	}
	return &azureADGroup{azureAD: d, raw: group}, nil
}

func (d *azureAD) CreateUser(options Userable) (UserableEntry, error) {
	newUser := models.NewUser()
	newUser.SetAccountEnabled(proto.Bool(true))
	newUser.SetDisplayName(proto.String(options.GetName()))
	// newUser.SetMailNickname(proto.String(options.GetMailNickname()))
	newUser.SetMobilePhone(proto.String(options.GetPhone()))
	identities := make([]models.ObjectIdentityable, 0)
	phoneIdentity := models.NewObjectIdentity()
	phoneIdentity.SetSignInType(proto.String("federated"))
	phoneIdentity.SetIssuer(proto.String("phone"))
	phoneIdentity.SetIssuerAssignedId(proto.String(options.GetPhone()))
	identities = append(identities, phoneIdentity)
	newUser.SetIdentities(identities)
	newPasswordProfile := models.NewPasswordProfile()
	newPasswordProfile.SetForceChangePasswordNextSignIn(proto.Bool(false))
	newPassword, err := password.Generate(32, 10, 10, false, false)
	if err != nil {
		return nil, err
	}
	newPasswordProfile.SetPassword(proto.String(newPassword))
	newUser.SetPasswordProfile(newPasswordProfile)
	// newUser.SetUserPrincipalName(proto.String(fmt.Sprintf("%s@%s", options.GetMailNickname(), d.config.EmailDomain)))
	user, err := d.client.Users().Post(&users.UsersRequestBuilderPostOptions{
		Body: newUser,
	})
	if err != nil {
		return nil, err
	}
	return &azureADUser{azureAD: d, raw: user}, err
}

type AzureADGroupRole string

const (
	AzureADGroupRoleOwner  AzureADGroupRole = "owners"
	AzureADGroupRoleMember AzureADGroupRole = "members"
)

func castAzureADGroupRoleFromDepartmentUserRole(role DepartmentUserRole) AzureADGroupRole {
	return map[DepartmentUserRole]AzureADGroupRole{
		DepartmentUserRoleAdmin:  AzureADGroupRoleOwner,
		DepartmentUserRoleMember: AzureADGroupRoleMember,
	}[role]
}

func (g *azureAD) postAddToAzureADGroup(role AzureADGroupRole, groupID, objectID string) error {
	opts := new(groups.GroupsRequestBuilderPostOptions)
	opts.Body = models.NewGroup()
	opts.Body.SetAdditionalData(map[string]any{
		"@odata.id": proto.String("https://graph.microsoft.com/v1.0/directoryObjects/" + objectID),
	})
	opts.Headers = make(map[string]string)
	opts.Headers["Content-Type"] = "application/json"
	rawUrl := fmt.Sprintf("https://graph.microsoft.com/v1.0/groups/%s/%s/$ref", groupID, role)
	requestBuilder := groups.NewGroupsRequestBuilder(rawUrl, g.adapter)
	return azureGroupPostWithNoContent(requestBuilder, g.adapter, opts)
}

type azureADGroup struct {
	*azureAD
	raw models.Groupable
}

func (g *azureADGroup) GetTarget() Target {
	return g.azureAD
}

func (g azureADGroup) GetID() (departmentId string) {
	return *g.raw.GetId()
}

func (g azureADGroup) GetName() (name string) {
	return *g.raw.GetDisplayName()
}

func (g azureADGroup) GetDescription() (name string) {
	return *g.raw.GetDescription()
}

func (g azureADGroup) GetChildDepartments() (departments []DepartmentableEntry) {
	groups, _ := g.client.GroupsById(*g.raw.GetId()).Members().Get(nil)
	for _, v := range groups.GetValue() {
		if *v.GetAdditionalData()["@odata.type"].(*string) == "#microsoft.graph.group" {
			group, _ := g.client.GroupsById(*v.GetId()).Get(nil)
			departments = append(departments, &azureADGroup{
				azureAD: g.azureAD,
				raw:     group,
			})
		}
	}
	return departments
}

func (g *azureADGroup) CreateChildDepartment(department Departmentable) (DepartmentableEntry, error) {
	newGroup := models.NewGroup()
	newGroup.SetDisplayName(proto.String(department.GetName()))
	newGroup.SetMailEnabled(proto.Bool(false))
	newGroup.SetMailNickname(proto.String("placeholder"))
	newGroup.SetSecurityEnabled(proto.Bool(true))
	newGroupable, err := g.client.Groups().Post(&groups.GroupsRequestBuilderPostOptions{
		Body: newGroup,
	})
	if err != nil {
		return nil, fmt.Errorf("Create group faild: %s", err)
	}
	err = g.postAddToAzureADGroup(AzureADGroupRoleMember, *g.raw.GetId(), *newGroupable.GetId())
	if err != nil {
		err = fmt.Errorf("Link group membership faild: %s", err)
	}
	return &azureADGroup{
		azureAD: g.azureAD,
		raw:     newGroupable,
	}, err
}

//case the func on doc is unavailable
func azureGroupPostWithNoContent(m *groups.GroupsRequestBuilder, requestAdapter abstractions.RequestAdapter, options *groups.GroupsRequestBuilderPostOptions) error {
	requestInfo, err := m.CreatePostRequestInformation(options)
	if err != nil {
		return err
	}
	errorMapping := abstractions.ErrorMappings{
		"4XX": odataerrors.CreateODataErrorFromDiscriminatorValue,
		"5XX": odataerrors.CreateODataErrorFromDiscriminatorValue,
	}
	err = requestAdapter.SendNoContentAsync(requestInfo, nil, errorMapping)
	if err != nil {
		return err
	}
	return nil
}

func azureGroupDeleteWithNoContent(m *groups.GroupsRequestBuilder, requestAdapter abstractions.RequestAdapter, options *groups.GroupsRequestBuilderPostOptions) error {
	requestInfo, err := m.CreatePostRequestInformation(options)
	if err != nil {
		return err
	}
	errorMapping := abstractions.ErrorMappings{
		"4XX": odataerrors.CreateODataErrorFromDiscriminatorValue,
		"5XX": odataerrors.CreateODataErrorFromDiscriminatorValue,
	}
	err = requestAdapter.SendNoContentAsync(requestInfo, nil, errorMapping)
	if err != nil {
		return err
	}
	return nil
}

func (g *azureADGroup) GetUsers() (users []UserableEntry) {
	groups, _ := g.client.GroupsById(*g.raw.GetId()).Members().Get(&members.MembersRequestBuilderGetOptions{
		QueryParameters: &members.MembersRequestBuilderGetQueryParameters{
			Select: defaultAzureADUserSelect,
		},
	})
	for _, v := range groups.GetValue() {
		if *v.GetAdditionalData()["@odata.type"].(*string) == "#microsoft.graph.user" {
			user, _ := g.client.UsersById(*v.GetId()).Get(nil)
			users = append(users, &azureADUser{
				azureAD: g.azureAD,
				raw:     user,
			})
		}
	}
	return users
}

func (g *azureADGroup) Admins() (users []UserableEntry) {
	groups, _ := g.client.GroupsById(*g.raw.GetId()).Owners().Get(&owners.OwnersRequestBuilderGetOptions{
		QueryParameters: &owners.OwnersRequestBuilderGetQueryParameters{
			Select: defaultAzureADUserSelect,
		},
	})
	for _, v := range groups.GetValue() {
		if *v.GetAdditionalData()["@odata.type"].(*string) == "#microsoft.graph.user" {
			user, _ := g.client.UsersById(*v.GetId()).Get(nil)
			users = append(users, &azureADUser{
				azureAD: g.azureAD,
				raw:     user,
			})
		}
	}
	return users
}

func (g *azureADGroup) AddToDepartment(options DepartmentModifyUserOptions, extID ExternalIdentity) error {
	azureADGroupRole := castAzureADGroupRoleFromDepartmentUserRole(options.Role)
	return g.postAddToAzureADGroup(azureADGroupRole, *g.raw.GetId(), extID.GetEntryID())
}

func (g *azureADGroup) RemoveFromDepartment(options DepartmentModifyUserOptions, extID ExternalIdentity) error {
	panic("not implemented") // TODO: Implement
}

func (u *azureADGroup) GetExternalIdentities() ExternalIdentities {
	desc := ""
	if u.raw.GetDescription() != nil {
		desc = *u.raw.GetDescription()
	}
	return ExternalIdentitiesFromStringList(strings.Split(desc, ","))
}

func (u azureADGroup) SetExternalIdentities(extIDs ExternalIdentities) error {
	extIDStrList := make([]string, 0)
	for _, extID := range extIDs {
		if !lo.Contains(extIDStrList, string(extID)) {
			extIDStrList = append(extIDStrList, string(extID))
		}
	}
	newGroup := models.NewGroup()
	newGroup.SetDescription(proto.String(strings.Join(extIDStrList, ",")))
	return u.client.GroupsById(*u.raw.GetId()).Patch(&groupsitem.GroupItemRequestBuilderPatchOptions{
		Body: newGroup,
	})
}

type azureADUser struct {
	*azureAD
	raw models.Userable
}

func (u azureADUser) GetID() string {
	return *u.raw.GetId()
}

func (u *azureADUser) GetTarget() Target {
	return u.azureAD
}

func (u azureADUser) GetName() string {
	return *u.raw.GetDisplayName()
}

func (u azureADUser) GetEmail() string {
	return *u.raw.GetMail()
}

func (u azureADUser) GetPhone() string {
	return *u.raw.GetMobilePhone()
}

func (u azureADUser) GetExternalIdentities() ExternalIdentities {
	return ExternalIdentitiesFromStringList(u.raw.GetOtherMails())
}

func (u azureADUser) SetExternalIdentities(extIDs ExternalIdentities) error {
	newOtherMails := make([]string, 0)
	for _, mail := range u.raw.GetOtherMails() {
		if _, err := ExternalIdentityParseString(mail); err != nil {
			newOtherMails = append(newOtherMails, mail)
		}
	}
	for _, extID := range extIDs {
		if !lo.Contains(newOtherMails, string(extID)) {
			newOtherMails = append(newOtherMails, string(extID))
		}
	}
	newUser := models.NewUser()
	newUser.SetOtherMails(newOtherMails)
	return u.client.UsersById(*u.raw.GetId()).Patch(&usersitem.UserItemRequestBuilderPatchOptions{
		Body: newUser,
	})
}

func (u azureADUser) GetEmailSet() (emails []string) {
	for _, mail := range u.raw.GetOtherMails() {
		if _, err := ExternalIdentityParseString(mail); err != nil {
			emails = append(emails, mail)
		}
	}
	return emails
}

func (u azureADUser) AddToEmailSet(email string) error {
	if lo.Contains(u.raw.GetOtherMails(), email) {
		return errors.New("already has email " + email)
	}
	newUser := models.NewUser()
	newUser.SetOtherMails(append(u.raw.GetOtherMails(), email))
	return u.client.UsersById(*u.raw.GetId()).Patch(&usersitem.UserItemRequestBuilderPatchOptions{
		Body: newUser,
	})
}

func (u azureADUser) DeleteFromEmailSet(email string) error {
	if !lo.Contains(u.raw.GetOtherMails(), email) {
		return errors.New("donot have this email " + email)
	}
	newEmails := lo.Filter(u.raw.GetOtherMails(), func(v string, i int) bool {
		return u.raw.GetOtherMails()[i] == email
	})
	newUser := models.NewUser()
	newUser.SetOtherMails(newEmails)
	return u.client.UsersById(*u.raw.GetId()).Patch(&usersitem.UserItemRequestBuilderPatchOptions{
		Body: newUser,
	})
}
