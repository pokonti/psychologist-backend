package middleware

import "github.com/gin-gonic/gin"

func CorsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		allowedOrigins := []string{
			"http://localhost:5173",
			"https://kbtucare.site",
			"https://www.kbtucare.site",
		}

		origin := c.Request.Header.Get("Origin")

		for _, b := range allowedOrigins {
			if b == origin {
				c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
				break
			}
		}

		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE, PATCH")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}
