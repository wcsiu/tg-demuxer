package s3

import (
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
)

var (
	sess     *session.Session
	uploader *s3manager.Uploader
)

// Load setup aws client
func Load() {
	// The session the S3 Uploader will use
	sess = session.Must(session.NewSession())

	// Create an uploader with the session and default options
	uploader = s3manager.NewUploader(sess)

	// Create an uploader with the session and custom options
	// uploader := s3manager.NewUploader(session, func(u *s3manager.Uploader) {
	//      u.PartSize = 64 * 1024 * 1024 // 64MB per part
	// })
}
