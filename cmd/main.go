package main

import (
	"backend/config"
	"backend/routes"
	"backend/utils"
	"log"
	"os"

	"github.com/joho/godotenv"
)

func main() {

    if err := godotenv.Load(); err != nil {
        log.Fatalf("error loading .env: %v", err)
    }
    // Debug: print out what we actually loaded
    log.Printf("EDAMAM_NUTRI_APP_ID='%s'", os.Getenv("EDAMAM_NUTRI_APP_ID"))
    log.Printf("EDAMAM_NUTRI_APP_KEY='%s'", os.Getenv("EDAMAM_NUTRI_APP_KEY"))

    config.InitDB()
	utils.InitS3() // ✅ initialize the S3 client
    utils.InitRekognition()
    r := routes.SetupRouter()
    r.Run(":8080")
}
