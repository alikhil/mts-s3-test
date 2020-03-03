package storage

import (
	"bytes"
	"context"
	"errors"
	"io"
	"io/ioutil"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
)

// ObjectID - is a shortcut for s3.ObjectIdentifier
type ObjectID = *s3.ObjectIdentifier

var NoSuchKeyError = errors.New("no such key")

const imageContentType = "image/jpeg"

// S3Context - context to work with s3 methods
type S3Context struct {
	session *session.Session
	config  *Config
}

// InitS3Context - creates and inits session for s3
func InitS3Context(cfg *Config) (*S3Context, error) {
	var ctx = &S3Context{
		config: cfg,
	}
	var err error
	ctx.session, err = session.NewSession(&aws.Config{
		Endpoint: aws.String(cfg.S3Endpoint),
		Region:   aws.String(cfg.S3Region),
		Credentials: credentials.NewStaticCredentials(
			cfg.S3AccessKeyID,
			cfg.S3SecretAccessKey,
			"",
		),

		// LogLevel: aws.LogLevel(aws.LogDebugWithHTTPBody),
	})
	return ctx, err
}

func (ctx *S3Context) UploadFile(file io.Reader, objectKey, contentType string) (string, error) {
	manager := s3manager.NewUploader(ctx.session)
	out, err := manager.Upload(&s3manager.UploadInput{
		Bucket:      aws.String(ctx.config.S3Bucket),
		Body:        file,
		Key:         aws.String(objectKey),
		ACL:         aws.String("public-read"),
		ContentType: aws.String(contentType),
	})

	if err != nil {
		return "", err
	}

	return out.Location, err
}

// UploadFile - uploads the file with objectKey key
func (ctx *S3Context) UploadImageFile(file io.Reader, objectKey string) (string, error) {
	// We assume that image is already converted to jpg
	return ctx.UploadFile(file, objectKey, imageContentType)

}

// UploadFileWithContext - uploads the file with objectKey key with context
func (ctx *S3Context) UploadFileWithContext(cctx context.Context, file io.Reader, objectKey, contentType string) (string, error) {

	manager := s3manager.NewUploader(ctx.session)
	out, err := manager.UploadWithContext(cctx, &s3manager.UploadInput{
		Bucket:      aws.String(ctx.config.S3Bucket),
		Body:        file,
		Key:         aws.String(objectKey),
		ACL:         aws.String("public-read"),
		ContentType: aws.String(contentType),
	})

	if err != nil {
		return "", err
	}

	return out.Location, err
}

// UploadImageFileWithContext - uploads the file with objectKey key with context
func (ctx *S3Context) UploadImageFileWithContext(cctx context.Context, file io.Reader, objectKey string) (string, error) {

	return ctx.UploadFileWithContext(cctx, file, objectKey, imageContentType)
}

// CopyObject - make a copy of object
func (ctx *S3Context) CopyObject(source, dest string) error {

	var service = s3.New(ctx.session)
	var _, err = service.CopyObject(&s3.CopyObjectInput{
		CopySource: aws.String(ctx.config.S3Bucket + "/" + source),
		Key:        aws.String(dest),
		ACL:        aws.String("public-read"),
		Bucket:     aws.String(ctx.config.S3Bucket),
	})

	return err
}

// GetObject - returns s3 object content
func (ctx *S3Context) GetObject(objectKey string) ([]byte, error) {
	var service = s3.New(ctx.session)
	object, err := service.GetObject(&s3.GetObjectInput{
		Bucket: aws.String(ctx.config.S3Bucket),
		Key:    aws.String(objectKey),
	})

	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case s3.ErrCodeNoSuchKey:
				return nil, NoSuchKeyError
			}
		}
		return nil, err
	}

	return ioutil.ReadAll(object.Body)
}

// ListFiles - list all objects with prefix
func (ctx *S3Context) ListFiles(prefix string) ([]ObjectID, error) {

	svc := s3.New(ctx.session)

	objects, err := svc.ListObjects(&s3.ListObjectsInput{
		Bucket: aws.String(ctx.config.S3Bucket),
		Prefix: aws.String(prefix),
	})

	if err != nil {
		return nil, err
	}

	obIdentifiers := make([]ObjectID, len(objects.Contents))
	for i, obj := range objects.Contents {
		obIdentifiers[i] = &s3.ObjectIdentifier{Key: obj.Key}
	}

	return obIdentifiers, nil
}

// DeleteFiles - deletes objects from s3
func (ctx *S3Context) DeleteFiles(obIdentifiers []ObjectID) error {
	svc := s3.New(ctx.session)

	svc.Handlers.Build.PushBack(func(r *request.Request) {

		if r.Operation.Name == "DeleteObjects" {
			buf := new(bytes.Buffer)
			var _, err = buf.ReadFrom(r.Body)
			if err == nil {
				updated := bytes.Replace(buf.Bytes(), []byte(` xmlns="http://s3.amazonaws.com/doc/2006-03-01/"`), []byte(""), -1)
				r.SetReaderBody(bytes.NewReader(updated))
			}
		}
	})

	var _, err = svc.DeleteObjects(&s3.DeleteObjectsInput{
		Bucket: aws.String(ctx.config.S3Bucket),
		Delete: &s3.Delete{
			Objects: obIdentifiers,
		},
	})
	return err
}

// DeleteFolder - Deletes all files with given prefix
func (ctx *S3Context) DeleteFolder(prefix string) error {
	var files, err = ctx.ListFiles(prefix)
	if err != nil {
		return err
	}

	return ctx.DeleteFiles(files)
}
