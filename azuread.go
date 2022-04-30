package orgmanager

import (
	"fmt"

	azidentity "github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	azurego "github.com/microsoft/kiota-authentication-azure-go"

	msgraphsdk "github.com/microsoftgraph/msgraph-sdk-go"
	"github.com/microsoftgraph/msgraph-sdk-go/models"
)

type AzureAD struct {
	client *msgraphsdk.GraphServiceClient
	config *azureADConfig
}

type azureADConfig struct {
	TenantID     string
	ClientID     string
	ClientSecret string
	RootGroupID  string
}

func (a *AzureAD) InitFormUnmarshaler(unmarshaler func(any) error) (Target, error) {
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
	adapter, err := msgraphsdk.NewGraphRequestAdapter(auth)
	if err != nil {
		return nil, fmt.Errorf("Error creating adapter: %v\n", err)
	}
	a.client = msgraphsdk.NewGraphServiceClient(adapter)
	return a, nil
}

func (d AzureAD) RootDepartment() UnionDepartment {
	rootGroup, _ := d.client.GroupsById(d.config.RootGroupID).Get(nil)
	return &azureGroup{
		client: d.client,
		raw:    rootGroup,
	}
}

type azureGroup struct {
	client *msgraphsdk.GraphServiceClient
	raw    models.Groupable
}

func (g azureGroup) Name() (name string) {
	return *g.raw.GetDisplayName()
}

func (g azureGroup) SubDepartments() (departments []UnionDepartment) {
	groups, _ := g.client.GroupsById(*g.raw.GetId()).Members().Get(nil)
	for _, v := range groups.GetValue() {
		if *v.GetAdditionalData()["@odata.type"].(*string) == "#microsoft.graph.group" {
			group, _ := g.client.GroupsById(*v.GetId()).Get(nil)
			departments = append(departments, &azureGroup{
				client: g.client,
				raw:    group,
			})
		}
	}
	return departments
}

func (g azureGroup) Users() (users []UnionUser) {
	groups, _ := g.client.GroupsById(*g.raw.GetId()).Members().Get(nil)
	for _, v := range groups.GetValue() {
		if *v.GetAdditionalData()["@odata.type"].(*string) == "#microsoft.graph.user" {
			user, _ := g.client.UsersById(*v.GetId()).Get(nil)
			users = append(users, &azureADUser{
				client: g.client,
				raw:    user,
			})
		}
	}
	return users
}

type azureADUser struct {
	client *msgraphsdk.GraphServiceClient
	raw    models.Userable
}

func (u azureADUser) UserId() string {
	return *u.raw.GetId()
}

func (u azureADUser) UserName() string {
	return *u.raw.GetDisplayName()
}

func (u azureADUser) UserEmail() string {
	return *u.raw.GetMail()
}
