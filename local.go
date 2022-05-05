package orgmanager

import (
	"encoding/json"
	"errors"

	"github.com/google/uuid"
	"github.com/samber/lo"
	"gorm.io/datatypes"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type local struct {
	db     *gorm.DB
	config *localConfig
}

func (l *local) InitFormUnmarshaler(unmarshaler func(any) error) (Target, error) {
	err := unmarshaler(&l.config)
	if err != nil {
		return nil, err
	}
	if l.config.RootDepartmentUUID == uuid.Nil {
		l.config.RootDepartmentUUID = localDefaultRootDepartmentUUID
	}
	l.db, err = gorm.Open(sqlite.Open(l.config.FileDSN), &gorm.Config{})
	l.db.AutoMigrate(&localUser{}, &localDepartment{})
	return l, err
}

type localConfig struct {
	Slug               string
	Platform           string
	FileDSN            string
	RootDepartmentUUID uuid.UUID
}

var localDefaultRootDepartmentUUID = uuid.NameSpaceDNS

func (l local) CreateUser(user Userable) (UserableEntry, error) {
	newUser := &localUser{
		Name:   user.GetName(),
		Names:  jsonMap(GetUserableNames(user)),
		Phone:  user.GetPhone(),
		Phones: jsonMap(GetUserablePhones(user)),
		Email:  user.GetEmail(),
		Emails: jsonMap(GetUserableEmails(user)),
	}
	if e, ok := user.(UserableEntry); ok {
		newUser.ExtIDs = jsonMap([]string{string(ExternalIdentityOfEntry(e))})
	}
	return newUser, l.db.Create(&newUser).Error
}

func (l local) LookupEntryUserByInternalExternalIdentity(internalExtID ExternalIdentity) (UserableEntry, error) {
	return l.lookupLocalUserByInternalExternalIdentity(internalExtID)
}

func (l local) lookupLocalUserByInternalExternalIdentity(internalExtID ExternalIdentity) (user *localUser, err error) {
	err = l.db.Where(&localUser{ID: uuid.MustParse(internalExtID.GetEntryID())}).Find(&user).Error
	return user, err
}

func (l local) LookupEntryDepartmentByInternalExternalIdentity(internalExtID ExternalIdentity) (DepartmentableEntry, error) {
	return l.lookupLocalDepartmentByInternalExternalIdentity(internalExtID)
}

func (l local) lookupLocalDepartmentByInternalExternalIdentity(internalExtID ExternalIdentity) (dept *localDepartment, err error) {
	err = l.db.Where(&localDepartment{ID: uuid.MustParse(internalExtID.GetEntryID())}).Find(&dept).Error
	return dept, err
}

func (l *local) LookupUser(user Userable) (UserableEntry, error) {
	result := &localUser{local: l}
	req := l.db.Debug().
		Where(datatypes.JSONQuery("names").HasKey(user.GetName()))
	if email := user.GetEmail(); email != "" {
		req = req.Or(datatypes.JSONQuery("emails").HasKey(user.GetEmail()))
	}
	if phone := user.GetPhone(); phone != "" {
		req = req.Or(datatypes.JSONQuery("phones").HasKey(user.GetPhone()))
	}
	req = req.Find(&result)
	if req.RowsAffected > 1 {
		return nil, errors.New("muli users matched")
	}
	return result, req.Error
}

func (l local) GetTargetSlug() string {
	return l.config.Slug
}

func (l local) GetPlatform() string {
	return l.config.Platform
}

func (l *local) GetRootDepartment() DepartmentableEntry {
	rootDepartment := new(localDepartment)
	rf := l.db.Where(&localDepartment{ID: l.config.RootDepartmentUUID}).Find(&rootDepartment).RowsAffected
	if rf == 0 {
		rootDepartment = &localDepartment{ID: l.config.RootDepartmentUUID, Name: "root"}
		l.db.Create(&rootDepartment)
	}
	rootDepartment.local = l
	return rootDepartment
}

func (l *local) GetAllUsers() (users []UserableEntry, err error) {
	localUsers := make([]localUser, 0)
	err = l.db.Model(&localUser{}).Find(&localUser{}).Error
	if err != nil {
		return nil, err
	}
	for _, v := range localUsers {
		v.local = l
		users = append(users, &v)
	}
	return users, err
}

func (l *local) LookupEntryByExternalIdentity(extID ExternalIdentity) (Entry, error) {
	return nil, nil
}

func (l *local) LookupEntryUserByExternalIdentity(extID ExternalIdentity) (UserEntryExtIDStoreable, error) {
	user := &localUser{local: l}
	return user, l.db.Where(&localUser{ID: uuid.MustParse(extID.GetEntryID())}).Find(&user).Error
}

func (l *local) LookupEntryDepartmentByExternalIdentity(extID ExternalIdentity) (DepartmentEntryExtIDStoreable, error) {
	dept := &localDepartment{local: l}
	return dept, l.db.Where(&localDepartment{ID: uuid.MustParse(extID.GetEntryID())}).Find(&dept).Error
}

type localUser struct {
	*local

	ID         uuid.UUID `gorm:"primaryKey"`
	Name       string
	Names      datatypes.JSONMap
	Phone      string
	Phones     datatypes.JSONMap
	Email      string
	Emails     datatypes.JSONMap
	ExtIDs     datatypes.JSONMap
	Departemts datatypes.JSONMap
}

func (u *localUser) BeforeCreate(_ *gorm.DB) error {
	if u.ID == uuid.Nil {
		u.ID = uuid.New()
	}
	return nil
}

func (u localUser) GetID() string {
	return u.ID.String()
}

func (u *localUser) GetTarget() Target {
	return u.local
}

func (u localUser) GetName() string {
	return u.Name
}

func (u localUser) SetName(name string) {
	u.Name = name
	u.Names = jsonMap(append(u.GetNames(), name))
}

func (u localUser) GetNames() (names []string) {
	return lo.Keys(u.Names)
}

func (u *localUser) SetNames(names []string) {
	u.Names = jsonMap(names)
}

func (u localUser) GetEmail() string {
	if u.Email == "" && len(u.GetEmails()) != 0 {
		return u.GetEmails()[0]
	}
	return u.Email
}

func (u *localUser) SetEmail(email string) {
	u.Email = email
	u.Emails = jsonMap(append(u.GetEmails(), email))
}

func (u localUser) GetEmails() []string {
	return lo.Keys(u.Emails)
}

func (u *localUser) SetEmails(emails []string) {
	u.Emails = jsonMap(emails)
}

func (u localUser) GetPhone() string {
	return u.Phone
}

func (u *localUser) SetPhone(phone string) {
	u.Phone = phone
	u.Phones = jsonMap(append(u.GetPhones(), phone))
}

func (u localUser) GetPhones() []string {
	return lo.Keys(u.Phones)
}

func (u *localUser) SetPhones(phones []string) {
	u.Phones = jsonMap(phones)
}

func (u localUser) GetExternalIdentities() (extIDs ExternalIdentities) {
	return ExternalIdentitiesFromStringList(lo.Keys(u.ExtIDs))
}

func (u *localUser) SetExternalIdentities(extIDs ExternalIdentities) (err error) {
	u.ExtIDs = jsonMap(extIDs.StringList())
	return u.db.Save(u).Error
}

func (u *localUser) Merge(user UserableEntry) error {
	u.Names = jsonMap(GetUserableNames(u), GetUserableNames(user)...)
	u.Phones = jsonMap(GetUserablePhones(u), GetUserablePhones(user)...)
	u.Emails = jsonMap(GetUserableEmails(u), GetUserableEmails(user)...)
	u.ExtIDs = jsonMap(u.GetExternalIdentities().StringList(), string(ExternalIdentityOfEntry(user)))
	return u.Save()
}

func (u *localUser) Save() error {
	return u.db.Save(&u).Error
}

type localDepartment struct {
	*local

	ID          uuid.UUID `gorm:"primaryKey"`
	Name        string
	Description string
	ParentID    uuid.UUID
	ExtIDs      datatypes.JSON
}

func (u *localDepartment) BeforeCreate(_ *gorm.DB) error {
	if u.ID == uuid.Nil {
		u.ID = uuid.New()
	}
	return nil
}

func (d localDepartment) GetID() string {
	return d.ID.String()
}

func (u *localDepartment) GetTarget() Target {
	return u.local
}

func (d localDepartment) GetName() string {
	return d.Name
}

func (d localDepartment) GetDescription() string {
	return d.Description
}

func (d localDepartment) GetChildDepartments() (departments []DepartmentableEntry) {
	localDepartments := make([]localDepartment, 0)
	d.db.Where(&localDepartment{ParentID: d.ID}).Find(&localDepartments)
	for _, v := range localDepartments {
		v.local = d.local
		departments = append(departments, &v)
	}
	return departments
}

func (d localDepartment) CreateChildDepartment(department Departmentable) (DepartmentableEntry, error) {
	newDepartment := &localDepartment{
		local:    d.local,
		Name:     department.GetName(),
		ParentID: d.ID,
	}
	return newDepartment, d.db.Create(&newDepartment).Error
}

func (d localDepartment) GetUsers() (users []UserableEntry) {
	localUsers := make([]localUser, 0)
	d.db.Model(&localUser{}).Where(datatypes.JSONQuery("departemts").HasKey(d.ID.String())).Find(&localUsers)
	for _, v := range localUsers {
		v.local = d.local
		users = append(users, &v)
	}
	return users
}

func (d localDepartment) GetExternalIdentities() (extIDs ExternalIdentities) {
	_ = json.Unmarshal(d.ExtIDs, &extIDs)
	return extIDs
}

func (d *localDepartment) SetExternalIdentities(extIDs ExternalIdentities) (err error) {
	d.ExtIDs, err = json.Marshal(extIDs)
	if err != nil {
		return err
	}
	return d.db.Save(d).Error
}

func JSON(in any) (bytes datatypes.JSON) {
	bytes, _ = json.Marshal(in)
	return bytes
}

func jsonMap(list []string, ext ...string) (m datatypes.JSONMap) {
	m = make(datatypes.JSONMap)
	for _, item := range list {
		if item == "" {
			continue
		}
		m[item] = true
	}
	for _, item := range ext {
		if item == "" {
			continue
		}
		m[item] = true
	}
	return m
}
