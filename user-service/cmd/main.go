package main

import (
	"github.com/gin-gonic/gin"
	"github.com/pokonti/psychologist-backend/user-service/config"
	"github.com/pokonti/psychologist-backend/user-service/internal/handlers"
	"github.com/pokonti/psychologist-backend/user-service/internal/models"
	"github.com/pokonti/psychologist-backend/user-service/internal/repository"
	"github.com/pokonti/psychologist-backend/user-service/routes"
)

func main() {
	config.ConnectDB()
	db := config.DB
	config.DB.AutoMigrate(&models.User{})

	repo := repository.NewPostgresProfileRepository(db)
	handler := handlers.NewProfileHandler(repo)

	r := gin.Default()
	routes.SetupRoutes(r, handler)
	r.Run(":8081")
}
