package middleware

import (
	"strconv"
	"time"

	"vote-system/internal/obs"

	"github.com/gin-gonic/gin"
)

func Metrics() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()

		route := c.FullPath()
		obs.ObserveHTTP(c.Request.Method, route, strconv.Itoa(c.Writer.Status()), time.Since(start))
	}
}
