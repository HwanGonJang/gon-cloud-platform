package repositories

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"

	"gon-cloud-platform/control-plane/internal/models"
	"gon-cloud-platform/control-plane/pkg/errors"
)

type UserRepository interface {
	Create(user *models.User) (*models.User, error)
	GetByID(id string) (*models.User, error)
	GetByEmail(email string) (*models.User, error)
	GetByUsername(username string) (*models.User, error)
	Update(user *models.User) error
	UpdateLastLogin(id string) error
	Delete(id string) error
	List(offset, limit int) ([]*models.User, int64, error)
	SetActive(id string, active bool) error
	StoreRefreshToken(userID string, token string) error
	VerifyRefreshToken(userID string, token string) (bool, error)
	InvalidateRefreshToken(userID string, token string) error
}

type userRepository struct {
	db *sqlx.DB
}

func NewUserRepository(db *sqlx.DB) UserRepository {
	return &userRepository{db: db}
}

func (r *userRepository) Create(user *models.User) (*models.User, error) {
	query := `
		INSERT INTO users (id, username, email, password, role, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, username, email, role, is_active, created_at, updated_at`

	err := r.db.Get(user, query,
		user.ID,
		user.Username,
		user.Email,
		user.Password,
		user.Role,
		user.IsActive,
		user.RefreshToken,
		user.CreatedAt,
		user.UpdatedAt,
	)

	if err != nil {
		if isDuplicateError(err) {
			return nil, errors.ErrUserAlreadyExists
		}
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	return user, nil
}

func (r *userRepository) GetByID(id string) (*models.User, error) {
	query := `
		SELECT id, username, email, password, role, is_active, created_at, updated_at
		FROM users 
		WHERE id = $1 AND is_active = true`

	user := &models.User{}
	err := r.db.Get(user, query, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to get user by ID: %w", err)
	}

	return user, nil
}

func (r *userRepository) GetByEmail(email string) (*models.User, error) {
	query := `
		SELECT id, username, email, password, role, is_active, created_at, updated_at
		FROM users 
		WHERE email = $1 AND is_active = true`

	user := &models.User{}
	err := r.db.Get(user, query, email)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to get user by email: %w", err)
	}

	return user, nil
}

func (r *userRepository) GetByUsername(username string) (*models.User, error) {
	query := `
		SELECT id, username, email, password, role, is_active, created_at, updated_at
		FROM users 
		WHERE username = $1 AND is_active = true`

	user := &models.User{}
	err := r.db.Get(user, query, username)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to get user by username: %w", err)
	}

	return user, nil
}

func (r *userRepository) Update(user *models.User) error {
	query := `
		UPDATE users 
		SET username = $2, email = $3, password = $4, role = $5, updated_at = $6
		WHERE id = $1 AND is_active = true`

	user.UpdatedAt = time.Now()

	result, err := r.db.Exec(query,
		user.ID,
		user.Username,
		user.Email,
		user.Password,
		user.Role,
		user.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return errors.ErrUserNotFound
	}

	return nil
}

func (r *userRepository) UpdateLastLogin(id string) error {
	query := `UPDATE users SET updated_at = $2 WHERE id = $1 AND is_active = true`

	result, err := r.db.Exec(query, id, time.Now())
	if err != nil {
		return fmt.Errorf("failed to update last login: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return errors.ErrUserNotFound
	}

	return nil
}

func (r *userRepository) Delete(id string) error {
	// Soft delete by setting is_active to false
	query := `UPDATE users SET is_active = false, updated_at = $2 WHERE id = $1`

	result, err := r.db.Exec(query, id, time.Now())
	if err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return errors.ErrUserNotFound
	}

	return nil
}

func (r *userRepository) List(offset, limit int) ([]*models.User, int64, error) {
	var users []*models.User
	var total int64

	// Get total count
	countQuery := "SELECT COUNT(*) FROM users WHERE is_active = true"
	err := r.db.Get(&total, countQuery)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count users: %w", err)
	}

	// Get paginated results
	query := `
		SELECT id, username, email, role, is_active, created_at, updated_at
		FROM users 
		WHERE is_active = true
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2`

	err = r.db.Select(&users, query, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list users: %w", err)
	}

	return users, total, nil
}

func (r *userRepository) SetActive(id string, active bool) error {
	query := `UPDATE users SET is_active = $2, updated_at = $3 WHERE id = $1`

	result, err := r.db.Exec(query, id, active, time.Now())
	if err != nil {
		return fmt.Errorf("failed to set user active status: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return errors.ErrUserNotFound
	}

	return nil
}

func (r *userRepository) StoreRefreshToken(userID string, token string) error {
	query := `UPDATE users SET refresh_token = $2, updated_at = $3 WHERE id = $1`

	result, err := r.db.Exec(query, userID, token, time.Now())
	if err != nil {
		return fmt.Errorf("failed to store refresh token: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return errors.ErrUserNotFound
	}

	return nil
}

func (r *userRepository) VerifyRefreshToken(userID string, token string) (bool, error) {
	query := `SELECT refresh_token FROM users WHERE id = $1 AND is_active = true`

	var storedToken string
	err := r.db.Get(&storedToken, query, userID)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, errors.ErrUserNotFound
		}
		return false, fmt.Errorf("failed to verify refresh token: %w", err)
	}

	return storedToken == token, nil
}

func (r *userRepository) InvalidateRefreshToken(userID string, token string) error {
	query := `UPDATE users SET refresh_token = NULL, updated_at = $2 WHERE id = $1 AND refresh_token = $3`

	result, err := r.db.Exec(query, userID, time.Now(), token)
	if err != nil {
		return fmt.Errorf("failed to invalidate refresh token: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return errors.ErrRefreshTokenNotFound
	}

	return nil
}

// Helper function to check if error is a duplicate key error
func isDuplicateError(err error) bool {
	// PostgreSQL unique violation error code
	return err != nil && (err.Error() == "pq: duplicate key value violates unique constraint \"users_email_key\"" ||
		err.Error() == "pq: duplicate key value violates unique constraint \"users_username_key\"")
}
