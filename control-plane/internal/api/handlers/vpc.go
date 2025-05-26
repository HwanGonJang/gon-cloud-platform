package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"gon-cloud-platform/control-plane/internal/api/handlers/dto"
	"gon-cloud-platform/control-plane/internal/services"
	"gon-cloud-platform/control-plane/internal/utils"
	"gon-cloud-platform/control-plane/pkg/errors"
	"gon-cloud-platform/control-plane/pkg/response"
)

type VPCHandler struct {
	vpcService services.VPCService
	logger     *utils.Logger
}

func NewVPCHandler(vpcService services.VPCService, logger *utils.Logger) *VPCHandler {
	return &VPCHandler{
		vpcService: vpcService,
		logger:     logger,
	}
}

// CreateVPC godoc
// @Summary Create a new VPC
// @Description Create a new Virtual Private Cloud
// @Tags VPC
// @Accept json
// @Produce json
// @Param vpc body models.CreateVPCRequest true "VPC creation request"
// @Success 201 {object} response.Response{data=models.VPCResponse}
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /api/v1/vpcs [post]
func (h *VPCHandler) CreateVPC(c *gin.Context) {
	var req dto.CreateVPCRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, errors.ErrInvalidRequest, err.Error())
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		response.Error(c, http.StatusUnauthorized, errors.ErrUnauthorized, "User not authenticated")
		return
	}

	userIDStr, ok := userID.(string)
	if !ok {
		response.Error(c, http.StatusInternalServerError, errors.ErrInvalidID, "Invalid user ID format")
		return
	}

	vpc, err := h.vpcService.CreateVPC(userIDStr, &req)
	if err != nil {
		switch err {
		case errors.ErrVPCAlreadyExists:
			response.Error(c, http.StatusConflict, err, "VPC already exists")
		case errors.ErrInvalidCIDR:
			response.Error(c, http.StatusBadRequest, err, "Invalid CIDR block")
		case errors.ErrCIDRConflict:
			response.Error(c, http.StatusConflict, err, "CIDR block conflicts with existing VPC")
		default:
			response.Error(c, http.StatusInternalServerError, errors.ErrInternalServer, err.Error())
		}
		return
	}

	response.Success(c, http.StatusCreated, "VPC created successfully", dto.ToVPCResponse(vpc))
}

// GetVPC godoc
// @Summary Get VPC by ID
// @Description Get a specific VPC by its ID
// @Tags VPC
// @Produce json
// @Param id path string true "VPC ID"
// @Success 200 {object} response.Response{data=models.VPCResponse}
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Failure 404 {object} response.Response
// @Router /api/v1/vpcs/{id} [get]
func (h *VPCHandler) GetVPC(c *gin.Context) {
	idStr := c.Param("id")

	userID, exists := c.Get("user_id")
	if !exists {
		response.Error(c, http.StatusUnauthorized, errors.ErrUnauthorized, "User not authenticated")
		return
	}

	userIDStr, ok := userID.(string)
	if !ok {
		response.Error(c, http.StatusInternalServerError, errors.ErrInvalidID, "Invalid user ID format")
		return
	}

	vpc, err := h.vpcService.GetVPC(idStr, userIDStr)
	if err != nil {
		switch err {
		case errors.ErrVPCNotFound:
			response.Error(c, http.StatusNotFound, err, "VPC not found")
		case errors.ErrUnauthorized:
			response.Error(c, http.StatusUnauthorized, err, "You don't have permission to access this VPC")
		default:
			response.Error(c, http.StatusInternalServerError, errors.ErrInternalServer, err.Error())
		}
		return
	}

	response.Success(c, http.StatusOK, "VPC retrieved successfully", dto.ToVPCResponse(vpc))
}

