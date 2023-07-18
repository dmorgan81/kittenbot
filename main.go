package main

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strings"
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

//go:embed latest.html
var latestTmpl string

type Event struct {
	Model  string `json:"model,omitempty"`
	Prompt string `json:"prompt,omitempty"`
	Seed   string `json:"seed,omitempty"`
}

func (e Event) String() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("model: %s | prompt: %s", e.Model, e.Prompt))
	if e.Seed != "" {
		sb.WriteString(fmt.Sprintf(" | seed: %s", e.Seed))
	}
	return sb.String()
}

func HandleRequest(ctx context.Context, evt Event) error {
	log := log.Default()
	log.SetPrefix("KITTENBOT - ")
	log.Println("HandleRequest called")
	defer func() {
		log.Println("HandleRequest finished")
	}()

	rand := rand.New(rand.NewSource(time.Now().Unix()))

	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return err
	}

	tmpl, err := template.New("latest").Parse(latestTmpl)
	if err != nil {
		return err
	}

	ssmc := ssm.NewFromConfig(cfg)

	key := os.Getenv("DEZGO_KEY")
	if key == "" {
		log.Println("Fetching Dezgo API key from Parameter Store")
		out, err := ssmc.GetParameter(ctx, &ssm.GetParameterInput{
			Name:           aws.String(os.Getenv("DEZGO_KEY_PARAM")),
			WithDecryption: aws.Bool(true),
		})
		if err != nil {
			return err
		}
		key = aws.ToString(out.Parameter.Value)
	}

	if evt.Model == "" || evt.Prompt == "" {
		log.Println("Fetching model/prompt from Parameter Store")
		out, err := ssmc.GetParametersByPath(ctx, &ssm.GetParametersByPathInput{
			Path:           aws.String(os.Getenv("PROMPTS_PARAM")),
			WithDecryption: aws.Bool(true),
		})
		if err != nil {
			return err
		}
		pair := strings.Split(aws.ToString(out.Parameters[rand.Intn(len(out.Parameters))].Value), "|")
		if evt.Model == "" {
			evt.Model = pair[0]
		}
		if evt.Prompt == "" {
			evt.Prompt = pair[1]
		}
	}
	log.Print("Event: %s\n", evt)

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

	log.Println("Generating image via Dezgo")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	seed := resp.Header.Get("x-input-seed")
	log.Printf("Image seed: %s\n", seed)

	now := time.Now().UTC().Format("20060102")
	bucket := os.Getenv("BUCKET")

	kitten := map[string]string{
		"Image":  now + ".png",
		"Prompt": evt.Prompt,
		"Model":  evt.Model,
		"Seed":   seed,
	}

	s3c := s3.NewFromConfig(cfg)
	uploader := manager.NewUploader(s3c)

	log.Printf("Uploading %s to %s\n", now+".png", bucket)
	if _, err := uploader.Upload(ctx, &s3.PutObjectInput{
		Bucket:       aws.String(bucket),
		Key:          aws.String(now + ".png"),
		ContentType:  aws.String("image/png"),
		Body:         resp.Body,
		Metadata:     kitten,
		StorageClass: s3types.StorageClassIntelligentTiering,
	}); err != nil {
		return err
	}

	log.Printf("Copying %s to latest.png\n", now+".png")
	if _, err := s3c.CopyObject(ctx, &s3.CopyObjectInput{
		Bucket:      aws.String(bucket),
		Key:         aws.String("latest.png"),
		ContentType: aws.String("image/png"),
		CopySource:  aws.String(fmt.Sprintf("%s/%s.png", bucket, now)),
		Metadata:    kitten,
	}); err != nil {
		return err
	}

	var data bytes.Buffer
	if err := tmpl.Execute(&data, kitten); err != nil {
		return err
	}

	log.Printf("Uploading %s to %s\n", now+".html", bucket)
	if _, err := uploader.Upload(ctx, &s3.PutObjectInput{
		Bucket:       aws.String(bucket),
		Key:          aws.String(now + ".html"),
		ContentType:  aws.String("text/html"),
		Body:         bytes.NewReader(data.Bytes()),
		Metadata:     kitten,
		StorageClass: s3types.StorageClassIntelligentTiering,
	}); err != nil {
		return err
	}

	log.Printf("Copying %s to latest.png\n", now+".html")
	if _, err := s3c.CopyObject(ctx, &s3.CopyObjectInput{
		Bucket:      aws.String(bucket),
		Key:         aws.String("latest.html"),
		ContentType: aws.String("text/html"),
		CopySource:  aws.String(fmt.Sprintf("%s/%s.html", bucket, now)),
		Metadata:    kitten,
	}); err != nil {
		return err
	}

	distribution := os.Getenv("DISTRIBUTION")
	log.Printf("Invaliding created files in CloudFront distribution %s\n", distribution)
	cf := cloudfront.NewFromConfig(cfg)
	if _, err := cf.CreateInvalidation(ctx, &cloudfront.CreateInvalidationInput{
		DistributionId: aws.String(distribution),
		InvalidationBatch: &cftypes.InvalidationBatch{
			CallerReference: aws.String(time.Now().UTC().Format("20060102150405")),
			Paths: &cftypes.Paths{
				Quantity: aws.Int32(4),
				Items: []string{
					"/latest.html",
					"/latest.png",
					fmt.Sprintf("/%s.png", now),
					fmt.Sprintf("/%s.html", now),
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
