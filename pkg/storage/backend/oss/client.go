package oss

import (
	"bytes"
	"context"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	api2 "github.com/pjoc-team/fsync/pkg/storage/api"
	"github.com/pjoc-team/fsync/pkg/util/fs"
	"github.com/pjoc-team/tracing/logger"
	"io"
)

type storage struct {
	blockSize      int
	bucket         string
	newSessionFunc func() (*session.Session, error)
	sess           *session.Session
}

func NewOssStorage(
	conf *Conf,
	blockSize int, debug bool,
) (api2.FileStorage, error) {
	s := &storage{}
	s.newSessionFunc = func() (*session.Session, error) {
		creds := credentials.NewStaticCredentials(conf.SecretID, conf.SecretKey, "")
		region := "Auto"
		config := &aws.Config{
			Region:           aws.String(region),
			Endpoint:         aws.String(conf.Endpoint),
			S3ForcePathStyle: aws.Bool(true),
			Credentials:      creds,
			// LogLevel:         aws.LogLevel(aws.LogDebug),
			// DisableSSL:       &disableSSL,
		}
		if debug {
			config.LogLevel = aws.LogLevel(aws.LogDebug)
		}
		sess, err := session.NewSession(config)
		if err != nil {
			logger.Log().Errorf("failed to create client, error: %v", err.Error())
			return nil, err
		}
		return sess, nil
	}
	sess, err := s.newSessionFunc()
	if err != nil {
		return nil, err
	}
	s.sess = sess

	s.blockSize = blockSize
	s.bucket = conf.Bucket
	return s, nil
}

func (s *storage) Create(ctx context.Context, path string, opts ...api2.Option) (
	io.WriteCloser, error,
) {
	log := logger.ContextLog(ctx)
	// sess, err := s.newSessionFunc()
	// if err != nil {
	// 	log.ErrorContextf(ctx, "failed to init upload, err: %v", err.Error())
	// 	return nil, err
	// }
	service := s3.New(s.sess)
	upload, err := service.CreateMultipartUpload(
		&s3.CreateMultipartUploadInput{
			Bucket: aws.String(s.bucket),
			Key:    aws.String(path),
		},
	)
	// upload, _, err := s.client.Object.InitiateMultipartUpload(ctx, path, nil)
	if err != nil {
		log.Errorf("failed to init upload, err: %v", err.Error())
		return nil, err
	}
	writer := s.ossWriteCloser(ctx, service, upload, path)
	return writer, nil
}

func (s *storage) Get(
	ctx context.Context, path string,
	options ...api2.Option,
) (io.ReadCloser, error) {
	// sess, err := s.newSessionFunc()
	// if err != nil {
	// 	log.ErrorContextf(ctx, "failed to init upload, err: %v", err.Error())
	// 	return nil, err
	// }
	service := s3.New(s.sess)
	resp, err := service.GetObject(
		&s3.GetObjectInput{
			Bucket: aws.String(s.bucket),
			Key:    aws.String(path),
		},
	)
	if err != nil {
		return nil, err
	}
	return resp.Body, err
}

func (s *storage) ossWriteCloser(
	ctx context.Context, service *s3.S3, upload *s3.CreateMultipartUploadOutput, name string,
) io.WriteCloser {
	o := &ossWriter{
		ctx:        ctx,
		upload:     upload,
		s:          s,
		service:    service,
		name:       name,
		partNumber: 1,
		complete:   &s3.CompletedMultipartUpload{},
		buf:        &bytes.Buffer{},
	}
	return o
}

type ossWriter struct {
	ctx     context.Context
	s       *storage
	service *s3.S3
	// upload     *cos.InitiateMultipartUploadResult
	upload     *s3.CreateMultipartUploadOutput
	name       string
	partNumber int64 // begin with 1
	// optcom     *cos.CompleteMultipartUploadOptions
	complete *s3.CompletedMultipartUpload
	buf      *bytes.Buffer
}

func (o *ossWriter) Write(p []byte) (n int, err error) {
	_, err = o.buf.Write(p)
	if err != nil {
		return 0, err
	}
	if o.buf.Len() < o.s.blockSize {
		return len(p), nil
	}
	err = o.uploadPart(o.buf.Bytes())
	if err != nil {
		return 0, err
	}
	o.buf = &bytes.Buffer{}
	return len(p), nil
}

func (o *ossWriter) uploadPart(reader []byte) error {
	log := logger.ContextLog(o.ctx)

	br := bytes.NewReader(reader)
	partNumber := o.partNumber
	resp, err := o.service.UploadPart(
		&s3.UploadPartInput{
			Body:       br,
			Bucket:     o.upload.Bucket,
			Key:        aws.String(o.name),
			PartNumber: aws.Int64(partNumber),
			UploadId:   o.upload.UploadId,
		},
	)

	// resp, err := o.s.client.Object.UploadPart(
	// 	o.ctx, o.name, o.upload.UploadID, o.partNumber, reader, nil,
	// )
	if err != nil {
		log.Errorf(
			"failed to upload part uploadId: %v, error: %v", *o.upload.UploadId,
			err.Error(),
		)
		return err
	}
	o.complete.Parts = append(
		o.complete.Parts, &s3.CompletedPart{
			ETag:       resp.ETag,
			PartNumber: aws.Int64(partNumber),
		},
	)

	o.partNumber++
	return nil
}

func (o *ossWriter) Close() error {
	log := logger.ContextLog(o.ctx)
	if o.buf.Len() > 0 {
		err := o.uploadPart(o.buf.Bytes())
		if err != nil {
			log.Errorf("failed to close stream, error: %v", err.Error())
			return err
		}
	}

	_, err := o.service.CompleteMultipartUpload(
		&s3.CompleteMultipartUploadInput{
			Bucket:          o.upload.Bucket,
			Key:             aws.String(o.name),
			MultipartUpload: o.complete,
			UploadId:        o.upload.UploadId,
		},
	)

	if err != nil {
		log.Errorf("failed to close stream, error: %v", err.Error())
		return err
	}
	return nil
}

func (s *storage) Info(ctx context.Context, path string) (*api2.FileInfo, error) {
	log := logger.ContextLog(ctx)
	service := s3.New(s.sess)
	resp, err := service.GetObject(
		&s3.GetObjectInput{
			Bucket: aws.String(s.bucket),
			Key:    aws.String(path),
		},
	)
	if err != nil {
		return nil, err
	}
	defer func() {
		err2 := resp.Body.Close()
		if err2 != nil {
			log.Errorf("failed to close body, err: %v", err2.Error())
		}
	}()
	fileInfo := &api2.FileInfo{
		Path:     path,
		FileName: fs.FileName(path),
		Size:     *resp.ContentLength,
	}
	return fileInfo, nil
}
