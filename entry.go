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

var _ = []EntryCenter{
	&azureAD{},
	&local{},
}

type EntryCenter interface {
	LookupEntryByExternalIdentity(extID ExternalIdentity) (Entry, error)
	LookupEntryUserByExternalIdentity(extID ExternalIdentity) (UserEntryExtIDStoreable, error)
	LookupEntryDepartmentByExternalIdentity(extID ExternalIdentity) (DepartmentEntryExtIDStoreable, error)
}

type UserEntryExtIDStoreable interface {
	UserableEntry
	EntryExtIDStoreable
}

type DepartmentEntryExtIDStoreable interface {
	Departmentable
	EntryExtIDStoreable
}

type EntryExtIDStoreable interface {
	GetExternalIdentities() []ExternalIdentity
	SetExternalIdentities(extIDs []ExternalIdentity) error
}

type TargetEntry interface {
	LookupEntryUserByInternalExternalIdentity(internalExtID ExternalIdentity) (UserableEntry, error)
	LookupEntryDepartmentByInternalExternalIdentity(internalExtID ExternalIdentity) (DepartmentableEntry, error)
}

//mail format as ei.{entry_type}.{external_entry_id}@{target_slug}.{platform}
type ExternalIdentity string

const InvalidExternalIdentity ExternalIdentity = ""

func (id ExternalIdentity) GetEntryType() EntryType {
	return EntryType(strings.Split(string(id), ".")[1])
}

func (id ExternalIdentity) CheckIfInternal(target Target) error {
	if id.GetPlatform() != target.GetPlatform() || id.GetTargetSlug() != target.GetTargetSlug() {
		return fmt.Errorf("not internal %s", id.GetEntryType())
	}
	return nil
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

func (id ExternalIdentity) GetTarget() (Target, error) {
	return GetTargetByPlatformAndSlug(id.GetPlatform(), id.GetTargetSlug())
}

func (id ExternalIdentity) Valid() bool {
	_, err := ExternalIdentityParseString(string(id))
	return err == nil
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

type Entry interface {
	GetID() string
	GetTargetSlug() string
	GetPlatform() string
}

func ExternalIdentityOfEntry(target Target, entry Entry) ExternalIdentity {
	if user, ok := entry.(UserableEntry); ok {
		return ExternalIdentityOfUser(target, user)
	}
	if dept, ok := entry.(DepartmentableEntry); ok {
		return ExternalIdentityOfDepartment(target, dept)
	}
	return InvalidExternalIdentity
}

func ExternalIdentityOfUser(target Target, user UserableEntry) ExternalIdentity {
	return ExternalIdentity(fmt.Sprintf("ei.user.%s@%s.%s", user.GetID(), target.GetTargetSlug(), target.GetPlatform()))
}

func ExternalIdentityOfDepartment(target Target, dept DepartmentableEntry) ExternalIdentity {
	return ExternalIdentity(fmt.Sprintf("ei.dept.%s@%s.%s", dept.GetID(), target.GetTargetSlug(), target.GetPlatform()))
}
