package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/pokonti/psychologist-backend/booking-service/config"
	"github.com/pokonti/psychologist-backend/booking-service/internal/models"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// Setup in-memory DB for testing
func setupTestDB() {
	db, _ := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	config.DB = db
	config.DB.AutoMigrate(&models.Slot{})
}

func TestReserveSlot_Concurrency(t *testing.T) {
	setupTestDB()
	gin.SetMode(gin.TestMode)

	// 1. Create one Available Slot in the DB
	slotID := "test-uuid-123"
	config.DB.Create(&models.Slot{
		ID:      slotID,
		Status:  models.StatusAvailable,
		Version: 1, // Initial version
	})

	h := &BookingHandler{}
	r := gin.Default()
	r.POST("/reserve/:id", h.ReserveSlot)

	// 2. We will fire 2 requests at once
	const concurrentRequests = 2
	var wg sync.WaitGroup
	wg.Add(concurrentRequests)

	// A channel to collect the status codes from both requests
	statusCodes := make(chan int, concurrentRequests)

	// 3. Launch goroutines simultaneously
	for i := 0; i < concurrentRequests; i++ {
		go func(studentNum int) {
			defer wg.Done()

			// Prepare request
			body, _ := json.Marshal(models.ReserveSlotInput{BookingType: "online"})
			req, _ := http.NewRequest("POST", "/reserve/"+slotID, bytes.NewBuffer(body))
			req.Header.Set("X-User-ID", "student-id") // Identity doesn't matter for the lock test

			// Execute request
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			statusCodes <- w.Code
		}(i)
	}

	wg.Wait()
	close(statusCodes)

	// 4. ANALYZE RESULTS
	results := []int{}
	for code := range statusCodes {
		results = append(results, code)
	}

	// We expect ONE success (200) and ONE conflict (409)
	has200 := false
	has409 := false

	for _, code := range results {
		if code == http.StatusOK {
			has200 = true
		}
		if code == http.StatusConflict {
			has409 = true
		}
	}

	// ASSERTIONS
	assert.True(t, has200, "One request should have succeeded")
	assert.True(t, has409, "One request should have failed with 409 Conflict")

	// 5. Verify the DB state
	var finalSlot models.Slot
	config.DB.First(&finalSlot, "id = ?", slotID)
	assert.Equal(t, models.StatusReserved, finalSlot.Status)
	assert.Equal(t, 2, finalSlot.Version, "Version should have incremented exactly once")
}
