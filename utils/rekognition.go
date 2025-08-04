// utils/rekognition.go
package utils

import (
	"context"
	"log"
	"os"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/rekognition"
)

var rekClient *rekognition.Client

// must be called once at startup (e.g. in main.go)
func InitRekognition() {
	awsRegion := os.Getenv("AWS_REGION")
	if awsRegion == "" {
		log.Fatal("AWS_REGION not set")
	}
	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion(awsRegion),
	)
	if err != nil {
		log.Fatalf("unable to load AWS config: %v", err)
	}
	rekClient = rekognition.NewFromConfig(cfg)
}

// RekClient returns the initialized Rekognition client
func RekClient() *rekognition.Client {
	if rekClient == nil {
		InitRekognition()
	}
	return rekClient
}
