package main

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"time"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/cloudfront"
	cftypes "github.com/aws/aws-sdk-go-v2/service/cloudfront/types"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
)

//go:embed index.html
var indexTmpl []byte

type Event struct {
	Prompt string `json:"prompt"`
	Model  string `json:"model"`
	Seed   string `json:"seed,omitempty"`
}

func HandleRequest(ctx context.Context, evt Event) error {
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return err
	}

	tmpl, err := template.New("index").Parse(string(indexTmpl))
	if err != nil {
		return err
	}

	key := os.Getenv("DEZGO_KEY")
	if key == "" {
		ssmc := ssm.NewFromConfig(cfg)
		out, err := ssmc.GetParameter(ctx, &ssm.GetParameterInput{
			Name:           aws.String(os.Getenv("DEZGO_KEY_PARAM")),
			WithDecryption: aws.Bool(true),
		})
		if err != nil {
			return err
		}
		key = aws.ToString(out.Parameter.Value)
	}

	body, err := json.Marshal(evt)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.dezgo.com/text2image", bytes.NewBuffer(body))
	if err != nil {
		return err
	}

	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("X-Dezgo-Key", key)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	seed := resp.Header.Get("x-input-seed")
	name := time.Now().UTC().Format("20060102")
	bucket := os.Getenv("BUCKET")

	kitten := map[string]string{
		"Name":   name,
		"Prompt": evt.Prompt,
		"Model":  evt.Model,
		"Seed":   seed,
	}

	s3c := s3.NewFromConfig(cfg)
	uploader := manager.NewUploader(s3c)
	if _, err := uploader.Upload(ctx, &s3.PutObjectInput{
		Bucket:       aws.String(bucket),
		Key:          aws.String(name + ".png"),
		ContentType:  aws.String("image/png"),
		Body:         resp.Body,
		Metadata:     kitten,
		StorageClass: s3types.StorageClassIntelligentTiering,
	}); err != nil {
		return err
	}

	var data bytes.Buffer
	if err := tmpl.Execute(&data, kitten); err != nil {
		return err
	}

	if _, err := uploader.Upload(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(bucket),
		Key:         aws.String("index.html"),
		ContentType: aws.String("text/html"),
		Body:        bytes.NewReader(data.Bytes()),
		Metadata:    kitten,
	}); err != nil {
		return err
	}

	if _, err := s3c.CopyObject(ctx, &s3.CopyObjectInput{
		Bucket:       aws.String(bucket),
		Key:          aws.String(name + ".html"),
		ContentType:  aws.String("text/html"),
		CopySource:   aws.String(bucket + "/index.html"),
		Metadata:     kitten,
		StorageClass: s3types.StorageClassIntelligentTiering,
	}); err != nil {
		return err
	}

	cf := cloudfront.NewFromConfig(cfg)
	if _, err := cf.CreateInvalidation(ctx, &cloudfront.CreateInvalidationInput{
		DistributionId: aws.String(os.Getenv("DISTRIBUTION")),
		InvalidationBatch: &cftypes.InvalidationBatch{
			CallerReference: aws.String(time.Now().UTC().Format("20060102150405")),
			Paths: &cftypes.Paths{
				Quantity: aws.Int32(2),
				Items: []string{
					"/index.html",
					fmt.Sprintf("/%s.png", name),
				},
			},
		},
	}); err != nil {
		return err
	}

	return nil
}

func main() {
	if _, ok := os.LookupEnv("AWS_LAMBDA_RUNTIME_API"); ok {
		lambda.Start(HandleRequest)
	} else {
		if err := HandleRequest(context.Background(), Event{
			Prompt: os.Args[1],
			Model:  os.Args[2],
		}); err != nil {
			panic(err)
		}
	}
}
