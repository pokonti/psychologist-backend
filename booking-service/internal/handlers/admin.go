package handlers

import (
	"context"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/pokonti/psychologist-backend/booking-service/config"
	"github.com/pokonti/psychologist-backend/booking-service/internal/models"
	"github.com/pokonti/psychologist-backend/proto/userprofile"
)

// GetAllBookings godoc
// @Summary      Admin: View all bookings
// @Description  Allows admin to see all booked slots across the platform.
// @Tags         admin
// @Produce      json
// @Security     BearerAuth
// @Success      200 {array} models.SlotResponse
// @Failure      401 {object} models.ErrorResponse
// @Router       /admin/bookings [get]
func (h *BookingHandler) GetAllBookings(c *gin.Context) {
	role := c.GetHeader("X-User-Role")
	if role != "admin" {
		c.JSON(http.StatusForbidden, models.ErrorResponse{Error: "Admin access required"})
		return
	}

	var slots []models.Slot
	if err := config.DB.Where("status = ?", models.StatusBooked).Find(&slots).Error; err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "Database error"})
		return
	}

	// Note: You could enrich with names here too if needed
	c.JSON(http.StatusOK, slots)
}

// ForceCancelBooking godoc
// @Summary      Admin: Force cancel a booking
// @Description  Allows admin to cancel any booking, even if they aren't the psychologist or student.
// @Tags         admin
// @Produce      json
// @Security     BearerAuth
// @Success      200 {object} models.MessageResponse
// @Failure      401 {object} models.ErrorResponse
// @Param        id path string true "Slot ID"
// @Router       /admin/bookings/{id}/cancel [post]
func (h *BookingHandler) ForceCancelBooking(c *gin.Context) {
	role := c.GetHeader("X-User-Role")
	if role != "admin" {
		c.JSON(http.StatusForbidden, models.ErrorResponse{Error: "Admin access required"})
		return
	}

	slotID := c.Param("id")

	// Logic: Same as CancelAppointment, but skip the "is owner" check
	result := config.DB.Model(&models.Slot{}).
		Where("id = ? AND status = ?", slotID, models.StatusBooked).
		Updates(map[string]interface{}{
			"status":                models.StatusAvailable,
			"student_id":            nil,
			"booking_type":          "",
			"questionnaire_answers": "",
		})

	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, models.ErrorResponse{Error: "Booking not found or already canceled"})
		return
	}

	c.JSON(http.StatusOK, models.MessageResponse{Message: "Booking force-canceled by admin"})
}

type AdminDashboard struct {
	TotalBookings        int64   `json:"total_bookings"`
	TotalWaitlisted      int64   `json:"total_waitlisted"`
	OnlineRatio          float64 `json:"online_ratio"`
	OfflineRatio         float64 `json:"offline_ratio"`
	MostPopularPsychID   string  `json:"most_popular_psych_id"`
	MostPopularPsychName string  `json:"most_popular_psych_name"`
}

// GetDashboard godoc
// @Summary      Admin: Get system statistics
// @Description  Returns global stats for the admin dashboard, including top psychologist.
// @Tags         admin
// @Produce      json
// @Security     BearerAuth
// @Success      200 {object} AdminDashboard
// @Failure      401 {object} models.ErrorResponse
// @Router       /admin/dashboard [get]
func (h *BookingHandler) GetDashboard(c *gin.Context) {
	if c.GetHeader("X-User-Role") != "admin" {
		c.JSON(http.StatusForbidden, models.ErrorResponse{Error: "Admin only"})
		return
	}

	var stats AdminDashboard
	var total int64

	// Booked stats
	config.DB.Model(&models.Slot{}).Where("status = ?", models.StatusBooked).Count(&stats.TotalBookings)
	config.DB.Model(&models.Slot{}).Where("status = ?", models.StatusBooked).Count(&total)

	var onlineCount int64
	config.DB.Model(&models.Slot{}).Where("booking_type = ?", "online").Count(&onlineCount)

	if total > 0 {
		stats.OnlineRatio = float64(onlineCount) / float64(total)
		stats.OfflineRatio = 1.0 - stats.OnlineRatio
	}

	config.DB.Model(&models.WaitlistEntry{}).Count(&stats.TotalWaitlisted)

	type Result struct {
		PsychologistID string
		Count          int
	}
	var res Result
	err := config.DB.Model(&models.Slot{}).
		Select("psychologist_id, count(*) as count").
		Where("status = ?", models.StatusBooked).
		Group("psychologist_id").
		Order("count desc").
		Limit(1).
		Scan(&res).Error

	stats.MostPopularPsychName = "N/A"
	stats.MostPopularPsychID = res.PsychologistID

	if err == nil && res.PsychologistID != "" {
		grpcResp, gErr := h.UserClient.GetBatchUserProfiles(context.Background(), &userprofile.GetBatchUserProfilesRequest{
			Ids: []string{res.PsychologistID},
		})

		if gErr == nil && len(grpcResp.Profiles) > 0 {
			stats.MostPopularPsychName = grpcResp.Profiles[0].FullName
		}
	}

	c.JSON(http.StatusOK, stats)
}

