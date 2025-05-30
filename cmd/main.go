package main

import (
    "backend/config"
    "backend/routes"
	"backend/utils"
)

func main() {
    config.InitDB()
	utils.InitS3() // ✅ initialize the S3 client
    r := routes.SetupRouter()
    r.Run(":8080")
}
