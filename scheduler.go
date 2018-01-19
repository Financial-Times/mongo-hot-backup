package main

import (
	log "github.com/Sirupsen/logrus"
	cron "gopkg.in/robfig/cron.v2"
)

type scheduler interface {
	SheduleBackups(colls []dbColl, cronExpr string, runAtStart bool)
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

func (s *cronScheduler) SheduleBackups(colls []dbColl, cronExpr string, runAtStart bool) {
	if runAtStart {
		err := s.backupService.Backup(colls)
		if err != nil {
			log.Errorf("Error making scheduled backup: %v\n", err)
		}

	}

	c := cron.New()
	var jobs []scheduledJob
	eID, _ := c.AddFunc(cronExpr, func() {
		err := s.backupService.Backup(colls)
		if err != nil {
			log.Errorf("Error making scheduled backup: %v\n", err)
		}
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
