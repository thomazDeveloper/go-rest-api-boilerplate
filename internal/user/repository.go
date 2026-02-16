package user

import (
	"context"
	"errors"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type txKey struct{}

// Repository defines user repository interface
type Repository interface {
	Create(ctx context.Context, user *User) error
	FindByEmail(ctx context.Context, email string) (*User, error)
	FindByID(ctx context.Context, id uint) (*User, error)
	Update(ctx context.Context, user *User) error
	Delete(ctx context.Context, id uint) error
	ListAllUsers(ctx context.Context, filters UserFilterParams, page, perPage int) ([]User, int64, error)
	AssignRole(ctx context.Context, userID uint, roleName string) error
	RemoveRole(ctx context.Context, userID uint, roleName string) error
	FindRoleByName(ctx context.Context, name string) (*Role, error)
	GetUserRoles(ctx context.Context, userID uint) ([]Role, error)
	Transaction(ctx context.Context, fn func(context.Context) error) error
}

type repository struct {
	db *bun.DB
}

// NewRepository creates a new user repository
func NewRepository(db *bun.DB) Repository {
	return &repository{db: db}
}

// getDB returns the DB from context if in transaction, otherwise returns the repository's DB
func (r *repository) getDB(ctx context.Context) *bun.DB {
	if tx, ok := ctx.Value(txKey{}).(*bun.DB); ok {
		return tx
	}
	return r.db
}

// Create creates a new user in the database
func (r *repository) Create(ctx context.Context, user *User) error {
	_, err := r.getDB(ctx).NewInsert().Model(user).Exec(ctx)
	if err != nil {
		return err
	}
	return nil
}

// FindByEmail finds a user by email
func (r *repository) FindByEmail(ctx context.Context, email string) (*User, error) {
	user := new(User)
	err := r.getDB(ctx).NewSelect().Model(user).Relation("Roles").Where("email = ?", email).Scan(ctx)

	if err != nil {
		 if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}

// FindByID finds a user by ID
func (r *repository) FindByID(ctx context.Context, id uint) (*User, error) {
	user := new(User)
	err := r.getDB(ctx).NewSelect().Model(user).Relation("Roles").Where("id = ?", id).Scan(ctx)
	if err != nil {
		 if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}

// Update updates a user in the database
func (r *repository) Update(ctx context.Context, user *User) error {
	// WHY: Save() syncs associations, potentially clearing roles
	_, err  := r.getDB(ctx).NewUpdate().
	Model(user).
	Column("name", "email", "password_hash", "updated_at").
	Where("id = ?", user.ID).
	Exec(ctx)
	if err != nil {
		return err
	}
	return nil
}

// Delete soft deletes a user from the database
func (r *repository) Delete(ctx context.Context, id uint) error {
	_, err  := r.getDB(ctx).NewDelete().Model((*User)(nil)).Where("id = ?", id).Exec(ctx)
	if err != nil {
		return err
	}
	
	return nil
}

// ListAllUsers retrieves paginated list of users with filters
func (r *repository) ListAllUsers(ctx context.Context, filters UserFilterParams, page, perPage int) ([]User, int64, error) {
	var users []User
	var total int64

	query := r.getDB(ctx).NewSelect().Model(&users).Relation("Roles")

	if filters.Role != "" {
		query = query.Joins("JOIN user_roles ON user_roles.user_id = users.id").
			Joins("JOIN roles ON roles.id = user_roles.role_id").
			Where("roles.name = ?", filters.Role)
	}

	if filters.Search != "" {
		// WHY: Escape SQL LIKE wildcards to prevent incorrect matches
		escapedSearch := strings.ReplaceAll(filters.Search, "%", "\\%")
		escapedSearch = strings.ReplaceAll(escapedSearch, "_", "\\_")
		searchPattern := "%" + escapedSearch + "%"
		query = query.Where("users.name LIKE ? OR users.email LIKE ?", searchPattern, searchPattern)
	}

	// WHY: Count distinct user IDs when using JOINs to avoid inflated totals
	if err := query.Distinct("users.id").Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * perPage

	// Defense-in-depth: Validate sort parameters at repository layer
	validSorts := map[string]bool{
		"name": true, "email": true, "created_at": true, "updated_at": true,
	}
	if !validSorts[filters.Sort] {
		return nil, 0, errors.New("invalid sort field")
	}
	if filters.Order != "asc" && filters.Order != "desc" {
		return nil, 0, errors.New("invalid sort order")
	}

	// Use type-safe GORM clause to prevent SQL injection
	orderColumn := fmt.Sprintf("%s %s", filters.Sort, filters.Order)

	total, err := query.Distinct().Order(orderColumn).Limit(perPage).Offset(offset).ScanAndCount(ctx) 
	if err != nil {
		return nil, 0, err
	}

	return users, total, nil
}

// AssignRole assigns a role to a user
func (r *repository) AssignRole(ctx context.Context, userID uint, roleName string) error {
	role, err := r.FindRoleByName(ctx, roleName)
	if err != nil {
		return err
	}
	if role == nil {
		return errors.New("role not found")
	}

	qr := "INSERT INTO user_roles (user_id, role_id, assigned_at) VALUES (?, ?, ?) ON CONFLICT (user_id, role_id) DO NOTHING"

	// Use database-level conflict handling for race-safe, idempotent role assignment
	// Works with both PostgreSQL and SQLite
	    _, err =  r.getDB(ctx).ExecContext(ctx, qr, userID, role.ID, time.Now())
		return err
}

// RemoveRole removes a role from a user
func (r *repository) RemoveRole(ctx context.Context, userID uint, roleName string) error {
	role, err := r.FindRoleByName(ctx, roleName)
	if err != nil {
		return err
	}
	if role == nil {
		return errors.New("role not found")
	}

	qr := "DELETE FROM user_roles WHERE user_id = ? AND role_id = ?"

	 _, err = r.getDB(ctx).ExecContext(ctx, qr, userID, role.ID)
	 return err
}

// FindRoleByName finds a role by name
func (r *repository) FindRoleByName(ctx context.Context, name string) (*Role, error) {
	role := new(Role)
	err := r.getDB(ctx).NewSelect().Model(role).Where("name = ?", name).Scan(ctx)
	if err != nil {
		 if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &role, nil
}

// GetUserRoles retrieves all roles for a user
func (r *repository) GetUserRoles(ctx context.Context, userID uint) ([]Role, error) {
	var roles []Role
	err := r.getDB(ctx).NewSelect().
   		 Model(&roles).
		Joins("JOIN user_roles ON user_roles.role_id = roles.id").
		Where("user_roles.user_id = ?", userID).
		Scan(ctx)
	if err != nil {
		return nil, err
	}
	return roles, nil
}

// Transaction executes a function within a database transaction
func (r *repository) CreateAndAssignRole(ctx context.Context, user *User, roleName string) error {
		tx, err := r.getDB(ctx).BeginTx(ctx, &sql.TxOptions{})
		txCtx := context.WithValue(ctx, txKey{}, tx)
		if err := r.Create(txCtx, user); err != nil {
			return err
		}
		return r.AssignRole(txCtx, user.ID, roleName)
}
