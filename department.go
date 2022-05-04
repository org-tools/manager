package orgmanager

type Departmentable interface {
	GetName() (name string)
}

type DepartmentableEntry interface {
	Entry
	Departmentable
	GetChildDepartments() (departments []DepartmentableEntry)
	CreateChildDepartment(department Departmentable) (DepartmentableEntry, error)
	GetUsers() (users []UserableEntry)
}

type Department struct {
	Name string
}

func (d Department) GetName() string {
	return d.Name
}

func (d Department) GetChildDepartments() (departments []DepartmentableEntry) {
	return departments
}

func (d Department) CreateChildDepartment(department Departmentable) (DepartmentableEntry, error) {
	return nil, nil
}

func (d Department) GetUsers() (users []UserableEntry) {
	return users
}

type DepartmentModifyUserOptions struct {
	Role DepartmentUserRole
}

type DepartmentUserWriter interface {
	AddToDepartment(options DepartmentModifyUserOptions, extID ExternalIdentity) error
	// RemoveFromDepartment(options DepartmentModifyUserOptions, extID ExternalIdentity) error
}
