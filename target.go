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
	GetTarget() Target
	GetTargetSlug() string
	GetPlatform() string
	GetRootDepartment() DepartmentableEntry
	GetAllUsers() (users []UserableEntry, err error)
}

func RecursionGetAllUsersIncludeChildDepartments(department DepartmentableEntry) (users []UserableEntry) {
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
