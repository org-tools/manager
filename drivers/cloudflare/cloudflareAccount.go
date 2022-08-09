package orgmanager

import (
	"context"
	"errors"
	"fmt"

	"github.com/cloudflare/cloudflare-go"
	. "github.com/org-tools/manager"
)

type cloudflareDNS struct {
	api    *cloudflare.API
	config *cloudflareConfig
}

type cloudflareConfig struct {
	Platform  string
	Slug      string
	ApiKey    string
	ApiEmail  string
	ApiToken  string
	Account   string
	AccountID string
}

func (c *cloudflareDNS) InitFormUnmarshaler(unmarshaler func(any) error) (Target, error) {
	err := unmarshaler(&c.config)
	if err != nil {
		return nil, err
	}
	switch {
	case c.config.ApiToken != "":
		c.api, err = cloudflare.NewWithAPIToken(c.config.ApiToken)
	case c.config.ApiKey != "" && c.config.ApiEmail != "":
		c.api, err = cloudflare.New(c.config.ApiKey, c.config.ApiEmail)
	default:
		return nil, errors.New("should have api-token or api-key with api-email")
	}
	return c, err
}

func (c *cloudflareDNS) LookupEntryUserByInternalExternalIdentity(internalExtID ExternalIdentity) (UserableEntry, error) {
	if c.config.AccountID != "" {
		c.GetRootDepartment()
	}
	member, err := c.api.AccountMember(context.Background(), c.config.AccountID, internalExtID.GetEntryID())
	if err != nil {
		return nil, err
	}
	return &cloudflareAccountMember{cloudflareDNS: c, member: member}, nil
}

func (c *cloudflareDNS) LookupEntryDepartmentByInternalExternalIdentity(internalExtID ExternalIdentity) (DepartmentableEntry, error) {
	account, _, err := c.api.Account(context.Background(), (internalExtID.GetEntryID()))
	if err != nil {
		return nil, err
	}
	return &cloudflareAccount{cloudflareDNS: c, account: account}, nil
}

func (c *cloudflareDNS) GetTarget() Target {
	return c
}

func (c cloudflareDNS) GetTargetSlug() string {
	return c.config.Slug
}

func (c cloudflareDNS) GetPlatform() string {
	return c.config.Platform
}

func (c *cloudflareDNS) GetRootDepartment() (DepartmentableEntry, error) {
	if c.config.AccountID == "" {
		params := cloudflare.AccountsListParams{}
		accounts, _, _ := c.api.Accounts(context.Background(), params)
		for _, account := range accounts {
			if account.Name == c.config.Account {
				c.config.AccountID = account.ID
				return &cloudflareAccount{cloudflareDNS: c, account: account}, nil
			}
		}
		return nil, fmt.Errorf("cloudflare account '%s' not found", c.config.Account)
	}

	account, _, _ := c.api.Account(context.Background(), c.config.AccountID)
	return &cloudflareAccount{cloudflareDNS: c, account: account}, nil
}

func (c cloudflareDNS) GetAllUsers() (users []UserableEntry, err error) {
	department, err := c.GetRootDepartment()
	if err != nil {
		return users, err
	}
	return department.GetUsers()
}

type cloudflareAccountMember struct {
	*cloudflareDNS
	member cloudflare.AccountMember
}

func (m cloudflareAccountMember) GetID() string {
	return m.member.ID
}

func (m cloudflareAccountMember) GetName() string {
	return fmt.Sprint(m.member.User.FirstName, m.member.User.LastName)
}

func (m cloudflareAccountMember) GetEmail() (email string) {
	return m.member.User.Email
}

func (m cloudflareAccountMember) GetPhone() (phone string) {
	return ""
}

type cloudflareAccount struct {
	*cloudflareDNS
	account cloudflare.Account
}

func (a cloudflareAccount) GetID() string {
	return a.account.ID
}

func (a cloudflareAccount) GetName() string {
	return a.account.Name
}

func (a cloudflareAccount) GetDescription() string {
	return ""
}

func (z cloudflareAccount) GetChildDepartments() (departments []DepartmentableEntry) {
	return departments
}

func (z cloudflareAccount) CreateChildDepartment(department Departmentable) (DepartmentableEntry, error) {
	panic("not implemented") // TODO: Implement
}

func (z *cloudflareAccount) GetUsers() (users []UserableEntry, err error) {
	opts := cloudflare.PaginationOptions{}
	members, _, err := z.api.AccountMembers(context.Background(), z.account.ID, opts)
	if err != nil {
		return
	}
	for _, member := range members {
		users = append(users, &cloudflareAccountMember{
			cloudflareDNS: z.cloudflareDNS,
			member:        member,
		})
	}
	return
}
