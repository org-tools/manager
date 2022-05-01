package orgmanager

import (
	"errors"
	"fmt"
	"strings"
)

type EntryType string

const (
	EntryTypeUser    EntryType = "user"
	EntryTypeDept    EntryType = "dept"
	EntryTypeProject EntryType = "project"
)

type EntryCenter interface {
	LookupEntryUserByExternalIdentity(extID ExternalIdentity) (UnionUser, error)
	LookupEntryDepartmentByExternalIdentity(extID ExternalIdentity) (UnionDepartment, error)
}

//mail format as ei.{entry_type}.{external_entry_id}@{target_slug}.{platform}
type ExternalIdentity string

func (id ExternalIdentity) GetEntryType() EntryType {
	return EntryType(strings.Split(string(id), ".")[1])
}

func (id ExternalIdentity) GetEntryID() string {
	return strings.Split(strings.Split(string(id), ".")[2], "@")[0]
}

func (id ExternalIdentity) GetTargetSlug() string {
	return strings.Split(strings.Split(string(id), ".")[2], "@")[1]
}

func (id ExternalIdentity) GetPlatform() string {
	return strings.Split(string(id), ".")[3]
}

func ExternalIdentityParseString(raw string) (ExternalIdentity, error) {
	list := strings.Split(raw, ".")
	if len(list) != 4 || list[0] != "ei" || len(strings.Split(list[2], "@")) != 2 {
		return "", errors.New("not external identity mail format")
	}
	return ExternalIdentity(raw), nil
}

func ExternalIdentitiesFromStringList(list []string) (extIDs []ExternalIdentity) {
	for _, v := range list {
		if extID, err := ExternalIdentityParseString(v); err == nil {
			extIDs = append(extIDs, extID)
		}
	}
	return extIDs
}

func ExternalIdentityOfUser(target Target, user UnionUser) ExternalIdentity {
	return ExternalIdentity(fmt.Sprintf("ei.user.%s@%s.%s", user.UserId(), target.GetTargetSlug(), target.GetPlatform()))
}

func ExternalIdentityOfDepartment(target Target, dept UnionDepartment) ExternalIdentity {
	return ExternalIdentity(fmt.Sprintf("ei.dept.%s@%s.%s", dept.DepartmentID(), target.GetTargetSlug(), target.GetPlatform()))
}
