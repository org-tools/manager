package manager

type Departmentable interface {
	GetName() string
	GetDescription() string
}

type DepartmentableEntry interface {
	Entry
	Departmentable
	GetChildDepartments() (departments []DepartmentableEntry)
	CreateChildDepartment(departmentable Departmentable) (DepartmentableEntry, error)
	GetUsers() (users []UserableEntry, err error)
}

func NewDepartment() *department {
	return new(department)
}

type department struct {
	Name        string
	Description string
}

func (d department) GetName() string {
	return d.Name
}

func (d department) GetDescription() string {
	return d.Description
}

type DepartmentModifyUserOptions struct {
	Role DepartmentUserRole
}

type DepartmentUserWriter interface {
	AddToDepartment(options DepartmentModifyUserOptions, extID ExternalIdentity) error
	RemoveFromDepartment(options DepartmentModifyUserOptions, extID ExternalIdentity) error
}
