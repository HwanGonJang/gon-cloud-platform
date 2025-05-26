package handlers

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v4"

	"gon-cloud-platform/control-plane/internal/api/handlers/dto"
	"gon-cloud-platform/control-plane/internal/models"
	"gon-cloud-platform/control-plane/internal/services"
	"gon-cloud-platform/control-plane/internal/utils"
	"gon-cloud-platform/control-plane/pkg/errors"
	"gon-cloud-platform/control-plane/pkg/response"
)

type AuthHandler struct {
	authService services.AuthService
	config      *utils.Config
	logger      *utils.Logger
}

func NewAuthHandler(authService services.AuthService, config *utils.Config, logger *utils.Logger) *AuthHandler {
	return &AuthHandler{
		authService: authService,
		config:      config,
		logger:      logger,
	}
}

// Login godoc
// @Summary User login
// @Description Authenticate user with email and password
// @Tags auth
// @Accept json
// @Produce json
// @Param request body LoginRequest true "Login credentials"
// @Success 200 {object} response.APIResponse{data=AuthResponse}
// @Failure 400 {object} response.APIResponse
// @Failure 401 {object} response.APIResponse
// @Failure 500 {object} response.APIResponse
// @Router /api/v1/auth/login [post]
func (h *AuthHandler) Login(c *gin.Context) {
	var req dto.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("Invalid login request", "error", err)
		response.Error(c, http.StatusBadRequest, errors.ErrInvalidRequest, err.Error())
		return
	}

	// Use auth service for login
	user, err := h.authService.Login(req.Email, req.Password)
	if err != nil {
		h.logger.Error("Login failed", "email", req.Email, "error", err)
		response.Error(c, http.StatusUnauthorized, errors.ErrInvalidCredentials, "Invalid email or password")
		return
	}

	// Generate tokens
	accessToken, refreshToken, expiresIn, err := h.generateTokens(user)
	if err != nil {
		h.logger.Error("Failed to generate tokens", "user_id", user.ID, "error", err)
		response.Error(c, http.StatusInternalServerError, errors.ErrInternalServer, "Failed to generate authentication tokens")
		return
	}

	h.authService.StoreRefreshToken(user.ID, refreshToken)

	authResponse := dto.AuthResponse{
		User: dto.UserResponse{
			ID:        user.ID,
			Username:  user.Username,
			Email:     user.Email,
			CreatedAt: user.CreatedAt,
		},
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    expiresIn,
	}

	h.logger.Info("User logged in successfully", "user_id", user.ID, "email", user.Email)
	response.Success(c, http.StatusOK, "Login successful", authResponse)
}

// Register godoc
// @Summary User registration
// @Description Register a new user account
// @Tags auth
// @Accept json
// @Produce json
// @Param request body RegisterRequest true "Registration details"
// @Success 201 {object} response.APIResponse{data=AuthResponse}
// @Failure 400 {object} response.APIResponse
// @Failure 409 {object} response.APIResponse
// @Failure 500 {object} response.APIResponse
// @Router /api/v1/auth/register [post]
func (h *AuthHandler) Register(c *gin.Context) {
	var req dto.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("Invalid registration request", "error", err)
		response.Error(c, http.StatusBadRequest, errors.ErrInvalidRequest, err.Error())
		return
	}

	// Use auth service for registration
	user, err := h.authService.Register(req.Name, req.Email, req.Password)
	if err != nil {
		if err == errors.ErrUserAlreadyExists {
			h.logger.Warn("Registration attempt with existing email", "email", req.Email)
			response.Error(c, http.StatusConflict, errors.ErrUserAlreadyExists, "User with this email already exists")
			return
		}
		h.logger.Error("Failed to create user", "email", req.Email, "error", err)
		response.Error(c, http.StatusInternalServerError, errors.ErrInternalServer, "Failed to create user account")
		return
	}

	// Generate tokens
	accessToken, refreshToken, expiresIn, err := h.generateTokens(user)
	if err != nil {
		h.logger.Error("Failed to generate tokens for new user", "user_id", user.ID, "error", err)
		response.Error(c, http.StatusInternalServerError, errors.ErrInternalServer, "Failed to generate authentication tokens")
		return
	}

	authResponse := dto.AuthResponse{
		User: dto.UserResponse{
			ID:        user.ID,
			Username:  user.Username,
			Email:     user.Email,
			CreatedAt: user.CreatedAt,
		},
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    expiresIn,
	}

	h.logger.Info("User registered successfully", "user_id", user.ID, "email", user.Email)
	response.Success(c, http.StatusCreated, "Registration successful", authResponse)
}

