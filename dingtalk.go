package orgmanager

import (
	"fmt"
	"strconv"

	"github.com/samber/lo"
	"github.com/zhaoyunxing92/dingtalk/v2"
	"github.com/zhaoyunxing92/dingtalk/v2/request"
	"github.com/zhaoyunxing92/dingtalk/v2/response"
)

type dingTalk struct {
	client *dingtalk.DingTalk
	config *dingTalkConfig
}

func (d *dingTalk) GetTarget() Target {
	return d
}

func (d dingTalk) GetTargetSlug() string {
	return d.config.Slug
}

func (d dingTalk) GetPlatform() string {
	return d.config.Platform
}

type dingTalkConfig struct {
	Platform   string
	Slug       string
	AppKey     string
	AppSecret  string
	RootDeptID int
}

func (d *dingTalk) InitFormUnmarshaler(unmarshaler func(any) error) (Target, error) {
	err := unmarshaler(&d.config)
	if err != nil {
		return nil, err
	}
	d.client, err = dingtalk.NewClient(d.config.AppKey, d.config.AppSecret)
	if err != nil {
		return nil, err
	}
	return d, nil
}

func (d *dingTalk) GetRootDepartment() (DepartmentableEntry, error) {
	return &dingTalkDept{
		dingTalk: d,
		deptId:   d.config.RootDeptID,
	}, nil
}

func (d *dingTalk) GetAllUsers() (users []UserableEntry, err error) {
	return RecursionGetAllUsersIncludeChildDepartments(lo.Must(d.GetRootDepartment()))
}

func (d *dingTalk) LookupEntryDepartmentByInternalExternalIdentity(internalExtID ExternalIdentity) (DepartmentableEntry, error) {
	deptID, err := strconv.Atoi(internalExtID.GetEntryID())
	if err != nil {
		return nil, err
	}
	resp, err := d.client.GetDeptDetail(&request.DeptDetail{
		DeptId: deptID,
	})
	if err != nil {
		return nil, err
	}
	return &dingTalkDept{
		dingTalk: d,
		deptId:   resp.Detail.Id,
		detial:   &resp,
	}, nil
}

func (d *dingTalk) LookupEntryUserByInternalExternalIdentity(internalExtID ExternalIdentity) (UserableEntry, error) {
	resp, err := d.client.GetUserDetail(&request.UserDetail{
		UserId: internalExtID.GetEntryID(),
	})
	if err != nil {
		return nil, err
	}
	return &dingTalkUser{
		dingTalk: d,
		userId:   resp.UserId,
		detial:   &resp,
	}, nil
}

type dingTalkDept struct {
	*dingTalk
	deptId  int
	rawList response.DeptList
	detial  *response.DeptDetail
}

func (d *dingTalkDept) AddToDepartment(options DepartmentModifyUserOptions, extID ExternalIdentity) error {
	panic(nil)
}

func (d dingTalkDept) GetChildDepartments() (departments []DepartmentableEntry) {
	resp, _ := d.dingTalk.client.GetDeptList(&request.DeptList{DeptId: d.deptId})
	for _, dept := range resp.List {
		departments = append(departments, &dingTalkDept{
			dingTalk: d.dingTalk,
			deptId:   dept.Id,
			rawList:  resp,
		})
	}
	return departments
}

func (d dingTalkDept) CreateChildDepartment(department Departmentable) (DepartmentableEntry, error) {
	resp, err := d.dingTalk.client.CreateDept(&request.CreateDept{
		Name:     department.GetName(),
		ParentId: uint(d.deptId),
	})
	if err != nil {
		return nil, fmt.Errorf("Create dingtalk Dept error: %s", err)
	}
	detailResp, err := d.dingTalk.client.GetDeptDetail(&request.DeptDetail{DeptId: resp.Dept.DeptId})
	if err != nil {
		return nil, fmt.Errorf("Get dingtalk Dept detail error: %s", err)
	}
	return &dingTalkDept{
		dingTalk: d.dingTalk,
		deptId:   detailResp.Detail.Id,
		detial:   &detailResp,
	}, err
}

func (g dingTalkDept) GetID() string {
	return strconv.Itoa(g.deptId)
}

func (d dingTalkDept) GetName() (name string) {
	if d.deptId == 0 {
		return "root"
	}
	if d.detial != nil {
		return d.detial.Detail.Name
	}
	for _, dept := range d.rawList.List {
		if dept.Id == d.deptId {
			return dept.Name
		}
	}
	return name
}

func (d dingTalkDept) GetDescription() string {
	if d.detial == nil {
		d.fetchDetail()
	}
	return d.detial.Detail.Brief
}

func (g dingTalkDept) GetUsers() (users []UserableEntry, err error) {
	cursor := 0
FETCH:
	resp, err := g.dingTalk.client.GetDeptDetailUserInfo(&request.DeptDetailUserInfo{DeptId: g.deptId, Size: 100, Cursor: cursor})
	if err != nil {
		return
	}
	for _, v := range resp.Page.List {
		users = append(users, &dingTalkUser{
			userId:   v.UserId,
			dingTalk: g.dingTalk,
			rawList:  &resp,
		})
	}
	if resp.Page.HasMore {
		cursor = resp.Page.NextCursor
		goto FETCH
	}
	return
}

func (d *dingTalkDept) fetchDetail() (err error) {
	detial, err := d.client.GetDeptDetail(&request.DeptDetail{
		DeptId: d.deptId,
	})
	d.detial = &detial
	return err
}

type dingtalkDeptAddUserOption struct {
	target dingTalk
	user   response.UserDetail
}

func (o *dingtalkDeptAddUserOption) FromInterface(union DepartmentModifyUserOptions) {
	o.user, _ = o.target.client.GetUserDetail(&request.UserDetail{UserId: ""})
}

func (d dingTalkDept) AddUser(union DepartmentModifyUserOptions) error {
	d.dingTalk.client.UpdateUser(request.NewUpdateUser("").SetDept(1).Build())
	return nil
}

type dingTalkUser struct {
	*dingTalk
	userId  string
	rawList *response.DeptDetailUserInfo
	detial  *response.UserDetail
}

func (u *dingTalkUser) fetchUserDetail() (err error) {
	detial, err := u.client.GetUserDetail(&request.UserDetail{
		UserId: u.userId,
	})
	u.detial = &detial
	return err
}

func (u dingTalkUser) GetID() string {
	return u.userId
}

func (u dingTalkUser) GetName() string {
	if u.detial != nil {
		return u.detial.Name
	}
	for _, userInfo := range u.rawList.Page.List {
		if userInfo.UserId == u.userId {
			return userInfo.Name
		}
	}
	return u.userId
}

func (u dingTalkUser) GetEmail() string {
	if u.detial == nil {
		u.fetchUserDetail()
	}
	return u.detial.OrgEmail
}

func (u dingTalkUser) GetPhone() string {
	if u.detial != nil {
		return u.detial.Mobile
	}
	for _, userInfo := range u.rawList.Page.List {
		if userInfo.UserId == u.userId {
			return userInfo.Mobile
		}
	}
	return u.userId
}

func (u dingTalkUser) GetEmailSet() (emails []string) {
	if u.detial != nil {
		return []string{u.detial.OrgEmail}
	}
	for _, userInfo := range u.rawList.Page.List {
		if userInfo.UserId == u.userId {
			return []string{}
		}
	}
	return emails
}
