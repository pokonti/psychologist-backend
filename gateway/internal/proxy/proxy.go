package proxy

import (
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"

	"github.com/gin-gonic/gin"
)

func Forward(target string) gin.HandlerFunc {
	targetURL, err := url.Parse(target)
	if err != nil {
		log.Fatalf("invalid proxy target %q: %v", target, err)
	}

	// creating reverse proxy
	proxy := httputil.NewSingleHostReverseProxy(targetURL)

	// director, to see host
	originalDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		originalDirector(req)
		req.Host = targetURL.Host
	}

	return func(c *gin.Context) {
		c.Request = c.Request.WithContext(c)
		proxy.ServeHTTP(c.Writer, c.Request)
	}
}
