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
    }

    // Protected user routes
    user := r.Group("/user")
    user.Use(middlewares.AuthMiddleware())
    {
        user.GET("/profile", controllers.GetProfile)
        user.PUT("/profile", controllers.UpdateProfile)
    }

    return r
}
