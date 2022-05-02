package orgmanager

import (
	"errors"
	"fmt"
	"path"

	"github.com/spf13/viper"
)

type Platform interface {
	InitFormUnmarshaler(unmarshaler func(any) error) (Target, error)
}

var enabledPlatform = map[string]Platform{
	"azuread":  &AzureAD{},
	"dingtalk": &DingTalk{},
	"github":   &GitHub{},
	"feishu":   &Feishu{},
}

func InitTarget(configKey string) (Target, error) {
	platformKey := viper.GetString(fmt.Sprintf("%s.platform", configKey))
	if p, exist := enabledPlatform[platformKey]; exist {
		target, err := p.InitFormUnmarshaler(func(a any) error {
			return viper.UnmarshalKey(fmt.Sprintf("%s", configKey), a)
		})
		if target.GetPlatform() == "" || target.GetTargetSlug() == "" {
			err = fmt.Errorf("Platform Or Slug of %s config not exist", path.Ext(platformKey))
		}
		return target, err
	}
	return nil, fmt.Errorf("Platform %s not exist", platformKey)
}

type Target interface {
	TargetEntry
	GetTargetSlug() string
	GetPlatform() string
	RootDepartment() UnionDepartment
}

func GetTargetByPlatformAndSlug(platform, slug string) (Target, error) {
	for _, target := range Targets {
		if target.GetTargetSlug() == slug && target.GetPlatform() == platform {
			return target, nil
		}
	}
	return nil, errors.New("target not found")
}

type Config struct {
	Platform string
}

type UnionUser interface {
	GetUserId() (userId string)
	GetUserName() (name string)
	GetEmailSet() (emails []string)
}

type UnionUserWithRole interface {
	GetUserId() (userId string)
	GetUserName() (name string)
	GetRole() string
}

type User struct {
	Name  string
	union UnionUser
}

func (d *User) FromInterface(in UnionUser) {
	*d = User{
		Name:  in.GetUserName(),
		union: in,
	}
}

type UnionDepartment interface {
	Name() (name string)
	DepartmentID() (departmentId string)
	SubDepartments() (departments []UnionDepartment)
	Users() (users []UnionUser)
}

type DepartmentCreateOptions struct {
	Name        string
	Description string
}

type UnionDepartmentWriter interface {
	CreateSubDepartment(options DepartmentCreateOptions) (UnionDepartment, error)
}

type UserCreateOptions struct {
	Name  string
	Email string
	Phone string
}

type UnionUserWriter interface {
	CreateUser(options UserCreateOptions) (UnionUser, error)
}

type DepartmentModifyUserOptions struct {
	Role DepartmentUserRole
}

type DepartmentUserWriter interface {
	AddToDepartment(options DepartmentModifyUserOptions, extID ExternalIdentity) error
	// RemoveFromDepartment(options DepartmentModifyUserOptions, extID ExternalIdentity) error
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

type EmailListGettable interface {
	GetEmailSet() (emails []string)
}

type EmailListEditable interface {
	AddToEmailSet(email string) error
	DeleteFromEmailSet(email string) error
}
