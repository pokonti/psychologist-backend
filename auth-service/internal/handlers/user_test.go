package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/pokonti/psychologist-backend/auth-service/config"
	"github.com/pokonti/psychologist-backend/auth-service/internal/models"
	"github.com/pokonti/psychologist-backend/auth-service/internal/utils"
	"github.com/pokonti/psychologist-backend/proto/userprofile"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// MockUserClient mocks the gRPC client
type MockUserClient struct {
	mock.Mock
}

func (m *MockUserClient) CreateUserProfile(ctx context.Context, in *userprofile.CreateUserProfileRequest, opts ...grpc.CallOption) (*userprofile.CreateUserProfileResponse, error) {
	args := m.Called(ctx, in)
	// Return nil if args.Get(0) is nil, otherwise cast it
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*userprofile.CreateUserProfileResponse), args.Error(1)
}

func (m *MockUserClient) GetUserProfileByID(ctx context.Context, in *userprofile.GetUserProfileByIDRequest, opts ...grpc.CallOption) (*userprofile.GetUserProfileByIDResponse, error) {
	return nil, nil
}

func setupTestDB() {
	// Using in-memory SQLite instead of Postgres
	db, _ := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	config.DB = db
	config.DB.AutoMigrate(&models.User{})
}

func setupRouter(ac *AuthController) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.Default()
	r.POST("/register", ac.Register)
	r.POST("/verify", ac.VerifyEmail)
	r.POST("/login", ac.Login)
	return r
}

func TestRegisterSuccess(t *testing.T) {
	setupTestDB()

	originalEmailFunc := utils.SendVerificationEmail
	defer func() { utils.SendVerificationEmail = originalEmailFunc }()

	utils.SendVerificationEmail = func(toEmail, code string) error {
		return nil
	}

	mockUserClient := new(MockUserClient)

	mockUserClient.On("CreateUserProfile", mock.Anything, mock.Anything).
		Return(&userprofile.CreateUserProfileResponse{Id: "123"}, nil)

	ac := &AuthController{
		UserClient: mockUserClient,
	}
	r := setupRouter(ac)

	input := RegisterInput{
		Email:    "newuser@test.com",
		Password: "password123",
		Role:     "student",
	}
	jsonBytes, _ := json.Marshal(input)
	req, _ := http.NewRequest("POST", "/register", bytes.NewBuffer(jsonBytes))
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	assert.Contains(t, w.Body.String(), "Registration successful")

	var user models.User
	config.DB.Where("email = ?", "newuser@test.com").First(&user)
	assert.NotEmpty(t, user.ID)
	assert.False(t, user.IsVerified) // Should be false initially
}

func TestVerifySuccess(t *testing.T) {
	setupTestDB()

	// Pre-seed DB with a user pending verification
	user := models.User{
		ID:               "user-123",
		Email:            "verify@test.com",
		Role:             "student",
		VerificationCode: "111111",
		CodeExpiresAt:    time.Now().Add(10 * time.Minute),
		IsVerified:       false,
	}
	config.DB.Create(&user)

	ac := &AuthController{}
	r := setupRouter(ac)

	input := VerifyInput{
		Email: "verify@test.com",
		Code:  "111111",
	}
	jsonBytes, _ := json.Marshal(input)
	req, _ := http.NewRequest("POST", "/verify", bytes.NewBuffer(jsonBytes))
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var updatedUser models.User
	config.DB.First(&updatedUser, "id = ?", "user-123")
	assert.True(t, updatedUser.IsVerified)
	assert.Empty(t, updatedUser.VerificationCode)
}

func TestLoginFailNotVerified(t *testing.T) {
	setupTestDB()

	// Pre-seed Unverified User
	hash, _ := utils.HashPassword("password123")
	user := models.User{
		ID:         "user-456",
		Email:      "login@test.com",
		Password:   hash,
		IsVerified: false, // Not verified
	}
	config.DB.Create(&user)

	ac := &AuthController{}
	r := setupRouter(ac)

	input := LoginInput{
		Email:    "login@test.com",
		Password: "password123",
	}
	jsonBytes, _ := json.Marshal(input)
	req, _ := http.NewRequest("POST", "/login", bytes.NewBuffer(jsonBytes))
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code) // Should be 403
	assert.Contains(t, w.Body.String(), "verify your email")
}
func TestLoginSuccess(t *testing.T) {
	setupTestDB()

	// Seed Verified User
	hash, _ := utils.HashPassword("password123")
	user := models.User{
		ID:         "user-success",
		Email:      "valid@test.com",
		Password:   hash,
		IsVerified: true,
	}
	config.DB.Create(&user)

	ac := &AuthController{}
	r := setupRouter(ac)

	input := LoginInput{
		Email:    "valid@test.com",
		Password: "password123",
	}
	jsonBytes, _ := json.Marshal(input)
	req, _ := http.NewRequest("POST", "/login", bytes.NewBuffer(jsonBytes))
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "token") // Should return a JWT
}
func TestRegisterFailDuplicate(t *testing.T) {
	setupTestDB()

	// Seed existing user
	config.DB.Create(&models.User{
		Email:      "duplicate@test.com",
		IsVerified: true,
	})

	ac := &AuthController{} // No mocks needed, it fails before calling them
	r := setupRouter(ac)

	input := RegisterInput{
		Email:    "duplicate@test.com",
		Password: "password123",
		Role:     "student",
	}
	jsonBytes, _ := json.Marshal(input)
	req, _ := http.NewRequest("POST", "/register", bytes.NewBuffer(jsonBytes))
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusConflict, w.Code) // 409
}
func TestVerifyFailWrongCode(t *testing.T) {
	setupTestDB()

	// Seed user
	config.DB.Create(&models.User{
		ID:               "user-wrong-code",
		Email:            "wrongcode@test.com",
		VerificationCode: "123456",
		CodeExpiresAt:    time.Now().Add(10 * time.Minute),
	})

	ac := &AuthController{}
	r := setupRouter(ac)

	input := VerifyInput{
		Email: "wrongcode@test.com",
		Code:  "999999", // WRONG CODE
	}
	jsonBytes, _ := json.Marshal(input)
	req, _ := http.NewRequest("POST", "/verify", bytes.NewBuffer(jsonBytes))
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code) // 401
}
func TestVerifyFailExpired(t *testing.T) {
	setupTestDB()

	// Seed user with expired time
	config.DB.Create(&models.User{
		ID:               "user-expired",
		Email:            "expired@test.com",
		VerificationCode: "123456",
		CodeExpiresAt:    time.Now().Add(-10 * time.Minute),
	})

	ac := &AuthController{}
	r := setupRouter(ac)

	input := VerifyInput{
		Email: "expired@test.com",
		Code:  "123456",
	}
	jsonBytes, _ := json.Marshal(input)
	req, _ := http.NewRequest("POST", "/verify", bytes.NewBuffer(jsonBytes))
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code) // 400
}