// ListVPCs godoc
// @Summary List VPCs
// @Description Get a paginated list of VPCs for the authenticated user
// @Tags VPC
// @Produce json
// @Param page query int false "Page number" default(1)
// @Param page_size query int false "Number of items per page" default(20)
// @Success 200 {object} response.Response{data=models.VPCListResponse}
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /api/v1/vpcs [get]
func (h *VPCHandler) ListVPCs(c *gin.Context) {
	// Parse pagination parameters
	page := 1
	pageSize := 20

	if pageStr := c.Query("page"); pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}

	if pageSizeStr := c.Query("page_size"); pageSizeStr != "" {
		if ps, err := strconv.Atoi(pageSizeStr); err == nil && ps > 0 && ps <= 100 {
			pageSize = ps
		}
	}

	userID, exists := c.Get("user_id")
	if !exists {
		response.Error(c, http.StatusUnauthorized, errors.ErrUnauthorized, "User not authenticated")
		return
	}

	userIDStr, ok := userID.(string)
	if !ok {
		response.Error(c, http.StatusInternalServerError, errors.ErrInvalidID, "Invalid user ID format")
		return
	}

	vpcList, err := h.vpcService.ListVPCs(userIDStr, page, pageSize)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, errors.ErrInternalServer, err.Error())
		return
	}

	response.Success(c, http.StatusOK, "VPCs retrieved successfully", vpcList)
}

// UpdateVPC godoc
// @Summary Update VPC
// @Description Update an existing VPC
// @Tags VPC
// @Accept json
// @Produce json
// @Param id path string true "VPC ID"
// @Param vpc body models.UpdateVPCRequest true "VPC update request"
// @Success 200 {object} response.Response{data=models.VPCResponse}
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Failure 404 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /api/v1/vpcs/{id} [put]
func (h *VPCHandler) UpdateVPC(c *gin.Context) {
	idStr := c.Param("id")

	var req dto.UpdateVPCRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, errors.ErrInvalidRequest, err.Error())
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		response.Error(c, http.StatusUnauthorized, errors.ErrUnauthorized, "User not authenticated")
		return
	}

	userIDStr, ok := userID.(string)
	if !ok {
		response.Error(c, http.StatusInternalServerError, errors.ErrInvalidID, "Invalid user ID format")
		return
	}

	vpc, err := h.vpcService.UpdateVPC(idStr, userIDStr, &req)
	if err != nil {
		switch err {
		case errors.ErrVPCNotFound:
			response.Error(c, http.StatusNotFound, err, "VPC not found")
		case errors.ErrUnauthorized:
			response.Error(c, http.StatusUnauthorized, err, "You don't have permission to update this VPC")
		default:
			response.Error(c, http.StatusInternalServerError, errors.ErrInternalServer, err.Error())
		}
		return
	}

	response.Success(c, http.StatusOK, "VPC updated successfully", dto.ToVPCResponse(vpc))
}

// DeleteVPC godoc
// @Summary Delete VPC
// @Description Delete an existing VPC
// @Tags VPC
// @Produce json
// @Param id path string true "VPC ID"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Failure 404 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /api/v1/vpcs/{id} [delete]
func (h *VPCHandler) DeleteVPC(c *gin.Context) {
	idStr := c.Param("id")

	userID, exists := c.Get("user_id")
	if !exists {
		response.Error(c, http.StatusUnauthorized, errors.ErrUnauthorized, "User not authenticated")
		return
	}

	userIDStr, ok := userID.(string)
	if !ok {
		response.Error(c, http.StatusInternalServerError, errors.ErrInvalidID, "Invalid user ID format")
		return
	}

	err := h.vpcService.DeleteVPC(idStr, userIDStr)
	if err != nil {
		switch err {
		case errors.ErrVPCNotFound:
			response.Error(c, http.StatusNotFound, err, "VPC not found")
		case errors.ErrUnauthorized:
			response.Error(c, http.StatusUnauthorized, err, "You don't have permission to delete this VPC")
		default:
			response.Error(c, http.StatusInternalServerError, errors.ErrInternalServer, err.Error())
		}
		return
	}

	response.Success(c, http.StatusOK, "VPC deleted successfully", nil)
}
