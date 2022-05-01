package orgmanager

import (
	"context"
	"fmt"

	"github.com/larksuite/oapi-sdk-go/core"
	"github.com/larksuite/oapi-sdk-go/core/config"
	contact "github.com/larksuite/oapi-sdk-go/service/contact/v3"
)

type Feishu struct {
	oapiConfig *config.Config
	config     *feishuConfig
}

func (d Feishu) GetTargetSlug() string {
	return d.config.Slug
}

func (d Feishu) GetPlatform() string {
	return d.config.Platform
}

func (f *Feishu) LookupEntryUserByInternalExternalIdentity(internalExtID ExternalIdentity) (UnionUser, error) {
	contactService := contact.NewService(f.oapiConfig)
	coreCtx := core.WrapContext(context.Background())
	req := contactService.Users.Get(coreCtx)
	req.SetUserId(internalExtID.GetEntryID())
	req.SetUserIdType("user_id")
	resp, err := req.Do()
	if err != nil {
		return nil, err
	}
	return &feishuUser{
		target: f,
		raw:    resp.User,
	}, nil
}

func (f *Feishu) LookupEntryDepartmentByInternalExternalIdentity(internalExtID ExternalIdentity) (UnionDepartment, error) {
	contactService := contact.NewService(f.oapiConfig)
	coreCtx := core.WrapContext(context.Background())
	req := contactService.Departments.Get(coreCtx)
	req.SetDepartmentId(internalExtID.GetEntryID())
	req.SetDepartmentIdType("department_id")
	resp, err := req.Do()
	if err != nil {
		return nil, err
	}
	return &feishuDepartment{
		target: f,
		raw:    resp.Department,
	}, nil
}

type feishuConfig struct {
	Platform  string
	Slug      string
	AppID     string
	AppSecret string
}

func (f Feishu) InitFormUnmarshaler(unmarshaler func(any) error) (Target, error) {
	err := unmarshaler(&f.config)
	if err != nil {
		return nil, err
	}
	appSettings := core.NewInternalAppSettings(
		core.SetAppCredentials(f.config.AppID, f.config.AppSecret),
	)
	f.oapiConfig = core.NewConfig(core.DomainFeiShu, appSettings, core.SetLoggerLevel(core.LoggerLevelError))
	return &f, nil
}

func (f *Feishu) RootDepartment() UnionDepartment {
	contactService := contact.NewService(f.oapiConfig)
	coreCtx := core.WrapContext(context.Background())
	req := contactService.Departments.List(coreCtx)
	req.SetFetchChild(true)
	resp, _ := req.Do()
	return feishuDepartment{
		target: f,
		raw:    resp.Items[0],
	}
}

type feishuDepartment struct {
	target *Feishu
	raw    *contact.Department
}

func (d feishuDepartment) Name() (name string) {
	if d.raw == nil || d.raw.DepartmentId == "0" {
		return "root"
	}
	return d.raw.Name
}

func (d feishuDepartment) DepartmentID() (departmentId string) {
	return d.raw.DepartmentId
}

func (d feishuDepartment) SubDepartments() (departments []UnionDepartment) {
	contactService := contact.NewService(d.target.oapiConfig)
	coreCtx := core.WrapContext(context.Background())
	req := contactService.Departments.List(coreCtx)
	req.SetParentDepartmentId(d.raw.DepartmentId)
	req.SetDepartmentIdType("department_id")
	resp, err := req.Do()
	if err != nil {
		fmt.Println(err)
	}
	for _, v := range resp.Items {
		departments = append(departments, &feishuDepartment{
			target: d.target,
			raw:    v,
		})
	}
	return departments
}

func (d feishuDepartment) Users() (users []UnionUser) {
	contactService := contact.NewService(d.target.oapiConfig)
	coreCtx := core.WrapContext(context.Background())
	req := contactService.Users.List(coreCtx)
	req.SetDepartmentId(d.raw.DepartmentId)
	req.SetDepartmentIdType("department_id")
	resp, _ := req.Do()
	for _, v := range resp.Items {
		users = append(users, &feishuUser{
			target: d.target,
			raw:    v,
		})
	}
	return users
}

type feishuUser struct {
	target *Feishu
	raw    *contact.User
}

func (u feishuUser) UserId() (userId string) {
	return u.raw.UserId
}

func (u feishuUser) UserName() (name string) {
	return u.raw.Name
}
