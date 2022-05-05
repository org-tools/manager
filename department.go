package orgmanager

type Departmentable interface {
	GetName() string
	GetDescription() string
}

type DepartmentableEntry interface {
	Entry
	Departmentable
	GetChildDepartments() (departments []DepartmentableEntry)
	CreateChildDepartment(department Departmentable) (DepartmentableEntry, error)
	GetUsers() (users []UserableEntry)
}

type Department struct {
	Name        string
	Description string
}

func (d Department) GetName() string {
	return d.Name
}

func (d Department) GetDescription() string {
	return d.Description
}

type DepartmentModifyUserOptions struct {
	Role DepartmentUserRole
}

type DepartmentUserWriter interface {
	AddToDepartment(options DepartmentModifyUserOptions, extID ExternalIdentity) error
	// RemoveFromDepartment(options DepartmentModifyUserOptions, extID ExternalIdentity) error
}
