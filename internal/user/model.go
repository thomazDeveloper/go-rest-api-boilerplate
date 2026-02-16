package user

import (
	"time"

	"gorm.io/gorm"
)

// User represents a user in the system
type User struct {
	bun.BaseModel `bun:"table:users,alias:u"`
	ID           int64           `bun:"id,pk,autoincrement" json:"id"`
	Name         string         `bun:"name,notnull" json:"name"`
	Email        string         `bun:"email,uniqueIndex,notnull" json:"email"`
	PasswordHash string         `bun:"password_hash,notnull" json:"-"`
	Roles        []Role         `bun:"m2m:user_roles,join:User=Role" json:"-"`
	CreatedAt time.Time `bun:"created_at,nullzero,notnull,default:current_timestamp" json:"created_at"`
    UpdatedAt time.Time `bun:"updated_at,nullzero,notnull,default:current_timestamp" json:"updated_at"`
    DeletedAt bun.NullTime `bun:"deleted_at,soft_delete" json:"-"`
}

// HasRole checks if user has specific role
func (u *User) HasRole(roleName string) bool {
	for _, role := range u.Roles {
		if role.Name == roleName {
			return true
		}
	}
	return false
}

// IsAdmin checks if user has admin role
func (u *User) IsAdmin() bool {
	return u.HasRole(RoleAdmin)
}

// GetRoleNames returns list of role names
func (u *User) GetRoleNames() []string {
	roleNames := make([]string, len(u.Roles))
	for i, role := range u.Roles {
		roleNames[i] = role.Name
	}
	return roleNames
}
