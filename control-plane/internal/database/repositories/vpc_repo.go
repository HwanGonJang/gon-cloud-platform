// control-plane/internal/database/repositories/vpc_repo.go
package repositories

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"

	"gon-cloud-platform/control-plane/internal/models"
)

type VPCRepository interface {
	Create(vpc *models.VPC) error
	GetByID(id string, userID string) (*models.VPC, error)
	GetByName(name string, userID string) (*models.VPC, error)
	List(userID string, page, pageSize int) ([]models.VPC, int, error)
	Update(id string, userID string, updates map[string]interface{}) error
	Delete(id string, userID string) error
	CheckCIDRConflict(cidrBlock string, userID string, excludeID *string) (bool, error)
}

type vpcRepository struct {
	db *sqlx.DB
}

func NewVPCRepository(db *sqlx.DB) VPCRepository {
	return &vpcRepository{db: db}
}

func (r *vpcRepository) Create(vpc *models.VPC) error {
	query := `
		INSERT INTO vpcs (id, name, cidr_block, description, user_id, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`

	_, err := r.db.Exec(query,
		vpc.ID,
		vpc.Name,
		vpc.CIDRBlock,
		vpc.Description,
		vpc.UserID,
		vpc.CreatedAt,
		vpc.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to create VPC: %w", err)
	}

	return nil
}

func (r *vpcRepository) GetByID(id string, userID string) (*models.VPC, error) {
	var vpc models.VPC
	query := `
		SELECT id, name, cidr_block, description, user_id, created_at, updated_at
		FROM vpcs 
		WHERE id = $1 AND user_id = $2
	`

	err := r.db.Get(&vpc, query, id, userID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get VPC by ID: %w", err)
	}

	return &vpc, nil
}

func (r *vpcRepository) GetByName(name string, userID string) (*models.VPC, error) {
	var vpc models.VPC
	query := `
		SELECT id, name, cidr_block, description, user_id, created_at, updated_at
		FROM vpcs 
		WHERE name = $1 AND user_id = $2
	`

	err := r.db.Get(&vpc, query, name, userID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get VPC by name: %w", err)
	}

	return &vpc, nil
}

func (r *vpcRepository) List(userID string, page, pageSize int) ([]models.VPC, int, error) {
	var vpcs []models.VPC
	var total int

	// Get total count
	countQuery := "SELECT COUNT(*) FROM vpcs WHERE user_id = $1"
	err := r.db.Get(&total, countQuery, userID)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count VPCs: %w", err)
	}

	// Get paginated results
	offset := (page - 1) * pageSize
	query := `
		SELECT id, name, cidr_block, description, user_id, created_at, updated_at
		FROM vpcs 
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	err = r.db.Select(&vpcs, query, userID, pageSize, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list VPCs: %w", err)
	}

	return vpcs, total, nil
}

func (r *vpcRepository) Update(id string, userID string, updates map[string]interface{}) error {
	if len(updates) == 0 {
		return nil
	}

	// Build dynamic update query
	setParts := make([]string, 0, len(updates)+1)
	args := make([]interface{}, 0, len(updates)+3)
	argIndex := 1

	for field, value := range updates {
		setParts = append(setParts, fmt.Sprintf("%s = $%d", field, argIndex))
		args = append(args, value)
		argIndex++
	}

	// Always update updated_at
	setParts = append(setParts, fmt.Sprintf("updated_at = $%d", argIndex))
	args = append(args, time.Now())
	argIndex++

	// Add WHERE conditions
	args = append(args, id, userID)

	query := fmt.Sprintf(`
		UPDATE vpcs 
		SET %s 
		WHERE id = $%d AND user_id = $%d
	`, strings.Join(setParts, ", "), argIndex-1, argIndex)

	result, err := r.db.Exec(query, args...)
	if err != nil {
		return fmt.Errorf("failed to update VPC: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("VPC not found or no permission")
	}

	return nil
}

func (r *vpcRepository) Delete(id string, userID string) error {
	query := "DELETE FROM vpcs WHERE id = $1 AND user_id = $2"

	result, err := r.db.Exec(query, id, userID)
	if err != nil {
		return fmt.Errorf("failed to delete VPC: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("VPC not found or no permission")
	}

	return nil
}

func (r *vpcRepository) CheckCIDRConflict(cidrBlock string, userID string, excludeID *string) (bool, error) {
	query := `
		SELECT COUNT(*) 
		FROM vpcs 
		WHERE user_id = $1 AND cidr_block = $2
	`
	args := []interface{}{userID, cidrBlock}

	if excludeID != nil {
		query += " AND id != $3"
		args = append(args, *excludeID)
	}

	var count int
	err := r.db.Get(&count, query, args...)
	if err != nil {
		return false, fmt.Errorf("failed to check CIDR conflict: %w", err)
	}

	return count > 0, nil
}
