package orgmanager

import (
	"errors"
	"fmt"
	"path"
)

type Platform interface {
	InitFormUnmarshaler(unmarshaler func(any) error) (Target, error)
}

var enabledPlatform = make(map[string]Platform)

func RegisterPlatform(name string, platform Platform) {
	enabledPlatform[name] = platform
}

func InitTarget(platformKey string, unmarshaler func(any) error) (Target, error) {
	if p, exist := enabledPlatform[platformKey]; exist {
		target, err := p.InitFormUnmarshaler(unmarshaler)
		if target.GetPlatform() == "" || target.GetTargetSlug() == "" {
			err = fmt.Errorf("Platform Or Slug of %s config not exist", path.Ext(platformKey))
		}
		return target, err
	}
	return nil, fmt.Errorf("Platform %s not exist", platformKey)
}

type Target interface {
	TargetEntry
	GetTarget() Target
	GetTargetSlug() string
	GetPlatform() string
	GetRootDepartment() (DepartmentableEntry, error)
	GetAllUsers() (users []UserableEntry, err error)
}

func TargetKey(t Target) string {
	return fmt.Sprintf("%s@%s", t.GetTargetSlug(), t.GetPlatform())
}

type TargetWithEnterpriseEmail interface {
	GetEnterpriseEmailDomains() []string
}

func RecursionGetAllUsersIncludeChildDepartments(department DepartmentableEntry) (users []UserableEntry, err error) {
	usersInThis, err := department.GetUsers()
	if err != nil {
		return
	}
	users = append(users, usersInThis...)
	for _, childDepartment := range department.GetChildDepartments() {
		usersUnder := make([]UserableEntry, 0)
		usersUnder, err = RecursionGetAllUsersIncludeChildDepartments(childDepartment)
		if err != nil {
			return
		}
		users = append(users, usersUnder...)
	}
	return
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
