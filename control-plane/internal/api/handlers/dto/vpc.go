package dto

import (
	"time"

	"gon-cloud-platform/control-plane/internal/models"
)

type CreateVPCRequest struct {
	Name        string  `json:"name" binding:"required,min=1,max=255"`
	CIDRBlock   string  `json:"cidr_block" binding:"required"`
	Description *string `json:"description,omitempty"`
}

type UpdateVPCRequest struct {
	Name        *string `json:"name,omitempty" binding:"omitempty,min=1,max=255"`
	Description *string `json:"description,omitempty"`
}

type VPCResponse struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	CIDRBlock   string    `json:"cidr_block"`
	Description *string   `json:"description"`
	UserID      string    `json:"user_id"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type VPCListResponse struct {
	VPCs       []VPCResponse `json:"vpcs"`
	Total      int           `json:"total"`
	Page       int           `json:"page"`
	PageSize   int           `json:"page_size"`
	TotalPages int           `json:"total_pages"`
}

// Convert VPC model to response
func ToVPCResponse(v *models.VPC) VPCResponse {
	return VPCResponse{
		ID:          v.ID,
		Name:        v.Name,
		CIDRBlock:   v.CIDRBlock,
		Description: v.Description,
		UserID:      v.UserID,
		CreatedAt:   v.CreatedAt,
		UpdatedAt:   v.UpdatedAt,
	}
}
