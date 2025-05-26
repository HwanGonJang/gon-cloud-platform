package response

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// APIResponse represents a standardized API response
type APIResponse struct {
	Success   bool        `json:"success"`
	Message   string      `json:"message"`
	Data      interface{} `json:"data,omitempty"`
	Error     *ErrorInfo  `json:"error,omitempty"`
	Meta      *Meta       `json:"meta,omitempty"`
	Timestamp time.Time   `json:"timestamp"`
}

// ErrorInfo provides detailed error information
type ErrorInfo struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

// Meta provides additional metadata for responses
type Meta struct {
	Page       int   `json:"page,omitempty"`
	Limit      int   `json:"limit,omitempty"`
	Total      int64 `json:"total,omitempty"`
	TotalPages int   `json:"total_pages,omitempty"`
}

// PaginationMeta creates meta information for paginated responses
type PaginationMeta struct {
	Page       int   `json:"page"`
	Limit      int   `json:"limit"`
	Total      int64 `json:"total"`
	TotalPages int   `json:"total_pages"`
}

// Success sends a successful response
func Success(c *gin.Context, statusCode int, message string, data interface{}) {
	response := APIResponse{
		Success:   true,
		Message:   message,
		Data:      data,
		Timestamp: time.Now(),
	}
	c.JSON(statusCode, response)
}

// SuccessWithMeta sends a successful response with metadata
func SuccessWithMeta(c *gin.Context, statusCode int, message string, data interface{}, meta *Meta) {
	response := APIResponse{
		Success:   true,
		Message:   message,
		Data:      data,
		Meta:      meta,
		Timestamp: time.Now(),
	}
	c.JSON(statusCode, response)
}

// Error sends an error response
func Error(c *gin.Context, statusCode int, err error, details string) {
	errorInfo := &ErrorInfo{
		Code:    getErrorCode(err),
		Message: err.Error(),
		Details: details,
	}

	response := APIResponse{
		Success:   false,
		Message:   "Request failed",
		Error:     errorInfo,
		Timestamp: time.Now(),
	}
	c.JSON(statusCode, response)
}

// ErrorWithMessage sends an error response with custom message
func ErrorWithMessage(c *gin.Context, statusCode int, message string, err error, details string) {
	errorInfo := &ErrorInfo{
		Code:    getErrorCode(err),
		Message: err.Error(),
		Details: details,
	}

	response := APIResponse{
		Success:   false,
		Message:   message,
		Error:     errorInfo,
		Timestamp: time.Now(),
	}
	c.JSON(statusCode, response)
}

// Created sends a 201 Created response
func Created(c *gin.Context, message string, data interface{}) {
	Success(c, http.StatusCreated, message, data)
}

// OK sends a 200 OK response
func OK(c *gin.Context, message string, data interface{}) {
	Success(c, http.StatusOK, message, data)
}

// NoContent sends a 204 No Content response
func NoContent(c *gin.Context, message string) {
	Success(c, http.StatusNoContent, message, nil)
}

// BadRequest sends a 400 Bad Request response
func BadRequest(c *gin.Context, err error, details string) {
	Error(c, http.StatusBadRequest, err, details)
}

// Unauthorized sends a 401 Unauthorized response
func Unauthorized(c *gin.Context, err error, details string) {
	Error(c, http.StatusUnauthorized, err, details)
}

// Forbidden sends a 403 Forbidden response
func Forbidden(c *gin.Context, err error, details string) {
	Error(c, http.StatusForbidden, err, details)
}

// NotFound sends a 404 Not Found response
func NotFound(c *gin.Context, err error, details string) {
	Error(c, http.StatusNotFound, err, details)
}

// Conflict sends a 409 Conflict response
func Conflict(c *gin.Context, err error, details string) {
	Error(c, http.StatusConflict, err, details)
}

// InternalServerError sends a 500 Internal Server Error response
func InternalServerError(c *gin.Context, err error, details string) {
	Error(c, http.StatusInternalServerError, err, details)
}

// Paginated sends a paginated response
func Paginated(c *gin.Context, message string, data interface{}, page, limit int, total int64) {
	totalPages := int((total + int64(limit) - 1) / int64(limit))

	meta := &Meta{
		Page:       page,
		Limit:      limit,
		Total:      total,
		TotalPages: totalPages,
	}

	SuccessWithMeta(c, http.StatusOK, message, data, meta)
}

// getErrorCode extracts error code from error
func getErrorCode(err error) string {
	if err == nil {
		return "UNKNOWN_ERROR"
	}

	// You can implement custom error types here
	// For now, we'll use a simple string representation
	return fmt.Sprintf("ERR_%d", hashString(err.Error()))
}

// hashString creates a simple hash for error codes
func hashString(s string) int {
	hash := 0
	for _, char := range s {
		hash = hash*31 + int(char)
	}
	if hash < 0 {
		hash = -hash
	}
	return hash % 10000
}

// Additional utility functions

// ValidationError sends a validation error response
func ValidationError(c *gin.Context, err error, details string) {
	errorInfo := &ErrorInfo{
		Code:    "VALIDATION_ERROR",
		Message: err.Error(),
		Details: details,
	}

	response := APIResponse{
		Success:   false,
		Message:   "Validation failed",
		Error:     errorInfo,
		Timestamp: time.Now(),
	}
	c.JSON(http.StatusBadRequest, response)
}

// ServiceUnavailable sends a 503 Service Unavailable response
func ServiceUnavailable(c *gin.Context, err error, details string) {
	Error(c, http.StatusServiceUnavailable, err, details)
}

// TooManyRequests sends a 429 Too Many Requests response
func TooManyRequests(c *gin.Context, err error, details string) {
	Error(c, http.StatusTooManyRequests, err, details)
}
