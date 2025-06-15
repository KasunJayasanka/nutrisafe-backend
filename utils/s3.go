package utils

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"mime"
	"os"
	"strings"
	"time"
	"log"

	"github.com/aws/aws-sdk-go-v2/config"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
)

var s3Client *s3.Client

func InitS3() {
	s3Region := os.Getenv("S3_REGION")
	if s3Region == "" {
		s3Region = os.Getenv("AWS_REGION") // fallback
	}

	cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion(s3Region))
	if err != nil {
		log.Fatalf("Unable to load AWS config for S3: %v", err)
	}

	s3Client = s3.NewFromConfig(cfg)
}


func UploadBase64ImageToS3(base64Data, filenamePrefix string) (string, error) {
	// Split base64 data ("data:<mime>;base64,<data>")
	parts := strings.Split(base64Data, ",")
	if len(parts) != 2 {
		return "", fmt.Errorf("invalid base64 image")
	}
	meta := parts[0]
	data := parts[1]

	// Detect content type
	mediaType := strings.SplitN(meta, ":", 2)[1]       // "image/jpeg;base64"
	contentType := strings.SplitN(mediaType, ";", 2)[0] // "image/jpeg"

	// Determine extension
	exts, _ := mime.ExtensionsByType(contentType)
	var ext string
	switch contentType {
	case "image/jpeg", "image/jpg":
		ext = ".jpg"
	default:
		if len(exts) > 0 {
			ext = exts[0]
		} else {
			// fallback: use subtype
			parts := strings.SplitN(contentType, "/", 2)
			if len(parts) == 2 {
				ext = "." + parts[1]
			}
		}
	}

	// Decode the image bytes
	imageData, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		return "", fmt.Errorf("failed to decode image: %v", err)
	}

	// Build a unique S3 key
	key := fmt.Sprintf("profile-pictures/%s-%d%s",
		filenamePrefix,
		time.Now().UnixNano(),
		ext,
	)

	// Upload to S3
	_, err = s3Client.PutObject(context.TODO(), &s3.PutObjectInput{
		Bucket:      aws.String(os.Getenv("S3_BUCKET")),
		Key:         aws.String(key),
		Body:        bytes.NewReader(imageData),
		ContentType: aws.String(contentType),
		ACL:         s3types.ObjectCannedACLPublicRead,
	})
	if err != nil {
		return "", fmt.Errorf("failed to upload to S3: %v", err)
	}

	// Return the public URL via CloudFront
	cfURL := os.Getenv("CLOUDFRONT_URL")
	return fmt.Sprintf("%s/%s", cfURL, key), nil
}