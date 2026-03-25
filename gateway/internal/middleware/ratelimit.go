package middleware

import (
	"log"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/ulule/limiter/v3"
	mgin "github.com/ulule/limiter/v3/drivers/middleware/gin"
	sredis "github.com/ulule/limiter/v3/drivers/store/redis"
)

// SetupRateLimiter initializes Redis and returns the Gin middleware
func SetupRateLimiter() gin.HandlerFunc {
	// limit 100 requests per 1 minute
	rate, err := limiter.NewRateFromFormatted("100-M")
	if err != nil {
		log.Fatalf("Failed to parse rate limit: %v", err)
	}

	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		redisURL = "redis:6379"
	}

	client := redis.NewClient(&redis.Options{
		Addr: redisURL,
	})

	store, err := sredis.NewStoreWithOptions(client, limiter.StoreOptions{
		Prefix:   "limiter_gateway",
		MaxRetry: 3,
	})
	if err != nil {
		log.Fatalf("Failed to create Redis store: %v", err)
	}

	// Create a new limiter instance
	instance := limiter.New(store, rate)

	// Return the Gin middleware
	// This will automatically return 429 Too Many Requests if the limit is exceeded
	return mgin.NewMiddleware(instance)
}

// SetupAuthLimiter is a stricter limit just for Login/Register to prevent brute force
func SetupAuthLimiter() gin.HandlerFunc {
	// Only 5 requests per minute for auth routes
	rate, _ := limiter.NewRateFromFormatted("5-M")

	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		redisURL = "redis:6379"
	}

	client := redis.NewClient(&redis.Options{Addr: redisURL})
	store, _ := sredis.NewStoreWithOptions(client, limiter.StoreOptions{Prefix: "limiter_auth"})

	return mgin.NewMiddleware(limiter.New(store, rate))
}
