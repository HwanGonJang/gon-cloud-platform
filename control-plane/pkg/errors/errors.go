package errors

import (
	"errors"
)

// Authentication errors
var (
	ErrInvalidCredentials   = errors.New("invalid email or password")
	ErrUserAlreadyExists    = errors.New("user already exists")
	ErrUserNotFound         = errors.New("user not found")
	ErrInvalidToken         = errors.New("invalid token")
	ErrTokenExpired         = errors.New("token expired")
	ErrUnauthorized         = errors.New("unauthorized")
	ErrForbidden            = errors.New("forbidden")
	ErrRefreshTokenNotFound = errors.New("refresh token not found")
)

// Password errors
var (
	ErrWeakPassword     = errors.New("password is too weak")
	ErrPasswordTooLong  = errors.New("password is too long")
	ErrPasswordMismatch = errors.New("passwords do not match")
)

// Request errors
var (
	ErrInvalidRequest   = errors.New("invalid request")
	ErrInvalidJSON      = errors.New("invalid JSON format")
	ErrMissingParameter = errors.New("missing required parameter")
	ErrInvalidParameter = errors.New("invalid parameter")
)

// Server errors
var (
	ErrInternalServer = errors.New("internal server error")
	ErrDatabaseError  = errors.New("database error")
	ErrServiceError   = errors.New("service error")
	ErrNotImplemented = errors.New("feature not implemented")
)

// Resource errors
var (
	ErrResourceNotFound    = errors.New("resource not found")
	ErrResourceExists      = errors.New("resource already exists")
	ErrResourceInUse       = errors.New("resource is in use")
	ErrResourceUnavailable = errors.New("resource unavailable")
)

// VPC and Network errors
var (
	ErrVPCNotFound           = errors.New("VPC not found")
	ErrVPCAlreadyExists      = errors.New("VPC already exists")
	ErrInvalidCIDR           = errors.New("invalid CIDR block")
	ErrCIDRConflict          = errors.New("CIDR block conflicts with existing VPC")
	ErrSubnetNotFound        = errors.New("subnet not found")
	ErrSubnetAlreadyExists   = errors.New("subnet already exists")
	ErrSubnetCIDROutOfRange  = errors.New("subnet CIDR is out of VPC range")
	ErrSecurityGroupNotFound = errors.New("security group not found")
	ErrInvalidSecurityRule   = errors.New("invalid security group rule")
)

// Instance errors
var (
	ErrInstanceNotFound      = errors.New("instance not found")
	ErrInstanceAlreadyExists = errors.New("instance already exists")
	ErrInstanceNotRunning    = errors.New("instance is not running")
	ErrInstanceNotStopped    = errors.New("instance is not stopped")
	ErrInvalidInstanceType   = errors.New("invalid instance type")
	ErrImageNotFound         = errors.New("image not found")
	ErrInsufficientResources = errors.New("insufficient resources")
)

// Validation errors
var (
	ErrValidationFailed = errors.New("validation failed")
	ErrInvalidEmail     = errors.New("invalid email format")
	ErrInvalidName      = errors.New("invalid name format")
	ErrInvalidID        = errors.New("invalid ID format")
)

// Error types for better error handling
type ErrorType string

const (
	ErrorTypeValidation     ErrorType = "validation"
	ErrorTypeAuthentication ErrorType = "authentication"
	ErrorTypeAuthorization  ErrorType = "authorization"
	ErrorTypeNotFound       ErrorType = "not_found"
	ErrorTypeConflict       ErrorType = "conflict"
	ErrorTypeInternal       ErrorType = "internal"
	ErrorTypeExternal       ErrorType = "external"
)

// AppError represents an application error with additional context
type AppError struct {
	Type    ErrorType `json:"type"`
	Code    string    `json:"code"`
	Message string    `json:"message"`
	Details string    `json:"details,omitempty"`
	Err     error     `json:"-"`
}

func (e *AppError) Error() string {
	if e.Err != nil {
		return e.Message + ": " + e.Err.Error()
	}
	return e.Message
}

func (e *AppError) Unwrap() error {
	return e.Err
}

// New creates a new AppError
func New(errType ErrorType, code, message string) *AppError {
	return &AppError{
		Type:    errType,
		Code:    code,
		Message: message,
	}
}

// Wrap wraps an existing error with additional context
func Wrap(err error, errType ErrorType, code, message string) *AppError {
	return &AppError{
		Type:    errType,
		Code:    code,
		Message: message,
		Err:     err,
	}
}

// Predefined error constructors
func ValidationError(message string) *AppError {
	return New(ErrorTypeValidation, "VALIDATION_ERROR", message)
}

func AuthenticationError(message string) *AppError {
	return New(ErrorTypeAuthentication, "AUTHENTICATION_ERROR", message)
}

func AuthorizationError(message string) *AppError {
	return New(ErrorTypeAuthorization, "AUTHORIZATION_ERROR", message)
}

func NotFoundError(message string) *AppError {
	return New(ErrorTypeNotFound, "NOT_FOUND", message)
}

func ConflictError(message string) *AppError {
	return New(ErrorTypeConflict, "CONFLICT", message)
}

func InternalError(message string) *AppError {
	return New(ErrorTypeInternal, "INTERNAL_ERROR", message)
}
