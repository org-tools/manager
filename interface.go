package orgmanager

import (
	"fmt"

	"github.com/spf13/viper"
)

type Platform interface {
	InitFormUnmarshaler(unmarshaler func(any) error) (Target, error)
}

var enabledPlatform = map[string]Platform{
	"azuread":  &AzureAD{},
	"dingtalk": &DingTalk{},
	"github":   &GitHub{},
}

func InitTarget(configKey string) (Target, error) {
	platformKey := viper.GetString(fmt.Sprintf("%s.platform", configKey))
	if p, exist := enabledPlatform[platformKey]; exist {
		return p.InitFormUnmarshaler(func(a any) error {
			return viper.UnmarshalKey(fmt.Sprintf("%s.config", configKey), a)
		})
	}
	return nil, fmt.Errorf("Platform %s not exist", platformKey)
}

type Target interface {
	RootDepartment() UnionDepartment
}

type Config struct {
	Platform string
	Config   any
}

type UnionUser interface {
	UserId() (userId string)
	UserName() (name string)
}

type User struct {
	Name  string
	union UnionUser
}

func (d *User) FromInterface(in UnionUser) {
	*d = User{
		Name:  in.UserName(),
		union: in,
	}
}

type UnionDepartment interface {
	Name() (name string)
	SubDepartments() (departments []UnionDepartment)
	Users() (users []UnionUser)
}

type Department struct {
	Name           string
	union          UnionDepartment
	SubDepartments []Department
	Users          []User
}

func (d *Department) FromInterface(in UnionDepartment) {
	*d = Department{
		Name:           in.Name(),
		union:          in,
		SubDepartments: []Department{},
		Users:          []User{},
	}
}

type PreFixOptions struct {
	FixUsers          bool
	FixSubDepartments bool
}

var defaultPrefixOptions = &PreFixOptions{
	FixUsers:          true,
	FixSubDepartments: true,
}

func (d *Department) PreFix(opts *PreFixOptions) {
	if opts.FixUsers {
		for _, v := range d.union.Users() {
			user := new(User)
			user.FromInterface(v)
			d.Users = append(d.Users, *user)
		}
	}
	if opts.FixSubDepartments {
		for _, v := range d.union.SubDepartments() {
			dept := new(Department)
			dept.FromInterface(v)
			dept.PreFix(opts)
			d.SubDepartments = append(d.SubDepartments, *dept)
		}
	}
}
