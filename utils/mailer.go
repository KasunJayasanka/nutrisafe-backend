package utils

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ses"
	"github.com/aws/aws-sdk-go-v2/service/ses/types"
)

var sesClient *ses.Client

func init() {
	cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion(os.Getenv("AWS_REGION")))
	if err != nil {
		log.Fatalf("AWS config load failed: %v", err)
	}
	sesClient = ses.NewFromConfig(cfg)
}

// generic SES sender
func sendEmail(to string, subject string, body string) error {
	input := &ses.SendEmailInput{
		Destination: &types.Destination{
			ToAddresses: []string{to},
		},
		Message: &types.Message{
			Subject: &types.Content{
				Data: aws.String(subject),
			},
			Body: &types.Body{
				Text: &types.Content{
					Data: aws.String(body),
				},
			},
		},
		Source: aws.String(os.Getenv("SES_EMAIL")),
	}

	_, err := sesClient.SendEmail(context.TODO(), input)
	if err != nil {
		log.Printf("SES send error: %v", err)
		return fmt.Errorf("email send failed: %v", err)
	}
	return nil
}

// MFA-specific email sender
func SendMFAEmail(to string, code string) error {
	subject := "Your MFA Code"
	body := fmt.Sprintf("Your MFA verification code is: %s\n\nUse this to complete your login.", code)
	return sendEmail(to, subject, body)
}

// Forgot Password email sender
func SendResetEmail(to string, token string) error {
	subject := "Password Reset Code"
	body := fmt.Sprintf("Your password reset code is: %s\n\nUse this in the app to set a new password.", token)
	return sendEmail(to, subject, body)
}
