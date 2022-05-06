package orgmanager

import (
	"context"
	"fmt"

	"github.com/larksuite/oapi-sdk-go/core"
	"github.com/larksuite/oapi-sdk-go/core/config"
	contact "github.com/larksuite/oapi-sdk-go/service/contact/v3"
	"github.com/samber/lo"
)

const (
	feishuDefaultUserIdType       = "user_id"
	feishuDefaultDepartmentIdType = "open_department_id"
)

type feishu struct {
	oapiConfig     *config.Config
	contactService *contact.Service
	config         *feishuConfig
}

func (d *feishu) GetTarget() Target {
	return d
}

func (d feishu) GetTargetSlug() string {
	return d.config.Slug
}

func (d feishu) GetPlatform() string {
	return d.config.Platform
}

func (f *feishu) LookupEntryUserByInternalExternalIdentity(internalExtID ExternalIdentity) (UserableEntry, error) {
	contactService := contact.NewService(f.oapiConfig)
	coreCtx := core.WrapContext(context.Background())
	req := contactService.Users.Get(coreCtx)
	req.SetUserId(internalExtID.GetEntryID())
	req.SetUserIdType(feishuDefaultUserIdType)
	resp, err := req.Do()
	if err != nil {
		return nil, err
	}
	return &feishuUser{
		feishu: f,
		raw:    resp.User,
	}, nil
}

func (f *feishu) LookupEntryDepartmentByInternalExternalIdentity(internalExtID ExternalIdentity) (DepartmentableEntry, error) {
	contactService := contact.NewService(f.oapiConfig)
	coreCtx := core.WrapContext(context.Background())
	req := contactService.Departments.Get(coreCtx)
	req.SetDepartmentId(internalExtID.GetEntryID())
	req.SetDepartmentIdType(feishuDefaultDepartmentIdType)
	resp, err := req.Do()
	if err != nil {
		return nil, err
	}
	return &feishuDepartment{
		feishu: f,
		raw:    resp.Department,
	}, nil
}

type feishuConfig struct {
	Platform  string
	Slug      string
	AppID     string
	AppSecret string
}

func (f feishu) InitFormUnmarshaler(unmarshaler func(any) error) (Target, error) {
	err := unmarshaler(&f.config)
	if err != nil {
		return nil, err
	}
	appSettings := core.NewInternalAppSettings(
		core.SetAppCredentials(f.config.AppID, f.config.AppSecret),
	)
	f.oapiConfig = core.NewConfig(core.DomainFeiShu, appSettings, core.SetLoggerLevel(core.LoggerLevelError))
	f.contactService = contact.NewService(f.oapiConfig)
	return &f, nil
}

func (f *feishu) GetRootDepartment() DepartmentableEntry {
	contactService := contact.NewService(f.oapiConfig)
	coreCtx := core.WrapContext(context.Background())
	req := contactService.Departments.List(coreCtx)
	req.SetFetchChild(true)
	resp, _ := req.Do()
	return &feishuDepartment{
		feishu: f,
		raw:    resp.Items[0],
	}
}

func (f *feishu) GetAllUsers() (users []UserableEntry, err error) {
	return RecursionGetAllUsersIncludeChildDepartments(f.GetRootDepartment()), err
}

type feishuDepartment struct {
	*feishu
	raw *contact.Department
}

