package orgmanager

import (
	"encoding/json"

	"github.com/google/uuid"
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
		Names:  JSON(GetUserableNames(user)),
		Phone:  user.GetPhone(),
		Phones: JSON(GetUserablePhones(user)),
		Email:  user.GetEmail(),
		Emails: JSON(GetUserableEmails(user)),
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

func (l local) lookupUser(user Userable) {

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

func (l local) LookupEntryByExternalIdentity(extID ExternalIdentity) (Entry, error) {
	return nil, nil
}

func (l local) LookupEntryUserByExternalIdentity(extID ExternalIdentity) (UserEntryExtIDStoreable, error) {
	panic("not implemented") // TODO: Implement
}

func (l local) LookupEntryDepartmentByExternalIdentity(extID ExternalIdentity) (DepartmentEntryExtIDStoreable, error) {
	panic("not implemented") // TODO: Implement
}

type localUser struct {
	*local

	ID         uuid.UUID `gorm:"primaryKey"`
	Name       string
	Names      datatypes.JSON
	Phone      string
	Phones     datatypes.JSON
	Email      string
	Emails     datatypes.JSON
	ExtIDs     datatypes.JSON
	Departemts datatypes.JSON
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

func (u localUser) GetName() string {
	return u.Name
}

func (u localUser) GetNames() (names []string) {
	_ = json.Unmarshal(u.Names, &names)
	return names
}

func (u localUser) GetEmail() string {
	return u.Email
}

func (u localUser) GetEmails() (emails []string) {
	_ = json.Unmarshal(u.Emails, &emails)
	return emails
}

func (u localUser) GetPhone() string {
	return u.Phone
}

func (u localUser) GetPhones() (phones []string) {
	_ = json.Unmarshal(u.Phones, &phones)
	return phones
}

func (u localUser) GetExternalIdentities() (extIDs []ExternalIdentity) {
	_ = json.Unmarshal(u.ExtIDs, &extIDs)
	return extIDs
}

func (u *localUser) SetExternalIdentities(extIDs []ExternalIdentity) (err error) {
	u.ExtIDs, err = json.Marshal(extIDs)
	if err != nil {
		return err
	}
	return u.db.Save(u).Error
}

type localDepartment struct {
	*local

	ID       uuid.UUID `gorm:"primaryKey"`
	Name     string
	ParentID uuid.UUID
	ExtIDs   datatypes.JSON
}

func (u *localDepartment) BeforeCreate(_ *gorm.DB) error {
	if u.ID == uuid.Nil {
		u.ID = uuid.New()
	}
	return nil
}

func (d localDepartment) GetName() (name string) {
	return d.Name
}

func (d localDepartment) GetID() (departmentId string) {
	return d.ID.String()
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

func JSON(in any) (bytes datatypes.JSON) {
	bytes, _ = json.Marshal(in)
	return bytes
}
