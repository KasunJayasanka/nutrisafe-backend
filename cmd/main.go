package main

import (
    "backend/config"
    "backend/routes"
)

func main() {
    config.InitDB()
    r := routes.SetupRouter()
    r.Run(":8080")
}
