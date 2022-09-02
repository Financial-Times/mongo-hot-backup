package main

import (
	"context"

	log "github.com/sirupsen/logrus"
	"gopkg.in/robfig/cron.v2"
)

type scheduler interface {
	ScheduleBackups(colls []dbColl, cronExpr string, runAtStart bool)
}

type cronScheduler struct {
	backupService backupService
	statusKeeper  statusKeeper
}

func newCronScheduler(backupService backupService, statusKeeper statusKeeper) *cronScheduler {
	return &cronScheduler{backupService, statusKeeper}
}

type scheduledJob struct {
	eID   cron.EntryID
	colls []dbColl
}

func (s *cronScheduler) ScheduleBackups(colls []dbColl, cronExpr string, runAtStart bool) {
	ctx := context.Background()

	if runAtStart {
		err := s.backupService.Backup(ctx, colls)
		if err != nil {
			log.Errorf("Error making scheduled backup: %v", err)
		}

	}

	c := cron.New()
	var jobs []scheduledJob
	eID, _ := c.AddFunc(cronExpr, func() {
		if err := s.backupService.Backup(ctx, colls); err != nil {
			log.Errorf("Error making scheduled backup: %v", err)
		}
		for _, job := range jobs {
			log.Printf("Next scheduled run: %v", c.Entry(job.eID).Next)
		}
	})
	jobs = append(jobs, scheduledJob{eID, colls})

	c.Start()

	for _, job := range jobs {
		log.Printf("Next scheduled run: %v", c.Entry(job.eID).Next)
	}
}
