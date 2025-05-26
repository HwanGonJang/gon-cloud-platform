package services

import (
	"crypto/sha256"
	"fmt"
	"time"

	"gon-cloud-platform/control-plane/internal/database/repositories"
	"gon-cloud-platform/control-plane/internal/models"
	"gon-cloud-platform/control-plane/internal/utils"
	"gon-cloud-platform/control-plane/pkg/errors"

	"github.com/google/uuid"
)

type AuthService interface {
	Register(name, email, password string) (*models.User, error)
	Login(email, password string) (*models.User, error)
	GetUserByID(id string) (*models.User, error)
	GetUserByEmail(email string) (*models.User, error)
	UpdateUser(user *models.User) error
	ValidatePassword(password string) error
	HashPassword(password string) string
	VerifyPassword(password, hash string) bool
	StoreRefreshToken(userID string, token string) error
	VerifyRefreshToken(userID string, token string) (bool, error)
	InvalidateRefreshToken(userID string, token string) error
}

type authService struct {
	userRepo repositories.UserRepository
	config   *utils.Config
	logger   *utils.Logger
}

func NewAuthService(userRepo repositories.UserRepository, config *utils.Config, logger *utils.Logger) AuthService {
	return &authService{
		userRepo: userRepo,
		config:   config,
		logger:   logger,
	}
}

func (s *authService) Register(name, email, password string) (*models.User, error) {
	// Validate password
	if err := s.ValidatePassword(password); err != nil {
		return nil, err
	}

	// Check if user already exists
	existingUser, err := s.userRepo.GetByEmail(email)
	if err == nil && existingUser != nil {
		s.logger.Warn("Registration attempt with existing email", "email", email)
		return nil, errors.ErrUserAlreadyExists
	}

	// Generate unique username from email if name is not provided
	username := name
	if username == "" {
		username = generateUsernameFromEmail(email)
	}

	// Ensure username is unique
	username, err = s.ensureUniqueUsername(username)
	if err != nil {
		return nil, fmt.Errorf("failed to generate unique username: %w", err)
	}

	// Hash password
	passwordHash := s.HashPassword(password)

	// Create user
	user := &models.User{
		ID:           uuid.New().String(),
		Username:     username,
		Email:        email,
		Password:     passwordHash,
		Role:         "user", // Default role
		IsActive:     true,
		RefreshToken: "",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	createdUser, err := s.userRepo.Create(user)
	if err != nil {
		s.logger.Error("Failed to create user", "email", email, "error", err)
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	s.logger.Info("User registered successfully", "user_id", createdUser.ID, "email", createdUser.Email)
	return createdUser, nil
}

func (s *authService) Login(email, password string) (*models.User, error) {
	// Find user by email
	user, err := s.userRepo.GetByEmail(email)
	if err != nil {
		s.logger.Warn("Login attempt with non-existent email", "email", email)
		return nil, errors.ErrInvalidCredentials
	}

	// Verify password
	if !s.VerifyPassword(password, user.Password) {
		s.logger.Warn("Invalid password attempt", "email", email, "user_id", user.ID)
		return nil, errors.ErrInvalidCredentials
	}

	// Update last login
	if err := s.userRepo.UpdateLastLogin(user.ID); err != nil {
		s.logger.Warn("Failed to update last login", "user_id", user.ID, "error", err)
	}

	s.logger.Info("User logged in successfully", "user_id", user.ID, "email", user.Email)
	return user, nil
}

func (s *authService) GetUserByID(id string) (*models.User, error) {
	user, err := s.userRepo.GetByID(id)
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (s *authService) GetUserByEmail(email string) (*models.User, error) {
	user, err := s.userRepo.GetByEmail(email)
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (s *authService) UpdateUser(user *models.User) error {
	user.UpdatedAt = time.Now()
	return s.userRepo.Update(user)
}

func (s *authService) ValidatePassword(password string) error {
	if len(password) < 6 {
		return errors.ErrWeakPassword
	}
	if len(password) > 100 {
		return errors.ErrPasswordTooLong
	}
	return nil
}

func (s *authService) HashPassword(password string) string {
	// Use SHA-256 with salt
	salt := s.config.App.Salt
	if salt == "" {
		salt = "default-salt" // Should be configured properly
	}
	hash := sha256.Sum256([]byte(password + salt))
	return fmt.Sprintf("%x", hash)
}

func (s *authService) VerifyPassword(password, hash string) bool {
	passwordHash := s.HashPassword(password)
	return passwordHash == hash
}

// ensureUniqueUsername generates a unique username by appending numbers if needed
func (s *authService) ensureUniqueUsername(baseUsername string) (string, error) {
	username := baseUsername
	counter := 1

	for {
		// Check if username exists
		_, err := s.userRepo.GetByUsername(username)
		if err != nil {
			if err == errors.ErrUserNotFound {
				// Username is available
				return username, nil
			}
			// Some other error occurred
			return "", err
		}

		// Username exists, try with counter
		username = fmt.Sprintf("%s%d", baseUsername, counter)
		counter++

		// Prevent infinite loop
		if counter > 9999 {
			return "", fmt.Errorf("unable to generate unique username after 9999 attempts")
		}
	}
}

// generateUsernameFromEmail creates a username from email address
func generateUsernameFromEmail(email string) string {
	// Extract the part before @ and clean it
	if atIndex := findAtIndex(email); atIndex != -1 {
		username := email[:atIndex]
		return cleanUsername(username)
	}
	return "user"
}

// findAtIndex finds the index of @ in email
func findAtIndex(email string) int {
	for i, char := range email {
		if char == '@' {
			return i
		}
	}
	return -1
}

// cleanUsername removes special characters and makes it suitable for username
func cleanUsername(input string) string {
	var result []rune
	for _, char := range input {
		if (char >= 'a' && char <= 'z') ||
			(char >= 'A' && char <= 'Z') ||
			(char >= '0' && char <= '9') ||
			char == '_' || char == '-' {
			result = append(result, char)
		}
	}

	username := string(result)
	if len(username) == 0 {
		return "user"
	}
	if len(username) > 30 {
		return username[:30]
	}
	return username
}

func (s *authService) StoreRefreshToken(userID string, token string) error {
	return s.userRepo.StoreRefreshToken(userID, token)
}

func (s *authService) VerifyRefreshToken(userID string, token string) (bool, error) {
	return s.userRepo.VerifyRefreshToken(userID, token)
}

func (s *authService) InvalidateRefreshToken(userID string, token string) error {
	return s.userRepo.InvalidateRefreshToken(userID, token)
}
