package main

import (
	"backend/config"
	"backend/routes"
	"backend/utils"
	"log"

	"github.com/joho/godotenv"
)

func main() {

    if err := godotenv.Load(); err != nil {
        log.Fatalf("error loading .env: %v", err)
    }


    config.InitDB()
	utils.InitS3() // ✅ initialize the S3 client
    utils.InitRekognition()
    r := routes.SetupRouter(config.DB)
    r.Run(":8080")
}
