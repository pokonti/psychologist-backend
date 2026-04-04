package clients

import (
	"log"
	"os"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

var S3Client *s3.S3
var BucketName string
var Endpoint string

func InitS3() {
	key := os.Getenv("DO_SPACES_KEY")
	secret := os.Getenv("DO_SPACES_SECRET")
	Endpoint = os.Getenv("DO_SPACES_ENDPOINT")
	BucketName = os.Getenv("DO_SPACES_BUCKET")

	if key == "" || secret == "" {
		log.Println("DO Spaces credentials missing. Image uploads will fail.")
		return
	}

	s3Config := &aws.Config{
		Credentials:      credentials.NewStaticCredentials(key, secret, ""),
		Endpoint:         aws.String(Endpoint),
		Region:           aws.String("us-east-1"),
		S3ForcePathStyle: aws.Bool(false),
	}

	newSession, err := session.NewSession(s3Config)
	if err != nil {
		log.Fatalf("Failed to create S3 session: %v", err)
	}

	S3Client = s3.New(newSession)
	log.Println("Connected to DigitalOcean Spaces")
}

// GeneratePresignedURL creates a temporary URL valid for 15 minutes
func GeneratePresignedURL(objectKey string, contentType string) (string, error) {
	req, _ := S3Client.PutObjectRequest(&s3.PutObjectInput{
		Bucket:      aws.String(BucketName),
		Key:         aws.String(objectKey),
		ContentType: aws.String(contentType),
	})

	// URL expires in 15 minutes
	urlStr, err := req.Presign(15 * time.Minute)
	if err != nil {
		return "", err
	}

	return urlStr, nil
}
