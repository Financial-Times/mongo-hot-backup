package main

import (
	"errors"
	"fmt"
	"time"

	health "github.com/Financial-Times/go-fthealth/v1_1"
	"github.com/Financial-Times/service-status-go/gtg"
)

type healthService struct {
	hours        int
	statusKeeper statusKeeper
	config       healthConfig
	checks       []health.Check
	gtgs         []gtg.StatusChecker
}

type healthConfig struct {
	appSystemCode string
	appName       string
}

func newHealthService(hours int, statusKeeper statusKeeper, colls []dbColl, config healthConfig) *healthService {
	hService := &healthService{
		hours:        hours,
		statusKeeper: statusKeeper,
		config:       config,
	}
	hService.checks = []health.Check{}
	hService.gtgs = []gtg.StatusChecker{}
	for _, coll := range colls {
		hService.checks = append(hService.checks, hService.backupImageCheck(coll))
		gtgF := func() gtg.Status {
			return gtgCheck(
				func() (string, error) {
					return hService.verifyExistingBackupImage(coll)
				},
			)
		}
		hService.gtgs = append(hService.gtgs, gtgF)
	}
	return hService
}

func (h *healthService) backupImageCheck(coll dbColl) health.Check {
	return health.Check{
		BusinessImpact:   "Restoring the database in case of an issue will have to be done from older backups. It will take longer to restore systems to a clean state.",
		Name:             fmt.Sprintf("%s/%s", coll.database, coll.collection),
		PanicGuide:       fmt.Sprintf("https://runbooks.ftops.tech/%s", systemCode),
		Severity:         2,
		TechnicalSummary: fmt.Sprintf("A backup for database %s, collection %s has not been made in the last %d hours.", coll.database, coll.collection, h.hours),
		Checker:          func() (string, error) { return h.verifyExistingBackupImage(coll) },
	}
}

func (h *healthService) verifyExistingBackupImage(coll dbColl) (string, error) {
	result, err := h.statusKeeper.Get(coll)
	if err != nil {
		return err.Error(), err
	}

	if int(time.Since(result.Timestamp).Hours()) > h.hours {
		msg := fmt.Sprintf("Last backup more than %d hours ago. Check backup was taken.", h.hours)
		return msg, errors.New(msg)
	}
	if !result.Success {
		msg := "Backup failed. Check backup was taken."
		return msg, errors.New(msg)
	}

	return "", nil
}

func (h *healthService) GTG() gtg.Status {
	return gtg.FailFastParallelCheck(h.gtgs)()
}

func gtgCheck(handler func() (string, error)) gtg.Status {
	if _, err := handler(); err != nil {
		return gtg.Status{GoodToGo: false, Message: err.Error()}
	}
	return gtg.Status{GoodToGo: true}
}
