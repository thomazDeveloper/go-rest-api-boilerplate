package user

import "time"

const (
	RoleGuest = "guest"
	RoleUser  = "user"
	RoleAdmin = "admin"
)

// Role represents a user role in the system
type Role struct {
	bun.BaseModel `bun:"table:roles,alias:r"`
	ID           int64           `bun:"id,pk,autoincrement" json:"id"`
	Name         string          `bun:"name,unique,notnull" json:"name"`
	Description string    `bun:"description" json:"description"`
	CreatedAt   time.Time `bun:"created_at,nullzero,notnull,default:current_timestamp" json:"created_at"`
	UpdatedAt   time.Time `bun:"updated_at,nullzero,notnull,default:current_timestamp" json:"updated_at"`
}
