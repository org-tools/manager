package orgmanager

import (
	"errors"
	"fmt"
	"net/mail"
	"path"
	"strings"

	"github.com/spf13/viper"
)

type Platform interface {
	InitFormUnmarshaler(unmarshaler func(any) error) (Target, error)
}

var enabledPlatform = map[string]Platform{
	"azuread":  &azureAD{},
	"dingtalk": &dingTalk{},
	"github":   &gitHub{},
	"feishu":   &feishu{},
	"local":    &local{},
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
	GetRootDepartment() UnionDepartment
	GetAllUsers() (users []BasicUserable, err error)
}

func RecursionGetAllUsersIncludeChildDepartments(department UnionDepartment) (users []BasicUserable) {
	users = append(users, department.GetUsers()...)
	for _, childDepartment := range department.GetChildDepartments() {
		users = append(users, RecursionGetAllUsersIncludeChildDepartments(childDepartment)...)
	}
	return users
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

type BasicUserable interface {
	Entry
	GetID() (userId string)
	GetName() (name string)
}

type UserableWithEmailSet interface {
	GetEmailSet() (emails []string)
}

type UnionUserWithRole interface {
	BasicUserable
	GetRole() (role DepartmentUserRole)
}

type User struct {
	Name  string
	union BasicUserable
}

func (d *User) FromInterface(in BasicUserable) {
	*d = User{
		Name:  in.GetName(),
		union: in,
	}
}

type UnionDepartment interface {
	GetName() (name string)
	GetID() (departmentId string)
	GetChildDepartments() (departments []UnionDepartment)
	GetUsers() (users []BasicUserable)
}

type DepartmentCreateOptions struct {
	Name        string
	Description string
}

type UnionDepartmentWriter interface {
	CreateSubDepartment(options DepartmentCreateOptions) (UnionDepartment, error)
}

type UnionUserCreateOptions interface {
	GetMailNickname() (mailNickname string)
	GetUserName() (userName string)
	GetUserPhone() (userPhone string)
}

type DefaultUserCreateOptions struct {
	Name  string
	Email string
	Phone string
}

func (o DefaultUserCreateOptions) GetUserName() (userName string) {
	return o.Name
}

func (o DefaultUserCreateOptions) GetUserPhone() (userPhone string) {
	return o.Phone
}

func (o DefaultUserCreateOptions) GetMailNickname() (mailNickname string) {
	if addr, err := mail.ParseAddress(o.Email); err == nil {
		mailNickname = strings.Split(addr.Address, "@")[0]
	}
	if mailNickname == "" {
		mailNickname = o.Name
	}
	return mailNickname
}

type UnionUserWriter interface {
	CreateUser(options UnionUserCreateOptions) (BasicUserable, error)
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
		Name:           in.GetName(),
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
		for _, v := range d.union.GetUsers() {
			user := new(User)
			user.FromInterface(v)
			d.Users = append(d.Users, *user)
		}
	}
	if opts.FixSubDepartments {
		for _, v := range d.union.GetChildDepartments() {
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
