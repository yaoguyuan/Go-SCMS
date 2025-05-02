package utils

import (
	"auth/models"
	"bytes"
	"encoding/json"
	"io"
	"strings"

	"github.com/gin-gonic/gin"
)

// SubInfo is a struct that holds information about the subject of the request.
// It includes the user's IP address, ID, and role.
type SubInfo struct {
	IP   string `json:"ip"`
	ID   uint   `json:"id"`
	Role string `json:"role"`
}

// GetUnAuthInfo returns a SubInfo struct from the context.
func GetSubInfo(c *gin.Context) *SubInfo {
	user, _ := c.Get("user")
	return &SubInfo{
		IP:   c.ClientIP(),
		ID:   user.(models.User).ID,
		Role: user.(models.User).Role,
	}
}

// GetRawBody returns the raw body string from the context.
func GetRawBody(c *gin.Context) string {
	bodyBytes, _ := io.ReadAll(c.Request.Body)
	// Reset the request body so it can be read in the handler
	c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
	return string(bodyBytes)
}

// GetParsedBody returns the parsed body from the raw body string.
// If the string cannot be parsed to JSON, it returns nil.
func GetParsedBody(rawBody string) map[string]interface{} {
	rawBodyBytes := []byte(rawBody)
	var parsedBody map[string]interface{}
	if err := json.Unmarshal(rawBodyBytes, &parsedBody); err != nil {
		return nil
	}
	return parsedBody
}

// GetRawQuery returns the raw query string from the context.
func GetRawQuery(c *gin.Context) string {
	return c.Request.URL.RawQuery
}

// GetParsedQuery returns the parsed query from the context.
func GetParsedQuery(c *gin.Context) map[string]interface{} {
	parsedQuery := make(map[string]interface{})
	for key, values := range c.Request.URL.Query() {
		parsedQuery[key] = strings.Join(values, ",")
	}
	return parsedQuery
}

// ObjInfo is a struct that holds information about the object of the request.
type ObjInfo struct {
	Op    string `json:"op"`
	Table string `json:"table"`
	ID    uint   `json:"id"`
}

// ObjInfo.Op is the operation type of the request.
// It can be one of the following values:
const (
	OpCreate = "CREATE"
	OpRead   = "READ"
	OpUpdate = "UPDATE"
	OpDelete = "DELETE"
)

// DataInfo is a struct that holds information about the data of the request.
type DataInfo struct {
	OldData map[string]interface{} `json:"old_data"`
	NewData map[string]interface{} `json:"new_data"`
}
