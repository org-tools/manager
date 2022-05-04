package orgmanager

import (
	"fmt"
	"strconv"

	"github.com/zhaoyunxing92/dingtalk/v2"
	"github.com/zhaoyunxing92/dingtalk/v2/request"
	"github.com/zhaoyunxing92/dingtalk/v2/response"
)

type dingTalk struct {
	client *dingtalk.DingTalk
	config *dingTalkConfig
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

func (d *dingTalk) GetRootDepartment() UnionDepartment {
	return d.getDingTalkRootDepartment()
}

func (d *dingTalk) getDingTalkRootDepartment() *dingTalkDept {
	return &dingTalkDept{
		dingTalk: d,
		deptId:   d.config.RootDeptID,
	}
}

func (d *dingTalk) GetAllUsers() (users []BasicUserable, err error) {
	for _, v := range d.getDingTalkRootDepartment().getAllDingTalkUsers() {
		users = append(users, v)
	}
	return users, err
}

func (d *dingTalk) LookupEntryDepartmentByInternalExternalIdentity(internalExtID ExternalIdentity) (UnionDepartment, error) {
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
		deptId:   resp.Id,
		detial:   &resp,
	}, nil
}

func (d *dingTalk) LookupEntryUserByInternalExternalIdentity(internalExtID ExternalIdentity) (BasicUserable, error) {
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

func (d dingTalkDept) GetChildDepartments() (departments []UnionDepartment) {
	for _, v := range d.getDingTalkChildDepts() {
		departments = append(departments, v)
	}
	return departments
}

func (d dingTalkDept) getDingTalkChildDepts() (depts []*dingTalkDept) {
	resp, _ := d.dingTalk.client.GetDeptList(&request.DeptList{DeptId: d.deptId})
	for _, dept := range resp.Depts {
		depts = append(depts, &dingTalkDept{
			dingTalk: d.dingTalk,
			deptId:   dept.Id,
			rawList:  resp,
		})
	}
	return depts
}

func (d dingTalkDept) CreateSubDepartment(options DepartmentCreateOptions) (UnionDepartment, error) {
	resp, err := d.dingTalk.client.CreateDept(&request.CreateDept{
		Name:     options.Name,
		ParentId: uint(d.deptId),
	})
	if err != nil {
		return nil, fmt.Errorf("Create dingtalk Dept error: %s", err)
	}
	detail, err := d.dingTalk.client.GetDeptDetail(&request.DeptDetail{DeptId: resp.DeptId})
	if err != nil {
		return nil, fmt.Errorf("Get dingtalk Dept detail error: %s", err)
	}
	return &dingTalkDept{
		dingTalk: d.dingTalk,
		deptId:   detail.Id,
		detial:   &detail,
	}, err
}

func (d dingTalkDept) GetName() (name string) {
	if d.deptId == 0 {
		return "root"
	}
	if d.detial != nil {
		return d.detial.Name
	}
	for _, dept := range d.rawList.Depts {
		if dept.Id == d.deptId {
			return dept.Name
		}
	}
	return name
}

func (g dingTalkDept) GetID() (departmentId string) {
	return strconv.Itoa(g.deptId)
}

func (g dingTalkDept) GetUsers() (users []BasicUserable) {
	for _, v := range g.getDingTalkUsers() {
		users = append(users, v)
	}
	return users
}

func (g *dingTalkDept) getDingTalkUsers() (users []*dingTalkUser) {
	cursor := 0
FETCH:
	resp, _ := g.dingTalk.client.GetDeptDetailUserInfo(&request.DeptDetailUserInfo{DeptId: g.deptId, Size: 100, Cursor: cursor})
	for _, v := range resp.DeptDetailUsers {
		users = append(users, &dingTalkUser{
			userId:   v.UserId,
			dingTalk: g.dingTalk,
			rawList:  &resp,
		})
	}
	if resp.HasMore {
		cursor = resp.NextCursor
		goto FETCH
	}
	return users
}

func (g *dingTalkDept) getAllDingTalkUsers() (users []*dingTalkUser) {
	users = append(users, g.getDingTalkUsers()...)
	for _, v := range g.getDingTalkChildDepts() {
		users = append(users, v.getAllDingTalkUsers()...)
	}
	return users
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

func (u dingTalkUser) GetID() string {
	return u.userId
}

func (u dingTalkUser) GetName() string {
	if u.detial != nil {
		return u.detial.Name
	}
	for _, userInfo := range u.rawList.DeptDetailUsers {
		if userInfo.UserId == u.userId {
			return userInfo.Name
		}
	}
	return u.userId
}

func (u dingTalkUser) GetEmailSet() (emails []string) {
	if u.detial != nil {
		return []string{u.detial.OrgEmail}
	}
	for _, userInfo := range u.rawList.DeptDetailUsers {
		if userInfo.UserId == u.userId {
			return []string{}
		}
	}
	return emails
}
