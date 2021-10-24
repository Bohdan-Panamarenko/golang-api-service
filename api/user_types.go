package api

type Role int

const (
	userRole Role = iota
	adminRole
	superadminRole
)

func (r Role) String() string {
	switch r {
	case userRole:
		return "user"
	case adminRole:
		return "admin"
	case superadminRole:
		return "superadmin"
	}
	return "unknown"
}
