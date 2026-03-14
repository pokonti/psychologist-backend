package handlers

import (
	"context"
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
	if err := config.DB.Where("is_booked = ?", true).Find(&slots).Error; err != nil {
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
		Where("id = ? AND is_booked = ?", slotID, true).
		Updates(map[string]interface{}{
			"is_booked":             false,
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
	config.DB.Model(&models.Slot{}).Where("is_booked = ?", true).Count(&stats.TotalBookings)
	config.DB.Model(&models.Slot{}).Where("is_booked = ?", true).Count(&total)

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
		Where("is_booked = ?", true).
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
