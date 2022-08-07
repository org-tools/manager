package manager

import (
	"net/mail"
	"strings"

	"github.com/samber/lo"
)

type Userable interface {
	GetName() (name string)
	GetEmail() (email string)
	GetPhone() (phone string)
}

type UserableEntry interface {
	Entry
	Userable
}

type UserableWithRole interface {
	UserableEntry
	GetRole() (role DepartmentUserRole)
}

func GetUserableMailNickname(user Userable) (mailNickname string) {
	if u, ok := user.(UserableWithMailNickName); ok {
		return u.GetMailNickname()
	}
	if addr, err := mail.ParseAddress(user.GetEmail()); err == nil {
		mailNickname = strings.Split(addr.Address, "@")[0]
	}
	if mailNickname == "" {
		mailNickname = user.GetName()
	}
	return mailNickname
}

type UserableWithMailNickName interface {
	Userable
	GetMailNickname() (mailNickname string)
}

func GetUserableEmails(user Userable) (mails []string) {
	if u, ok := user.(UserableWithMails); ok {
		return lo.Uniq(append(u.GetEmails(), user.GetEmail()))
	}
	return []string{user.GetEmail()}
}

type UserableWithMails interface {
	Userable
	GetEmails() (mails []string)
}

func GetUserablePhones(user Userable) (phones []string) {
	if u, ok := user.(UserableWithPhones); ok {
		return lo.Uniq(append(u.GetPhones(), user.GetPhone()))
	}
	return []string{user.GetPhone()}
}

type UserableWithPhones interface {
	Userable
	GetPhones() (phones []string)
}

func GetUserableNames(user Userable) (names []string) {
	if u, ok := user.(UserableWithNames); ok {
		return lo.Uniq(append(u.GetNames(), user.GetName()))
	}
	return []string{user.GetName()}
}

type UserableWithNames interface {
	Userable
	GetNames() (names []string)
}

type UserableCanMerge interface {
	UserableEntry
	Merge(user UserableEntry) error
}

func NewUser() *User {
	return new(User)
}

type User struct {
	Name  string
	Email string
	Phone string
}

func (o User) GetName() (name string) {
	return o.Name
}

func (o User) GetEmail() (email string) {
	return o.Email
}

func (o User) GetPhone() (phonr string) {
	return o.Phone
}

func (o User) GetMailNickname() (mailNickname string) {
	if addr, err := mail.ParseAddress(o.Email); err == nil {
		mailNickname = strings.Split(addr.Address, "@")[0]
	}
	if mailNickname == "" {
		mailNickname = o.Name
	}
	return mailNickname
}

type UserWriteable interface {
	CreateUser(options Userable) (UserableEntry, error)
	LookupUser(options Userable) (UserableEntry, error)
}
