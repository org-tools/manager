package manager

type DepartmentUserRole uint

const (
	DepartmentUserRoleMember DepartmentUserRole = iota
	DepartmentUserRoleAdmin
)

type DepartmentUserAction uint

const (
	DepartmentUserActionSet DepartmentUserAction = iota
	DepartmentUserActionAdd
	DepartmentUserActionDelete
)
