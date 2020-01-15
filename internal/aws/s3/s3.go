package s3

import (
	"io"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/wcsiu/tg-demuxer/internal/config"
)

var (
	sess       *session.Session
	uploader   *s3manager.Uploader
	bucketName string
)

// Load setup aws client
func Load() {
	// The session the S3 Uploader will use
	var c = aws.Config{
		Credentials: credentials.NewStaticCredentials(config.C.S3.AccessKeyID, config.C.S3.SecretAccessKey, ""),
		Region:      aws.String(config.C.S3.Region),
	}
	sess = session.Must(session.NewSession(&c))

	// Create an uploader with the session and default options
	uploader = s3manager.NewUploader(sess)

	// retrieve the bucket name from config
	bucketName = config.C.S3.BucketName

	// Create an uploader with the session and custom options
	// uploader := s3manager.NewUploader(session, func(u *s3manager.Uploader) {
	//      u.PartSize = 64 * 1024 * 1024 // 64MB per part
	// })
}

// Upload upload file using s3manager
func Upload(key string, body io.Reader) (*s3manager.UploadOutput, error) {
	var input = s3manager.UploadInput{
		Bucket: &bucketName,
		Key:    &key,
		Body:   body,
	}
	return uploader.Upload(&input)
}
