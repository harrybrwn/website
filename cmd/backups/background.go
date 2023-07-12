package main

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/pkg/errors"
	"gopkg.hrry.dev/homelab/pkg/log"
)

type JobInfo struct {
	db *DBInfo
	// dynamic configuration
	config *JobConfig
}

func Start(ctx context.Context, store *s3.S3, info *JobInfo) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	ticker := time.NewTicker(info.config.Wait)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			logger.Info("running job")
			backup, err := BackupPostgres(ctx, info.db, store, info.config.Bucket)
			if err != nil {
				logger.WithError(err).Error("failed to run postgres backup")
				break
			}
			logger.WithFields(log.Fields{
				"file":      backup.Filename,
				"upload_id": backup.Upload.UploadID,
			}).Info("backup finished")
		case d := <-info.config.Channels.wait:
			logger.WithField("duration", d).Info("resetting wait duration")
			ticker.Reset(d)
		case n := <-info.config.Channels.max:
			logger.WithField("max", n).Info("resetting max jobs")
			// TODO update config
		case <-ctx.Done():
			logger.Info("stopping background backups job")
			return
		}
	}
}

func newConfig(wait time.Duration, max int, bucket string) *JobConfig {
	return &JobConfig{
		Wait:   wait,
		Max:    max,
		Bucket: bucket,
		Channels: JobConfigChannels{
			wait: make(chan time.Duration),
			max:  make(chan int),
		},
	}
}

type JobConfig struct {
	Channels JobConfigChannels `json:"-"`
	// wait time between backup jobs
	Wait time.Duration `json:"wait"`
	// maximum number of parallel backup jobs
	Max    int    `json:"max"`
	Bucket string `json:"-"`
}

type JobConfigChannels struct {
	// Updates the job wait duration
	wait chan time.Duration
	max  chan int
}

type UpdateConfigRequest struct {
	Wait *string `json:"wait"`
	Max  *int    `json:"max"`
}

type UpdateConfigResponse struct {
	Message string `json:"message"`
}

func (cfg *JobConfig) Update(req *UpdateConfigRequest) error {
	switch {
	case req.Max != nil:
		cfg.Max = *req.Max
		cfg.Channels.max <- cfg.Max
	case req.Wait != nil:
		dur, err := time.ParseDuration(*req.Wait)
		if err != nil {
			return errors.Wrap(err, "failed to update wait time config")
		}
		if dur < time.Second {
			return errors.New("duration is too small")
		}
		cfg.Channels.wait <- dur
		cfg.Wait = dur
	default:
		return errors.New("empty update request")
	}
	return nil
}
