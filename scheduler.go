package main

import (
	log "github.com/Sirupsen/logrus"
	cron "gopkg.in/robfig/cron.v2"
)

type scheduler interface {
	SheduleBackups(colls []fullColl, cronExpr string, runAtStart bool)
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
	colls []fullColl
}

func (s *cronScheduler) SheduleBackups(colls []fullColl, cronExpr string, runAtStart bool) {
	if runAtStart {
		s.backupService.Backup(colls)
	}

	c := cron.New()
	var jobs []scheduledJob
	eID, _ := c.AddFunc(cronExpr, func() {
		s.backupService.Backup(colls)
		for _, job := range jobs {
			log.Printf("Next scheduled run: %v\n", c.Entry(job.eID).Next)
		}
	})
	jobs = append(jobs, scheduledJob{eID, colls})

	c.Start()

	for _, job := range jobs {
		log.Printf("Next scheduled run: %v\n", c.Entry(job.eID).Next)
	}
}
