package routes

import (
	"backend/controllers"
	"backend/middlewares"

	"github.com/gin-gonic/gin"
)

func SetupRouter() *gin.Engine {
	r := gin.Default()

	// Public auth routes
	auth := r.Group("/auth")
	{
		auth.POST("/register", controllers.Register)
		auth.POST("/login", controllers.Login)
		auth.POST("/verify-mfa", controllers.VerifyMFA)
		auth.POST("/forgot-password", controllers.ForgotPassword)
		auth.POST("/reset-password", controllers.ResetPassword)

	}

	// Protected user routes
	user := r.Group("/user")
	user.Use(middlewares.AuthMiddleware())
	{
		user.GET("/profile", controllers.GetProfile)
		user.PATCH("/profile", controllers.UpdateProfile)
		user.PATCH("/mfa", controllers.ToggleMFA)
		user.PATCH("/onboarding", controllers.OnboardUser)

        user.POST("/meals", controllers.LogMeal)
        user.GET("/meals", controllers.ListMeals)
        user.GET("/recommendations", controllers.GetRecommendations)

	}

	r.GET("/food/search", controllers.SearchFoods)
	r.POST("/food/recognize", controllers.RecognizeFood)

	r.POST("/dev/upload-image", controllers.DevUploadImage)

	return r
}
