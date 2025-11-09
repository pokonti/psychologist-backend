package proxy

import (
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
)

func Forward(target string) gin.HandlerFunc {
	return func(c *gin.Context) {
		req, err := http.NewRequest(c.Request.Method, target+c.Request.RequestURI, c.Request.Body)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create request"})
			return
		}

		req.Header = c.Request.Header

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			c.JSON(http.StatusBadGateway, gin.H{"error": "Service unavailable"})
			return
		}
		defer resp.Body.Close()

		body, _ := io.ReadAll(resp.Body)
		c.Data(resp.StatusCode, resp.Header.Get("Content-Type"), body)
	}
}
