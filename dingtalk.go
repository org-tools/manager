package orgmanager

import (
	"fmt"
	"strconv"

	"github.com/zhaoyunxing92/dingtalk/v2"
	"github.com/zhaoyunxing92/dingtalk/v2/request"
	"github.com/zhaoyunxing92/dingtalk/v2/response"
)

type DingTalk struct {
	client *dingtalk.DingTalk
	config *dingTalkConfig
}

func (d DingTalk) GetTargetSlug() string {
	return d.config.Slug
}

func (d DingTalk) GetPlatform() string {
	return d.config.Platform
}

type dingTalkConfig struct {
	Platform   string
	Slug       string
	AppKey     string
	AppSecret  string
	RootDeptID int
}

func (d *DingTalk) InitFormUnmarshaler(unmarshaler func(any) error) (Target, error) {
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

func (d DingTalk) RootDepartment() UnionDepartment {
	return &dingTalkDept{
		target: d,
		deptId: d.config.RootDeptID,
	}
}

type dingTalkDept struct {
	target  DingTalk
	deptId  int
	rawList response.DeptList
	detial  *response.DeptDetail
}

func (d dingTalkDept) SubDepartments() (groups []UnionDepartment) {
	groups = make([]UnionDepartment, 0)
	resp, _ := d.target.client.GetDeptList(&request.DeptList{DeptId: d.deptId})
	for _, dept := range resp.Depts {
		groups = append(groups, &dingTalkDept{
			target:  d.target,
			deptId:  dept.Id,
			rawList: resp,
		})
	}
	return groups
}

func (d dingTalkDept) CreateSubDepartment(options DepartmentCreateOptions) (UnionDepartment, error) {
	resp, err := d.target.client.CreateDept(&request.CreateDept{
		Name:             options.Name,
		ParentId:         uint(d.deptId),
		Order:            0,
		SourceIdentifier: "",
	})
	if err != nil {
		return nil, fmt.Errorf("Create dingtalk Dept error: %s", err)
	}
	detail, err := d.target.client.GetDeptDetail(&request.DeptDetail{DeptId: resp.DeptId})
	if err != nil {
		return nil, fmt.Errorf("Get dingtalk Dept detail error: %s", err)
	}
	return &dingTalkDept{
		target: d.target,
		deptId: detail.Id,
		detial: &detail,
	}, err
}

func (d dingTalkDept) Name() (name string) {
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

func (g dingTalkDept) DepartmentID() (departmentId string) {
	return strconv.Itoa(g.deptId)
}

func (g dingTalkDept) Users() (users []UnionUser) {
	users = make([]UnionUser, 0)
	cursor := 0
FETCH:
	resp, _ := g.target.client.GetDeptDetailUserInfo(&request.DeptDetailUserInfo{DeptId: g.deptId, Size: 100, Cursor: cursor})
	for _, v := range resp.DeptDetailUsers {
		users = append(users, &dingTalkUser{
			userId:  v.UserId,
			target:  g.target,
			rawList: resp,
		})
	}
	if resp.HasMore {
		cursor = resp.NextCursor
		goto FETCH
	}
	return users
}

type dingtalkDeptAddUserOption struct {
	target DingTalk
	user   response.UserDetail
}

func (o *dingtalkDeptAddUserOption) FromInterface(union DepartmentAddUserOptions) {
	o.user, _ = o.target.client.GetUserDetail(&request.UserDetail{UserId: ""})
}

func (d dingTalkDept) AddUser(union DepartmentAddUserOptions) error {
	d.target.client.UpdateUser(request.NewUpdateUser("").SetDept(1).Build())
	return nil
}

type dingTalkUser struct {
	target  DingTalk
	userId  string
	rawList response.DeptDetailUserInfo
}

func (u dingTalkUser) ExternalIdentity() ExternalIdentity {
	return ExternalIdentity(fmt.Sprintf("ei.user.%s@%s.%s", u.userId, u.target.config.Slug, u.target.config.Platform))
}

func (u dingTalkUser) UserId() string {
	return u.userId
}

func (u dingTalkUser) UserName() string {
	for _, userInfo := range u.rawList.DeptDetailUsers {
		if userInfo.UserId == u.userId {
			return userInfo.Name
		}
	}
	return u.userId
}
