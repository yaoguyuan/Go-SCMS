package utils

import (
	"math"
	"strconv"

	"github.com/gin-gonic/gin"
)

// PaginationParams is a struct that holds pagination parameters.
type PaginationParams struct {
	PageNum  int
	PageSize int
	Offset   int
}

// PaginationResult is a struct that holds pagination result.
type PaginationResult struct {
	CurrentPage  int   `json:"current_page"`
	TotalPages   int   `json:"total_pages"`
	CurrentCount int   `json:"current_count"`
	TotalCount   int64 `json:"total_count"`
	HasNext      bool  `json:"has_next"`
	HasPrev      bool  `json:"has_prev"`
}

// GetPaginationParams returns a PaginationParams struct from the Context.
func GetPaginationParams(c *gin.Context) *PaginationParams {
	// Get pageNum and pageSize off the query string
	pageNum, err := strconv.Atoi(c.DefaultQuery("pageNum", "1"))
	if err != nil || pageNum < 1 {
		panic("Invalid pageNum: Must be a positive integer")
	}
	pageSize, err := strconv.Atoi(c.DefaultQuery("pageSize", "10"))
	if err != nil || pageSize < 1 {
		panic("Invalid pageSize: Must be a positive integer")
	}

	// Calculate offset
	offset := (pageNum - 1) * pageSize

	return &PaginationParams{PageNum: pageNum, PageSize: pageSize, Offset: offset}
}

// GetPaginationResult returns a PaginationResult struct.
func GetPaginationResult(params *PaginationParams, data_count int, total_count int64) *PaginationResult {
	// Calculate the total number of pages
	totalPages := int(math.Ceil(float64(total_count) / float64(params.PageSize)))
	if params.PageNum > totalPages {
		panic("Invalid pageNum: Exceeds total number of pages")
	}

	return &PaginationResult{
		CurrentPage:  params.PageNum,
		TotalPages:   totalPages,
		CurrentCount: data_count,
		TotalCount:   total_count,
		HasNext:      params.PageNum < totalPages,
		HasPrev:      params.PageNum > 1,
	}
}