// RefreshToken godoc
// @Summary Refresh access token
// @Description Generate new access token using refresh token
// @Tags auth
// @Accept json
// @Produce json
// @Param request body RefreshTokenRequest true "Refresh token"
// @Success 200 {object} response.APIResponse{data=TokenResponse}
// @Failure 400 {object} response.APIResponse
// @Failure 401 {object} response.APIResponse
// @Failure 500 {object} response.APIResponse
// @Router /api/v1/auth/refresh [post]
func (h *AuthHandler) RefreshToken(c *gin.Context) {
	var req dto.RefreshTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("Invalid refresh token request", "error", err)
		response.Error(c, http.StatusBadRequest, errors.ErrInvalidRequest, err.Error())
		return
	}

	// Parse and validate refresh token
	token, err := jwt.ParseWithClaims(req.RefreshToken, &dto.RefreshClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(h.config.JWT.Secret), nil
	})

	if err != nil || !token.Valid {
		h.logger.Warn("Invalid refresh token", "error", err)
		response.Error(c, http.StatusUnauthorized, errors.ErrInvalidToken, "Invalid refresh token")
		return
	}

	claims, ok := token.Claims.(*dto.RefreshClaims)
	if !ok {
		h.logger.Error("Failed to parse refresh token claims")
		response.Error(c, http.StatusUnauthorized, errors.ErrInvalidToken, "Invalid token claims")
		return
	}

	// Check if token is expired
	if claims.ExpiresAt != nil && claims.ExpiresAt.Before(time.Now()) {
		h.logger.Warn("Refresh token expired", "user_id", claims.UserID)
		response.Error(c, http.StatusUnauthorized, errors.ErrTokenExpired, "Refresh token expired")
		return
	}

	// Verify token exists in database
	valid, err := h.authService.VerifyRefreshToken(claims.UserID, req.RefreshToken)
	if err != nil {
		h.logger.Error("Failed to verify refresh token", "user_id", claims.UserID, "error", err)
		response.Error(c, http.StatusInternalServerError, errors.ErrInternalServer, "Failed to verify refresh token")
		return
	}
	if !valid {
		h.logger.Warn("Invalid refresh token", "user_id", claims.UserID)
		response.Error(c, http.StatusUnauthorized, errors.ErrInvalidToken, "Invalid refresh token")
		return
	}

	// Get user from database
	user, err := h.authService.GetUserByID(claims.UserID)
	if err != nil {
		h.logger.Error("User not found for refresh token", "user_id", claims.UserID, "error", err)
		response.Error(c, http.StatusUnauthorized, errors.ErrUserNotFound, "User not found")
		return
	}

	// Invalidate old refresh token
	if err := h.authService.InvalidateRefreshToken(claims.UserID, req.RefreshToken); err != nil {
		h.logger.Error("Failed to invalidate old refresh token", "user_id", claims.UserID, "error", err)
		// Continue anyway as new tokens are already generated
	}

	// Generate new tokens
	accessToken, newRefreshToken, expiresIn, err := h.generateTokens(user)
	if err != nil {
		h.logger.Error("Failed to generate new tokens", "user_id", user.ID, "error", err)
		response.Error(c, http.StatusInternalServerError, errors.ErrInternalServer, "Failed to generate new tokens")
		return
	}

	tokenResponse := dto.TokenResponse{
		AccessToken:  accessToken,
		RefreshToken: newRefreshToken,
		ExpiresIn:    expiresIn,
	}

	h.logger.Info("Tokens refreshed successfully", "user_id", user.ID)
	response.Success(c, http.StatusOK, "Tokens refreshed successfully", tokenResponse)
}

// generateTokens generates both access and refresh tokens
func (h *AuthHandler) generateTokens(user *models.User) (string, string, int64, error) {
	// Access token (short-lived)
	accessTokenDuration := time.Duration(h.config.JWT.AccessTokenExpiration) * time.Millisecond
	accessClaims := &dto.Claims{
		UserID:   user.ID,
		Username: user.Email,
		Role:     "user",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(accessTokenDuration)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    h.config.App.Name,
			Subject:   user.ID,
		},
	}

	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims)
	accessTokenString, err := accessToken.SignedString([]byte(h.config.JWT.Secret))
	if err != nil {
		return "", "", 0, fmt.Errorf("failed to sign access token: %w", err)
	}

	// Refresh token (long-lived)
	refreshTokenDuration := time.Duration(h.config.JWT.RefreshTokenExpiration) * time.Millisecond
	refreshClaims := &dto.RefreshClaims{
		UserID:   user.ID,
		Username: user.Email,
		Role:     "user",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(refreshTokenDuration)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    h.config.App.Name,
			Subject:   user.ID,
		},
	}

	refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims)
	refreshTokenString, err := refreshToken.SignedString([]byte(h.config.JWT.Secret))
	if err != nil {
		return "", "", 0, fmt.Errorf("failed to sign refresh token: %w", err)
	}

	// Store refresh token in database
	if err := h.authService.StoreRefreshToken(user.ID, refreshTokenString); err != nil {
		return "", "", 0, fmt.Errorf("failed to store refresh token: %w", err)
	}

	return accessTokenString, refreshTokenString, int64(accessTokenDuration.Seconds()), nil
}

// GetCurrentUser User godoc
// @Summary Get Current authenticated User
// @Description Get current authenticated user
// @Tags auth
// @Accept json
// @Produce json
// @Param request body RegisterRequest true "Registration details"
// @Success 201 {object} response.APIResponse{data=UserResponse}
// @Failure 400 {object} response.APIResponse
// @Failure 409 {object} response.APIResponse
// @Failure 500 {object} response.APIResponse
// @Router /api/v1/auth/me [get]
func (h *AuthHandler) GetCurrentUser(c *gin.Context) {
	userIDStr, exists := c.Get("user_id")
	if !exists {
		response.Error(c, http.StatusUnauthorized, errors.ErrUnauthorized, "User not authenticated")
		return
	}

	userID, ok := userIDStr.(string)
	if !ok {
		h.logger.Error("Invalid user ID from middleware", "user_id", userIDStr)
		response.Error(c, http.StatusUnauthorized, errors.ErrUnauthorized, "Invalid user ID")
		return
	}

	user, err := h.authService.GetUserByID(userID)
	if err != nil {
		h.logger.Error("Failed to get current user", "user_id", userID, "error", err)
		response.Error(c, http.StatusNotFound, errors.ErrUserNotFound, "User not found")
		return
	}

	userResponse := dto.UserResponse{
		ID:        user.ID,
		Username:  user.Username,
		Email:     user.Email,
		CreatedAt: user.CreatedAt,
	}

	response.Success(c, http.StatusOK, "User retrieved successfully", userResponse)
}
