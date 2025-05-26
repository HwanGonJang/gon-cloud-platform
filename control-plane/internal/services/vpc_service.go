package services

import (
	"fmt"
	"math"
	"net"
	"time"

	"github.com/google/uuid"

	"gon-cloud-platform/control-plane/internal/api/handlers/dto"
	"gon-cloud-platform/control-plane/internal/database/repositories"
	"gon-cloud-platform/control-plane/internal/models"
	"gon-cloud-platform/control-plane/internal/network"
	"gon-cloud-platform/control-plane/internal/utils"
	"gon-cloud-platform/control-plane/pkg/errors"
)

type VPCService interface {
	CreateVPC(userID string, req *dto.CreateVPCRequest) (*models.VPC, error)
	GetVPC(id string, userID string) (*models.VPC, error)
	ListVPCs(userID string, page, pageSize int) (*dto.VPCListResponse, error)
	UpdateVPC(id string, userID string, req *dto.UpdateVPCRequest) (*models.VPC, error)
	DeleteVPC(id string, userID string) error
}

type vpcService struct {
	vpcRepo    repositories.VPCRepository
	ovsManager network.OVSManager
	logger     *utils.Logger
}

func NewVPCService(vpcRepo repositories.VPCRepository, ovsManager network.OVSManager, logger *utils.Logger) VPCService {
	return &vpcService{
		vpcRepo:    vpcRepo,
		ovsManager: ovsManager,
		logger:     logger,
	}
}

