package orgmanager

import (
	"net/mail"
	"strings"
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
	if u, ok := user.(UserCreateableWithMailNickName); ok {
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

type UserCreateableWithMailNickName interface {
	Userable
	GetMailNickname() (mailNickname string)
}

func GetUserableEmails(user Userable) (mails []string) {
	if u, ok := user.(UserCreateableWithMails); ok {
		return u.GetEmails()
	}
	return []string{user.GetEmail()}
}

type UserCreateableWithMails interface {
	Userable
	GetEmails() (mails []string)
}

func GetUserablePhones(user Userable) (phones []string) {
	if u, ok := user.(UserCreateableWithPhones); ok {
		return u.GetPhones()
	}
	return []string{user.GetPhone()}
}

type UserCreateableWithPhones interface {
	Userable
	GetPhones() (phones []string)
}

func GetUserableNames(user Userable) (names []string) {
	if u, ok := user.(UserCreateableWithNames); ok {
		return u.GetNames()
	}
	return []string{user.GetName()}
}

type UserCreateableWithNames interface {
	Userable
	GetNames() (names []string)
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
}
