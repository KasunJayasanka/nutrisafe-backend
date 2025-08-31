package services

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"time"

	"backend/models"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	awssns "github.com/aws/aws-sdk-go-v2/service/sns"
	"gorm.io/gorm"
)

type PushService struct {
	db              *gorm.DB
	sns             *awssns.Client
	fcmPlatformArn  string
	apnsPlatformArn string
}

func NewPushService(db *gorm.DB) (*PushService, error) {
	region := os.Getenv("AWS_REGION")
	if region == "" { region = "ap-south-1" }
	cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion(region))
	if err != nil { return nil, err }
	return &PushService{
		db:              db,
		sns:             awssns.NewFromConfig(cfg),
		fcmPlatformArn:  os.Getenv("SNS_FCM_ARN"),
		apnsPlatformArn: os.Getenv("SNS_APNS_ARN"),
	}, nil
}

type RegisterDeviceReq struct {
	Platform string `json:"platform"` // "android" | "ios"
	Token    string `json:"token"`
}

func (p *PushService) tokenHash(tok string) string {
	h := sha256.Sum256([]byte(tok))
	return hex.EncodeToString(h[:])
}

func (p *PushService) platformArn(platform string) (string, error) {
	switch strings.ToLower(platform) {
	case "android", "ios":
		if p.fcmPlatformArn == "" {
			return "", errors.New("SNS_FCM_ARN not set")
		}
		return p.fcmPlatformArn, nil
	default:
		return "", errors.New("unknown platform")
	}
}


func (p *PushService) RegisterDevice(userID uint, platform, token string) (*models.UserDevice, error) {
	appArn, err := p.platformArn(platform)
	if err != nil { return nil, err }

	out, err := p.sns.CreatePlatformEndpoint(context.TODO(), &awssns.CreatePlatformEndpointInput{
		PlatformApplicationArn: aws.String(appArn),
		Token:                  aws.String(token),
	})
	if err != nil { return nil, err }

	dev := &models.UserDevice{
		UserID:      userID,
		Platform:    strings.ToLower(platform),
		TokenHash:   p.tokenHash(token),
		EndpointARN: aws.ToString(out.EndpointArn),
		UpdatedAt:   time.Now(),
	}
	var existing models.UserDevice
	if err := p.db.Where("user_id=? AND token_hash=?", userID, dev.TokenHash).First(&existing).Error; err == nil {
		existing.EndpointARN = dev.EndpointARN
		existing.Platform = dev.Platform
		existing.UpdatedAt = time.Now()
		_ = p.db.Save(&existing).Error
		return &existing, nil
	}
	_ = p.db.Create(dev).Error
	return dev, nil
}

func (p *PushService) PushToUser(userID uint, title, body string, data map[string]string) {
	var endpoints []models.UserDevice
	if err := p.db.Where("user_id = ? AND enabled = ?", userID, true).Find(&endpoints).Error; err != nil {
		return
	}
	if len(endpoints) == 0 {
		return
	}

	msg := map[string]any{
		"default": body,
		"GCM": map[string]any{
			"notification": map[string]string{
				"title": title,
				"body":  body,
			},
			"data": data,
		},
	}

	raw, _ := json.Marshal(msg)
	for _, d := range endpoints {
		_, _ = p.sns.Publish(context.TODO(), &awssns.PublishInput{
			MessageStructure: aws.String("json"),
			Message:          aws.String(string(raw)),
			TargetArn:        aws.String(d.EndpointARN),
		})
	}
}
