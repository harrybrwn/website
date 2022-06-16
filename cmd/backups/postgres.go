package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/prometheus/client_golang/prometheus"
	"harrybrown.com/pkg/log"
)

type PGDumpFormat string

const (
	DumpFormatPlain     PGDumpFormat = "plain"
	DumpFormatCustom    PGDumpFormat = "custom"
	DumpFormatDirectory PGDumpFormat = "directory"
	DumpFormatTar       PGDumpFormat = "tar"
)

func GetPGDumpContentType(format PGDumpFormat) string {
	switch format {
	case DumpFormatPlain:
		return "application/sql"
	case DumpFormatCustom:
		return "application/vnd.postgresql-custom"
	case DumpFormatDirectory:
		return "application/vnd.postgresql-directory"
	case DumpFormatTar:
		return "application/x-tar"
	default:
		return ""
	}
}

func GetDBInfo() *DBInfo {
	format := getenv("BACKUP_POSTGRES_DUMP_FORMAT", string(DumpFormatCustom))
	if len(GetPGDumpContentType(PGDumpFormat(format))) == 0 {
		logger.WithField("pg_dump_format", format).Fatal("invalid pg_dump archive format")
	}
	return &DBInfo{
		Host:       getenv("POSTGRES_HOST", "localhost"),
		Port:       getenv("POSTGRES_PORT", "5432"),
		User:       os.Getenv("POSTGRES_USER"),
		PW:         os.Getenv("POSTGRES_PASSWORD"),
		DB:         os.Getenv("POSTGRES_DB"),
		DumpFormat: PGDumpFormat(format),
	}
}

type DBInfo struct {
	Host string
	Port string
	User string
	PW   string
	DB   string

	DumpFormat PGDumpFormat
}

type BackupInfo struct {
	Filename string
	Upload   *s3manager.UploadOutput
}

func BackupPostgres(ctx context.Context, db *DBInfo, store *s3.S3, bucket string) (*BackupInfo, error) {
	timer := prometheus.NewTimer(prometheus.ObserverFunc(pgDumpDuration.Set))
	defer timer.ObserveDuration()

	cmd := BackupCommand(db)
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, err
	}
	defer stderr.Close()
	go func() {
		logger := logger.WithFields(log.Fields{
			"source":  "stderr",
			"command": cmd.Path,
		})
		rd := bufio.NewReader(stderr)
		for {
			line, err := rd.ReadString('\n')
			if err == io.EOF {
				return
			} else if err != nil {
				logger.WithError(err).Warn("failed to read backup command stderr")
				return
			}
			logger.WithField("line", line).Warn("pg_dump stderr")
		}
	}()
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	defer stdout.Close()
	if err = cmd.Start(); err != nil {
		return nil, err
	}

	filename := pgBackupFilename(db)
	uploader := s3manager.NewUploaderWithClient(store, func(u *s3manager.Uploader) {
		u.Concurrency = 3 * s3manager.DefaultUploadConcurrency
		u.PartSize = 2 * s3manager.DefaultUploadPartSize
	})
	out, err := uploader.UploadWithContext(ctx, &s3manager.UploadInput{
		Bucket:      aws.String(bucket),
		Key:         aws.String(filename),
		ContentType: aws.String(GetPGDumpContentType(db.DumpFormat)),
		// Upload will try to seek to detect the size but we won't know the size
		// of the command output before it runs. We use an io.ReadCloser that
		// cannot seek so that the s3manager does not try to determine the size
		// of the pipe.
		// https://docs.aws.amazon.com/code-samples/latest/catalog/go-s3-upload_arbitrary_sized_stream.go.html
		Body: &nonSeekReader{reader: stdout},
		Metadata: map[string]*string{
			"pg_dump_user":   aws.String(db.User),
			"pg_dump_host":   aws.String(db.Host),
			"pg_dump_port":   aws.String(db.Port),
			"pg_dump_db":     aws.String(db.DB),
			"pg_dump_format": aws.String(string(db.DumpFormat)),
		},
	})
	if err != nil {
		return nil, err
	}
	logFields := log.Fields{"location": out.Location, "upload_id": out.UploadID}
	if out.VersionID != nil {
		logFields["version_id"] = *out.VersionID
	}
	if out.ETag != nil {
		logFields["etag"] = *out.ETag
	}
	logger.WithFields(logFields).Info("backup successfully uploaded")
	info := BackupInfo{Filename: filename, Upload: out}
	return &info, cmd.Wait()
}

func pgBackupFilename(db *DBInfo) string {
	return fmt.Sprintf(
		"postgres/pg_backup_%s-%s_%v.dump",
		db.DumpFormat,
		db.Host,
		time.Now().Unix(),
	)
}

type nonSeekReader struct {
	reader io.ReadCloser
}

func (r *nonSeekReader) Read(b []byte) (int, error) { return r.reader.Read(b) }
func (r *nonSeekReader) Close() error               { return r.reader.Close() }

// TODO check that this exists on startup
const pgDumpCommandPath = "/usr/bin/pg_dump"

func BackupCommand(db *DBInfo) *exec.Cmd {
	opts := []string{
		"--format", string(db.DumpFormat),
		"--host", db.Host,
		"--port", db.Port,
		"--username", db.User,
		"--dbname", db.DB,
		"--no-password",
	}
	cmd := exec.Command(pgDumpCommandPath, opts...)
	cmd.Env = append(cmd.Env, fmt.Sprintf("PGPASSWORD=%s", db.PW))
	return cmd
}

var pgDumpDuration = prometheus.NewGauge(prometheus.GaugeOpts{
	Name: "backups_pgdump_upload_duration",
	Help: "The time it takes to run the pg_dump command and upload the result to s3.",
})

func init() {
	prometheus.MustRegister(pgDumpDuration)
}
