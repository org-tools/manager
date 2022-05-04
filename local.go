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

func (l local) CreateUser(options UnionUserCreateOptions) (BasicUserable, error) {
	newUser := &localUser{
		Name:  options.GetUserName(),
		Phone: options.GetUserPhone(),
		Email: options.GetMailNickname(),
	}
	return newUser, l.db.Create(&newUser).Error
}

func (l local) LookupEntryUserByInternalExternalIdentity(internalExtID ExternalIdentity) (BasicUserable, error) {
	return l.lookupLocalUserByInternalExternalIdentity(internalExtID)
}

func (l local) lookupLocalUserByInternalExternalIdentity(internalExtID ExternalIdentity) (user *localUser, err error) {
	err = l.db.Where(&localUser{ID: uuid.MustParse(internalExtID.GetEntryID())}).Find(&user).Error
	return user, err
}

func (l local) LookupEntryDepartmentByInternalExternalIdentity(internalExtID ExternalIdentity) (UnionDepartment, error) {
	return l.lookupLocalDepartmentByInternalExternalIdentity(internalExtID)
}

func (l local) lookupLocalDepartmentByInternalExternalIdentity(internalExtID ExternalIdentity) (dept *localDepartment, err error) {
	err = l.db.Where(&localDepartment{ID: uuid.MustParse(internalExtID.GetEntryID())}).Find(&dept).Error
	return dept, err
}

func (l local) GetTargetSlug() string {
	return l.config.Slug
}

func (l local) GetPlatform() string {
	return l.config.Platform
}

func (l *local) GetRootDepartment() UnionDepartment {
	rootDepartment := new(localDepartment)
	rf := l.db.Where(&localDepartment{ID: l.config.RootDepartmentUUID}).Find(&rootDepartment).RowsAffected
	if rf == 0 {
		rootDepartment = &localDepartment{ID: l.config.RootDepartmentUUID, Name: "root"}
		l.db.Create(&rootDepartment)
	}
	rootDepartment.local = l
	return rootDepartment
}

func (l *local) GetAllUsers() (users []BasicUserable, err error) {
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

func (u localUser) GetID() (userId string) {
	return u.ID.String()
}

func (u localUser) GetName() (name string) {
	return u.Name
}

func (u localUser) GetEmailSet() (emails []string) {
	_ = json.Unmarshal(u.Emails, &emails)
	return emails
}

func (u localUser) GetExternalIdentities() (extIDs []ExternalIdentity) {
	for _, v := range u.ExtIDs {
		extIDs = append(extIDs, ExternalIdentity(v))
	}
	return extIDs
}

func (u *localUser) SetExternalIdentities(extIDs []ExternalIdentity) (err error) {
	extIDStringList := make([]string, 0)
	for _, v := range extIDs {
		extIDStringList = append(extIDStringList, string(v))
	}
	u.ExtIDs, err = json.Marshal(extIDStringList)
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

func (d localDepartment) GetChildDepartments() (departments []UnionDepartment) {
	localDepartments := make([]localDepartment, 0)
	d.db.Where(&localDepartment{ParentID: d.ID}).Find(&localDepartments)
	for _, v := range localDepartments {
		v.local = d.local
		departments = append(departments, &v)
	}
	return departments
}

func (d localDepartment) GetUsers() (users []BasicUserable) {
	localUsers := make([]localUser, 0)
	d.db.Model(&localUser{}).Where(datatypes.JSONQuery("departemts").HasKey(d.ID.String())).Find(&localUsers)
	for _, v := range localUsers {
		v.local = d.local
		users = append(users, &v)
	}
	return users
}
