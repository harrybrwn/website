package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"path/filepath"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/smithy-go"
)

type RWFile interface {
	Stat() (fs.FileInfo, error)
	Write(p []byte) (n int, err error)
	Close() error
}

type ReaderFromFile interface {
	fs.File
	ReadFrom(io.Reader) (int64, error)
}

type WriteFS interface {
	// Open(name string) (RWFile, error)
	Create(name string) (RWFile error)
}

func NewS3FS(client *s3.Client) *S3FileSystem {
	return &S3FileSystem{s3: client}
}

type S3FileSystem struct {
	s3 *s3.Client
}

func (sys *S3FileSystem) Open(name string) (fs.File, error) {
	return sys.OpenContext(context.Background(), name)
}

func (sys *S3FileSystem) Stat(name string) (fs.FileInfo, error) {
	return sys.StatContext(context.Background(), name)
}

type S3FileError struct {
	inner error
	fsErr error
}

func (e *S3FileError) Error() string {
	return fmt.Sprintf("s3 filesystem: %s: %s", e.inner.Error(), e.fsErr.Error())
}

func (e *S3FileError) Is(other error) bool {
	if errors.Is(e.inner, other) || errors.Is(e.fsErr, other) {
		return true
	}
	return false
}

func (sys *S3FileSystem) OpenContext(ctx context.Context, name string) (fs.File, error) {
	bucket, key := sys.split(name)
	object, err := sys.s3.GetObject(ctx, &s3.GetObjectInput{
		Bucket: bucket,
		Key:    key,
	})
	if err != nil {
		return nil, handleS3Err(err)
	}
	return &s3File{
		bucket: bucket,
		key:    key,
		object: object,
	}, nil
}

func (sys *S3FileSystem) StatContext(ctx context.Context, name string) (fs.FileInfo, error) {
	bucket, key := sys.split(name)
	info := s3FileInfo{
		bucket: bucket,
		key:    key,
		mod:    nil,
		length: 0,
	}
	var (
		err  error
		head *s3.HeadObjectOutput
	)
	if key == nil {
		_, err = sys.s3.HeadBucket(ctx, &s3.HeadBucketInput{
			Bucket: bucket,
		})
	} else {
		head, err = sys.s3.HeadObject(ctx, &s3.HeadObjectInput{
			Bucket: bucket,
			Key:    key,
		})
	}
	if err != nil {
		return nil, &S3FileError{inner: err, fsErr: fs.ErrNotExist}
	}
	if head != nil {
		info.mod = head.LastModified
		info.length = head.ContentLength
	}
	return &info, nil
}

type RWC struct {
}

func (sys *S3FileSystem) CreateContext(ctx context.Context, name string) (RWFile, error) {
	bucket, key := sys.split(name)
	var b bytes.Buffer

	return &s3File{
		bucket: bucket,
		key:    key,
		rw:     &b,
	}, nil
}

// split a name into (bucket, key) pair
func (sys *S3FileSystem) split(name string) (*string, *string) {
	parts := strings.Split(name, string(filepath.Separator))
	// Only bucket name
	if len(parts) < 2 {
		return &name, nil
	}
	key := filepath.Join(parts[1:]...)
	return &parts[0], &key
}

type s3File struct {
	bucket *string
	key    *string

	// for reads
	object *s3.GetObjectOutput
	// for writes
	// rwc io.ReadWriteCloser
	rw io.ReadWriter

	s3 *s3.Client
}

func (f *s3File) Read(b []byte) (int, error) {
	return f.object.Body.Read(b)
}

func (f *s3File) Write(b []byte) (int, error) {
	return f.rw.Write(b)
}

func (f *s3File) ReadFrom(r io.Reader) (int64, error) {
	_, err := f.s3.PutObject(context.Background(), &s3.PutObjectInput{
		Bucket: f.bucket,
		Key:    f.key,
		Body:   r,
	})
	if err != nil {
		return 0, err
	}
	return 0, nil
}

func (f *s3File) Close() error {
	if f.object != nil {
		return f.object.Body.Close()
	}
	return nil
}

func (f *s3File) Stat() (fs.FileInfo, error) {
	return &s3FileInfo{
		bucket: f.bucket,
		key:    f.key,
		mod:    f.object.LastModified,
		length: f.object.ContentLength,
	}, nil
}

var (
	_ fs.FS     = (*S3FileSystem)(nil)
	_ fs.StatFS = (*S3FileSystem)(nil)
	_ fs.File   = (*s3File)(nil)
)

type s3FileInfo struct {
	bucket *string
	key    *string
	// Last-Modified
	mod *time.Time
	// Content-Length
	length int64
}

func (info *s3FileInfo) Name() string {
	if info.bucket != nil {
		return filepath.Join(*info.bucket, *info.key)
	}
	return *info.key
}
func (info *s3FileInfo) Size() int64  { return info.length }
func (*s3FileInfo) Mode() fs.FileMode { return fs.FileMode(0644) }
func (info *s3FileInfo) ModTime() time.Time {
	if info.mod == nil {
		return time.Time{}
	}
	return *info.mod
}
func (info *s3FileInfo) IsDir() bool { return info.key == nil }
func (*s3FileInfo) Sys() any         { return &S3FileSystem{} }

var _ fs.FileInfo = (*s3FileInfo)(nil)

func handleS3Err(err error) error {
	// TODO handle not found to return fs.ErrNotExist
	switch e := err.(type) {
	case *types.BucketAlreadyExists:
		return &S3FileError{inner: e, fsErr: fs.ErrExist}
	case *smithy.OperationError:
		return &S3FileError{inner: e}
	default:
		return e
	}
}