// GetAllReviews godoc
// @Summary      Admin: View all reviews
// @Description  Admin views unmasked ratings and reviews across the platform, including student and psychologist identities.
// @Tags         admin
// @Produce      json
// @Security     BearerAuth
// @Success      200 {array} models.AdminReviewResponse
// @Failure      403 {object} models.ErrorResponse "Admin access required"
// @Failure      500 {object} models.ErrorResponse "Database error"
// @Router       /admin/reviews [get]
func (h *BookingHandler) GetAllReviews(c *gin.Context) {
	role := c.GetHeader("X-User-Role")

	if role != "admin" {
		c.JSON(http.StatusForbidden, models.ErrorResponse{Error: "Admin access required"})
		return
	}

	var slots []models.Slot
	if err := config.DB.
		Where("rating > 0").
		Order("start_time desc").
		Find(&slots).Error; err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "Database error"})
		return
	}

	if len(slots) == 0 {
		c.JSON(http.StatusOK, []models.AdminReviewResponse{})
		return
	}

	var userIDs []string
	uniqueUsers := make(map[string]bool)

	for _, s := range slots {
		if !uniqueUsers[s.PsychologistID] {
			uniqueUsers[s.PsychologistID] = true
			userIDs = append(userIDs, s.PsychologistID)
		}
		if s.StudentID != nil && !uniqueUsers[*s.StudentID] {
			uniqueUsers[*s.StudentID] = true
			userIDs = append(userIDs, *s.StudentID)
		}
	}

	userMap := make(map[string]string)
	grpcResp, err := h.UserClient.GetBatchUserProfiles(c.Request.Context(), &userprofile.GetBatchUserProfilesRequest{
		Ids: userIDs,
	})

	if err == nil {
		for _, p := range grpcResp.Profiles {
			userMap[p.Id] = p.FullName
		}
	} else {
		log.Printf("Failed to fetch user profiles for admin reviews: %v", err)
	}

	var response []models.AdminReviewResponse
	for _, s := range slots {
		psychName := "Unknown Psychologist"
		if name, ok := userMap[s.PsychologistID]; ok {
			psychName = name
		}

		studentName := "Unknown Student"
		studentID := ""
		if s.StudentID != nil {
			studentID = *s.StudentID
			if name, ok := userMap[*s.StudentID]; ok {
				studentName = name
			}
		}

		response = append(response, models.AdminReviewResponse{
			SlotID:           s.ID,
			PsychologistID:   s.PsychologistID,
			PsychologistName: psychName,
			StudentID:        studentID,
			StudentName:      studentName,
			StartTime:        s.StartTime,
			Rating:           s.Rating,
			Review:           s.Review,
		})
	}

	c.JSON(http.StatusOK, response)
}

// AdminGetUserSessions godoc
// @Summary      Admin: View all sessions for a specific user
// @Description  Returns a history of all slots (available, reserved, booked) associated with a specific user ID.
// @Tags         admin
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      string  true  "User ID (UUID)"
// @Success      200  {array}   models.AdminUserSessionResponse
// @Failure      403  {object}  models.ErrorResponse "Admin access required"
// @Router       /admin/users/{id}/sessions [get]
func (h *BookingHandler) AdminGetUserSessions(c *gin.Context) {
	if c.GetHeader("X-User-Role") != "admin" {
		c.JSON(http.StatusForbidden, models.ErrorResponse{Error: "Admin access required"})
		return
	}

	targetID := c.Param("id")

	// 1. Find slots where this user is the Psychologist OR the Student
	var slots []models.Slot
	if err := config.DB.
		Where("psychologist_id = ? OR student_id = ?", targetID, targetID).
		Order("start_time desc").
		Find(&slots).Error; err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "Database error"})
		return
	}

	if len(slots) == 0 {
		c.JSON(http.StatusOK, []models.AdminUserSessionResponse{})
		return
	}

	// 2. Collect IDs for gRPC Name Enrichment
	var userIDs []string
	uniqueIDs := make(map[string]bool)
	for _, s := range slots {
		if !uniqueIDs[s.PsychologistID] {
			uniqueIDs[s.PsychologistID] = true
			userIDs = append(userIDs, s.PsychologistID)
		}
		if s.StudentID != nil && !uniqueIDs[*s.StudentID] {
			uniqueIDs[*s.StudentID] = true
			userIDs = append(userIDs, *s.StudentID)
		}
	}

	// 3. Fetch names from User Service
	userMap := make(map[string]string)
	resp, err := h.UserClient.GetBatchUserProfiles(c.Request.Context(), &userprofile.GetBatchUserProfilesRequest{Ids: userIDs})
	if err == nil {
		for _, p := range resp.Profiles {
			userMap[p.Id] = p.FullName
		}
	}

	// 4. Build Response
	var response []models.AdminUserSessionResponse
	for _, s := range slots {
		res := models.AdminUserSessionResponse{
			ID:               s.ID,
			StartTime:        s.StartTime,
			Status:           s.Status,
			BookingType:      s.BookingType,
			PsychologistID:   s.PsychologistID,
			PsychologistName: userMap[s.PsychologistID],
			Rating:           s.Rating,
		}
		if s.StudentID != nil {
			res.StudentID = *s.StudentID
			res.StudentName = userMap[*s.StudentID]
		}
		response = append(response, res)
	}

	c.JSON(http.StatusOK, response)
}
