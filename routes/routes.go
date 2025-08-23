package routes

import (
	"backend/controllers"
	"backend/middlewares"
	"backend/services"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func SetupRouter(db *gorm.DB) *gin.Engine {
	r := gin.Default()
	
	rtHub := services.NewRealtimeHub()
	pushSvc, _ := services.NewPushService(db)
	services.InitAlertDeps(db, rtHub, pushSvc)

	rtCtl := controllers.NewRealtimeController(rtHub)
	devCtl := controllers.NewDeviceController(pushSvc)
	developmentCtl := controllers.NewDevController(pushSvc)


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
		analyticsSvc := services.NewAnalyticsService(db)
		analyticsCtl := controllers.NewAnalyticsController(analyticsSvc)

		user.GET("/profile", controllers.GetProfile)
		user.PATCH("/profile", controllers.UpdateProfile)
		user.PATCH("/mfa", controllers.ToggleMFA)
		user.PATCH("/onboarding", controllers.OnboardUser)

		user.POST("/meals", controllers.LogMeal)
		user.GET("/meals", controllers.ListMeals)
		user.PATCH("/meals/:id", controllers.UpdateMeal)
		user.DELETE("/meals/:id", controllers.DeleteMeal)
		user.GET("/recommendations", controllers.GetRecommendations)
		user.GET("/meals/:id", controllers.GetMealByID)
		user.GET("/meals/recent", controllers.ListRecentMeals)
		user.GET("/meal-items/recent", controllers.ListRecentMealItems)

		user.GET("/meals/warnings", controllers.ListMealWarnings)
		user.GET("/meals/:id/warnings", controllers.GetMealWarnings)

		user.GET("/goals", controllers.GetGoals)
		user.PATCH("/goals", controllers.UpdateGoals)

		user.PATCH("/daily-activity", controllers.UpdateDailyActivity)
		user.GET("/daily-progress", controllers.GetDailyProgressHistory)
		user.GET("/goals-by-date", controllers.GetGoalsByDate)
		user.GET("/nutrient-breakdown-by-date", controllers.GetNutrientBreakdownByDate)

		user.GET("/analytics/summary", analyticsCtl.GetAnalyticsSummary)
		user.GET("/analytics/weekly-overview", analyticsCtl.GetWeeklyOverview)

		user.POST("/devices/register", devCtl.Register)
		user.POST("/notifications/toggle", controllers.ToggleNotifications)
		user.POST("/dev/push", developmentCtl.PushTest)

	}

	ws := r.Group("/ws")
	ws.Use(middlewares.AuthMiddleware())
	{
		ws.GET("/alerts", rtCtl.AlertsWS)
	}

	r.GET("/food/search", controllers.SearchFoods)
	r.POST("/food/recognize", controllers.RecognizeFood)

	r.POST("/dev/upload-image", controllers.DevUploadImage)

	return r
}