func (s *vpcService) CreateVPC(userID string, req *dto.CreateVPCRequest) (*models.VPC, error) {
	s.logger.Info("Creating new VPC", "user_id", userID, "name", req.Name)

	// Validate CIDR block
	if err := s.validateCIDRBlock(req.CIDRBlock); err != nil {
		s.logger.Error("Invalid CIDR block", "error", err, "cidr", req.CIDRBlock)
		return nil, errors.ErrInvalidCIDR
	}

	// Check for name conflicts
	existingVPC, err := s.vpcRepo.GetByName(req.Name, userID)
	if err != nil {
		s.logger.Error("Failed to check name conflict", "error", err, "name", req.Name)
		return nil, errors.ErrVPCNotFound
	}
	if existingVPC != nil {
		s.logger.Warn("VPC name already exists", "name", req.Name, "user_id", userID)
		return nil, errors.ErrVPCAlreadyExists
	}

	// Create VPC model
	now := time.Now()
	vpc := &models.VPC{
		ID:          uuid.New().String(),
		Name:        req.Name,
		CIDRBlock:   req.CIDRBlock,
		Description: req.Description,
		UserID:      userID,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	// Create VPC in database
	if err := s.vpcRepo.Create(vpc); err != nil {
		s.logger.Error("Failed to create VPC in database", "error", err, "vpc_id", vpc.ID)
		return nil, errors.Wrap(err, errors.ErrorTypeInternal, "DB_ERROR", "Failed to create VPC")
	}

	// Create Open vSwitch bridge for VPC
	bridgeName := fmt.Sprintf("gcp-vpc-%s", vpc.ID[:8])
	if err := s.ovsManager.CreateBridge(bridgeName, vpc.CIDRBlock); err != nil {
		s.logger.Error("Failed to create OVS bridge", "error", err, "bridge_name", bridgeName)
		// Rollback database changes
		if delErr := s.vpcRepo.Delete(vpc.ID, userID); delErr != nil {
			s.logger.Error("Failed to rollback VPC creation", "error", delErr, "vpc_id", vpc.ID)
		}
		return nil, errors.Wrap(err, errors.ErrorTypeInternal, "OVS_ERROR", "Failed to create network bridge")
	}

	s.logger.Info("VPC created successfully", "vpc_id", vpc.ID, "name", vpc.Name)
	return vpc, nil
}

func (s *vpcService) GetVPC(id string, userID string) (*models.VPC, error) {
	s.logger.Info("Getting VPC", "vpc_id", id, "user_id", userID)

	vpc, err := s.vpcRepo.GetByID(id, userID)
	if err != nil {
		s.logger.Error("Failed to get VPC", "error", err, "vpc_id", id)
		return nil, errors.Wrap(err, errors.ErrorTypeInternal, "DB_ERROR", "Failed to get VPC")
	}
	if vpc == nil {
		s.logger.Warn("VPC not found", "vpc_id", id)
		return nil, errors.ErrVPCNotFound
	}

	s.logger.Info("VPC retrieved successfully", "vpc_id", id)
	return vpc, nil
}

func (s *vpcService) ListVPCs(userID string, page, pageSize int) (*dto.VPCListResponse, error) {
	s.logger.Info("Listing VPCs", "user_id", userID, "page", page, "page_size", pageSize)

	// Validate pagination parameters
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	vpcs, total, err := s.vpcRepo.List(userID, page, pageSize)
	if err != nil {
		s.logger.Error("Failed to list VPCs", "error", err, "user_id", userID)
		return nil, errors.Wrap(err, errors.ErrorTypeInternal, "DB_ERROR", "Failed to list VPCs")
	}

	// Convert to response format
	vpcResponses := make([]dto.VPCResponse, len(vpcs))
	for i, vpc := range vpcs {
		vpcResponses[i] = dto.ToVPCResponse(&vpc)
	}

	totalPages := int(math.Ceil(float64(total) / float64(pageSize)))

	s.logger.Info("VPCs listed successfully", "count", len(vpcs), "total", total)
	return &dto.VPCListResponse{
		VPCs:       vpcResponses,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	}, nil
}

func (s *vpcService) UpdateVPC(id string, userID string, req *dto.UpdateVPCRequest) (*models.VPC, error) {
	s.logger.Info("Updating VPC", "vpc_id", id, "user_id", userID)

	// Check if VPC exists
	existingVPC, err := s.vpcRepo.GetByID(id, userID)
	if err != nil {
		s.logger.Error("Failed to get VPC", "error", err, "vpc_id", id)
		return nil, errors.Wrap(err, errors.ErrorTypeInternal, "DB_ERROR", "Failed to get VPC")
	}
	if existingVPC == nil {
		s.logger.Warn("VPC not found", "vpc_id", id)
		return nil, errors.ErrVPCNotFound
	}

	// Build update map
	updates := make(map[string]interface{})

	if req.Name != nil && *req.Name != existingVPC.Name {
		// Check for name conflicts
		conflictVPC, err := s.vpcRepo.GetByName(*req.Name, userID)
		if err != nil {
			s.logger.Error("Failed to check name conflict", "error", err, "name", *req.Name)
			return nil, errors.Wrap(err, errors.ErrorTypeInternal, "DB_ERROR", "Failed to check VPC name")
		}
		if conflictVPC != nil && conflictVPC.ID != id {
			s.logger.Warn("VPC name already exists", "name", *req.Name)
			return nil, errors.ErrVPCAlreadyExists
		}
		updates["name"] = *req.Name
	}

	if req.Description != nil {
		updates["description"] = req.Description
	}

	if len(updates) == 0 {
		s.logger.Info("No updates provided", "vpc_id", id)
		return existingVPC, nil
	}

	// Update in database
	if err := s.vpcRepo.Update(id, userID, updates); err != nil {
		s.logger.Error("Failed to update VPC", "error", err, "vpc_id", id)
		return nil, errors.Wrap(err, errors.ErrorTypeInternal, "DB_ERROR", "Failed to update VPC")
	}

	// Return updated VPC
	updatedVPC, err := s.vpcRepo.GetByID(id, userID)
	if err != nil {
		s.logger.Error("Failed to get updated VPC", "error", err, "vpc_id", id)
		return nil, errors.Wrap(err, errors.ErrorTypeInternal, "DB_ERROR", "Failed to get updated VPC")
	}

	s.logger.Info("VPC updated successfully", "vpc_id", id)
	return updatedVPC, nil
}

func (s *vpcService) DeleteVPC(id string, userID string) error {
	s.logger.Info("Deleting VPC", "vpc_id", id, "user_id", userID)

	// Check if VPC exists
	vpc, err := s.vpcRepo.GetByID(id, userID)
	if err != nil {
		s.logger.Error("Failed to get VPC", "error", err, "vpc_id", id)
		return errors.Wrap(err, errors.ErrorTypeInternal, "DB_ERROR", "Failed to get VPC")
	}
	if vpc == nil {
		s.logger.Warn("VPC not found", "vpc_id", id)
		return errors.ErrVPCNotFound
	}

	// Delete OVS bridge
	bridgeName := fmt.Sprintf("gcp-vpc-%s", id[:8])
	if err := s.ovsManager.DeleteBridge(bridgeName); err != nil {
		s.logger.Error("Failed to delete OVS bridge", "error", err, "bridge_name", bridgeName)
		// Continue with deletion even if bridge deletion fails
	}

	// Delete from database
	if err := s.vpcRepo.Delete(id, userID); err != nil {
		s.logger.Error("Failed to delete VPC", "error", err, "vpc_id", id)
		return errors.Wrap(err, errors.ErrorTypeInternal, "DB_ERROR", "Failed to delete VPC")
	}

	s.logger.Info("VPC deleted successfully", "vpc_id", id)
	return nil
}

// validateCIDRBlock validates that the CIDR block is valid and within allowed ranges
func (s *vpcService) validateCIDRBlock(cidr string) error {
	_, ipNet, err := net.ParseCIDR(cidr)
	if err != nil {
		return errors.Wrap(err, errors.ErrorTypeValidation, "INVALID_CIDR", "Invalid CIDR format")
	}

	// Check if it's a private IP range (RFC 1918)
	privateRanges := []string{
		"10.0.0.0/8",     // 10.0.0.0 - 10.255.255.255
		"172.16.0.0/12",  // 172.16.0.0 - 172.31.255.255
		"192.168.0.0/16", // 192.168.0.0 - 192.168.255.255
	}

	isPrivate := false
	for _, privateRange := range privateRanges {
		_, privateNet, _ := net.ParseCIDR(privateRange)
		if privateNet.Contains(ipNet.IP) {
			isPrivate = true
			break
		}
	}

	if !isPrivate {
		return errors.New(errors.ErrorTypeValidation, "INVALID_CIDR", "CIDR block must be within private IP ranges (RFC 1918)")
	}

	// Check prefix length (minimum /28, maximum /16)
	ones, _ := ipNet.Mask.Size()
	if ones < 16 || ones > 28 {
		return errors.New(errors.ErrorTypeValidation, "INVALID_CIDR", "CIDR block prefix must be between /16 and /28")
	}

	return nil
}
