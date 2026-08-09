package api

import "github.com/gin-gonic/gin"

// Response 统一响应结构
type Response struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// PageResult 分页结果
type PageResult struct {
	List     interface{} `json:"list"`
	Total    int64       `json:"total"`
	Page     int         `json:"page"`
	PageSize int         `json:"page_size"`
}

func OK(c *gin.Context, data interface{}) {
	c.JSON(200, Response{
		Code:    0,
		Message: "success",
		Data:    data,
	})
}

func OKMsg(c *gin.Context, msg string) {
	c.JSON(200, Response{
		Code:    0,
		Message: msg,
	})
}

func OKPage(c *gin.Context, list interface{}, total int64, page, pageSize int) {
	c.JSON(200, Response{
		Code: 0,
		Message: "success",
		Data: PageResult{
			List:     list,
			Total:    total,
			Page:     page,
			PageSize: pageSize,
		},
	})
}

func Fail(c *gin.Context, code int, msg string) {
	c.JSON(200, Response{
		Code:    code,
		Message: msg,
	})
}

func paramInt(c *gin.Context, key string, def int) int {
	v := c.Query(key)
	if v == "" {
		return def
	}
	n := 0
	for _, c := range v {
		if c < '0' || c > '9' {
			return def
		}
		n = n*10 + int(c-'0')
	}
	return n
}