func (d feishuDepartment) AddToDepartment(options DepartmentModifyUserOptions, extID ExternalIdentity) error {
	if err := extID.CheckIfInternal(d.feishu); err != nil {
		return err
	}
	contactService := contact.NewService(d.feishu.oapiConfig)
	coreCtx := core.WrapContext(context.Background())
	userGetReq := contactService.Users.Get(coreCtx)
	userGetReq.SetUserId(extID.GetEntryID())
	userGetReq.SetUserIdType(feishuDefaultUserIdType)
	userGetResp, err := userGetReq.Do()
	if err != nil {
		return fmt.Errorf("user not found: %s", err)
	}
	if lo.Contains(userGetResp.User.DepartmentIds, d.raw.OpenDepartmentId) {
		return fmt.Errorf("user already in dept: %s", d.raw.Name)
	}
	userPatchReq := contactService.Users.Patch(coreCtx, &contact.User{
		DepartmentIds: append(userGetResp.User.DepartmentIds, d.raw.OpenDepartmentId),
	})
	userPatchReq.SetUserId(extID.GetEntryID())
	userPatchReq.SetUserIdType(feishuDefaultUserIdType)
	_, err = userPatchReq.Do()
	if options.Role == DepartmentUserRoleMember || err != nil {
		return err
	}
	deptPatchReq := contactService.Departments.Patch(coreCtx, &contact.Department{
		LeaderUserId: userGetResp.User.UserId,
	})
	deptPatchReq.SetDepartmentId(extID.GetEntryID())
	deptPatchReq.SetUserIdType(feishuDefaultUserIdType)
	deptPatchReq.SetDepartmentIdType(feishuDefaultUserIdType)
	_, err = deptPatchReq.Do()
	return err
}

func (d feishuDepartment) GetID() string {
	return d.raw.OpenDepartmentId
}

func (d feishuDepartment) GetName() string {
	if d.raw == nil || d.raw.DepartmentId == "0" {
		return "root"
	}
	return d.raw.Name
}

func (d feishuDepartment) GetDescription() string {
	return ""
}

func (d feishuDepartment) GetChildDepartments() (departments []DepartmentableEntry) {
	contactService := contact.NewService(d.feishu.oapiConfig)
	coreCtx := core.WrapContext(context.Background())
	req := contactService.Departments.List(coreCtx)
	req.SetParentDepartmentId(d.raw.OpenDepartmentId)
	req.SetDepartmentIdType(feishuDefaultDepartmentIdType)
	resp, err := req.Do()
	if err != nil {
		fmt.Println(err)
	}
	for _, v := range resp.Items {
		departments = append(departments, &feishuDepartment{
			feishu: d.feishu,
			raw:    v,
		})
	}
	return departments
}

func (d feishuDepartment) CreateChildDepartment(department Departmentable) (DepartmentableEntry, error) {
	contactService := contact.NewService(d.feishu.oapiConfig)
	coreCtx := core.WrapContext(context.Background())
	req := contactService.Departments.Create(coreCtx, &contact.Department{
		Name:               department.GetName(),
		ParentDepartmentId: d.raw.OpenDepartmentId,
	})
	req.SetDepartmentIdType(feishuDefaultDepartmentIdType)
	resp, err := req.Do()
	if err != nil {
		return nil, err
	}
	return &feishuDepartment{
		feishu: d.feishu,
		raw:    resp.Department,
	}, nil
}

func (d feishuDepartment) GetUsers() (users []UserableEntry) {
	contactService := contact.NewService(d.feishu.oapiConfig)
	coreCtx := core.WrapContext(context.Background())
	req := contactService.Users.List(coreCtx)
	req.SetDepartmentId(d.raw.OpenDepartmentId)
	req.SetDepartmentIdType(feishuDefaultDepartmentIdType)
	resp, _ := req.Do()
	for _, v := range resp.Items {
		users = append(users, &feishuUser{
			feishu: d.feishu,
			raw:    v,
		})
	}
	return users
}

type feishuUser struct {
	*feishu
	raw *contact.User
}

func (u feishuUser) GetID() (userId string) {
	return u.raw.UserId
}

func (u feishuUser) GetName() (name string) {
	return u.raw.Name
}

func (u feishuUser) GetEmail() string {
	return u.raw.Email
}

func (u feishuUser) GetEmails() []string {
	return []string{u.raw.Email, u.raw.EnterpriseEmail}
}

func (u feishuUser) GetPhone() string {
	return u.raw.Mobile
}
