package orgmanager

import (
	"github.com/zhaoyunxing92/dingtalk/v2"
	"github.com/zhaoyunxing92/dingtalk/v2/request"
	"github.com/zhaoyunxing92/dingtalk/v2/response"
)

type DingTalk struct {
	client *dingtalk.DingTalk
	config *dingTalkConfig
}

type dingTalkConfig struct {
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
		client: d.client,
		deptId: d.config.RootDeptID,
	}
}

type dingTalkDept struct {
	client  *dingtalk.DingTalk
	deptId  int
	rawList response.DeptList
}

func (d dingTalkDept) SubDepartments() (groups []UnionDepartment) {
	groups = make([]UnionDepartment, 0)
	resp, _ := d.client.GetDeptList(&request.DeptList{DeptId: d.deptId})
	for _, dept := range resp.Depts {
		groups = append(groups, &dingTalkDept{
			client:  d.client,
			deptId:  dept.Id,
			rawList: resp,
		})
	}
	return groups
}

func (d dingTalkDept) Name() (name string) {
	if d.deptId == 0 {
		return "root"
	}
	for _, dept := range d.rawList.Depts {
		if dept.Id == d.deptId {
			return dept.Name
		}
	}
	return name
}

func (g dingTalkDept) Users() (users []UnionUser) {
	users = make([]UnionUser, 0)
	cursor := 0
FETCH:
	resp, _ := g.client.GetDeptDetailUserInfo(&request.DeptDetailUserInfo{DeptId: g.deptId, Size: 100, Cursor: cursor})
	for _, v := range resp.DeptDetailUsers {
		users = append(users, &dingTalkUser{
			userId:  v.UserId,
			client:  g.client,
			rawList: resp,
		})
	}
	if resp.HasMore {
		cursor = resp.NextCursor
		goto FETCH
	}
	return users
}

type dingTalkUser struct {
	userId  string
	rawList response.DeptDetailUserInfo
	client  *dingtalk.DingTalk
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
