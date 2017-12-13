package main

import (
	"errors"
	"fmt"
	"time"

	health "github.com/Financial-Times/go-fthealth/v1_1"
)

type healthService struct {
	statusKeeper statusKeeper
	config       healthConfig
	checks       []health.Check
}

type healthConfig struct {
	appSystemCode string
	appName       string
}

func newHealthService(statusKeeper statusKeeper, colls []fullColl, config healthConfig) *healthService {
	hService := &healthService{
		statusKeeper: statusKeeper,
		config:       config,
	}
	hService.checks = []health.Check{}
	for _, coll := range colls {
		hService.checks = append(hService.checks, hService.backupImageCheck(coll))
	}
	return hService
}

func (h *healthService) backupImageCheck(coll fullColl) health.Check {
	return health.Check{
		BusinessImpact:   "Restoring the database in case of an issue will have to be done from older backups. It will take longer to restore systems to a clean state.",
		Name:             fmt.Sprintf("%s/%s", coll.database, coll.collection),
		PanicGuide:       "https://dewey.ft.com/mongo-hot-backup.html",
		Severity:         1,
		TechnicalSummary: fmt.Sprintf("A backup for database %s, collection %s has not been made in the last 26 hours.", coll.database, coll.collection),
		Checker:          func() (string, error) { return h.verifyExistingBackupImage(coll) },
	}
}

func (h *healthService) verifyExistingBackupImage(coll fullColl) (string, error) {
	result, err := h.statusKeeper.Get(coll)
	if err != nil {
		return err.Error(), err
	}

	if time.Since(result.Timestamp).Hours() > 26 {
		msg := "Last backup more than 26 hours ago. Check backup was taken."
		return msg, errors.New(msg)
	}
	if !result.Success {
		msg := "Backup failed. Check backup was taken."
		return msg, errors.New(msg)
	}

	return "", nil
}
