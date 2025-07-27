package services

import (
	"context"
	"encoding/base64"
	"errors"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/rekognition"
	"github.com/aws/aws-sdk-go-v2/service/rekognition/types"
)

type RekognitionService struct {
    client *rekognition.Client
}

func NewRekognitionService() (*RekognitionService, error) {
    cfg, err := config.LoadDefaultConfig(context.TODO(),
        config.WithRegion(os.Getenv("AWS_REGION")))
    if err != nil {
        return nil, err
    }
    return &RekognitionService{client: rekognition.NewFromConfig(cfg)}, nil
}

// RecognizeLabels returns the top labels for a base64-encoded image
func (r *RekognitionService) RecognizeLabels(base64Img string) ([]string, error) {
    idx := len("data:image/jpeg;base64,")
    if idx > len(base64Img) || !strings.HasPrefix(base64Img, "data:image") {
        return nil, errors.New("invalid data URI")
    }
    data, err := base64.StdEncoding.DecodeString(base64Img[idx:])
    if err != nil {
        return nil, err
    }

    out, err := r.client.DetectLabels(context.TODO(), &rekognition.DetectLabelsInput{
        Image:         &types.Image{Bytes: data},
        MaxLabels:     aws.Int32(5),
        MinConfidence: aws.Float32(75),
    })
    if err != nil {
        return nil, err
    }

    var labels []string
    for _, l := range out.Labels {
        labels = append(labels, *l.Name)
    }
    return labels, nil
}
