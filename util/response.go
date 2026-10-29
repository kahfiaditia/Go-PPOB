package util

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type APIResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

func OK(c *gin.Context, message string, data interface{}) {
	c.JSON(http.StatusOK, APIResponse{Success: true, Message: message, Data: data})
}

func Created(c *gin.Context, message string, data interface{}) {
	c.JSON(http.StatusCreated, APIResponse{Success: true, Message: message, Data: data})
}

func Accepted(c *gin.Context, message string, data interface{}) {
	c.JSON(http.StatusAccepted, APIResponse{Success: true, Message: message, Data: data})
}

func BadRequest(c *gin.Context, err string) {
	c.JSON(http.StatusBadRequest, APIResponse{Success: false, Error: err})
}

func Unauthorized(c *gin.Context, err string) {
	c.JSON(http.StatusUnauthorized, APIResponse{Success: false, Error: err})
}

func Forbidden(c *gin.Context, err string) {
	c.JSON(http.StatusForbidden, APIResponse{Success: false, Error: err})
}

func NotFound(c *gin.Context, err string) {
	c.JSON(http.StatusNotFound, APIResponse{Success: false, Error: err})
}

func InternalError(c *gin.Context, err string) {
	c.JSON(http.StatusInternalServerError, APIResponse{Success: false, Error: err})
}

func TooManyRequests(c *gin.Context, err string) {
	c.JSON(http.StatusTooManyRequests, APIResponse{Success: false, Error: err})
}
