package main

import (
	health "github.com/Financial-Times/go-fthealth/v1_1"
	"github.com/boltdb/bolt"
	"fmt"
	"encoding/json"
	"time"
	"errors"
)

type healthService struct {
	db             *bolt.DB
	config         healthConfig
	checks         []health.Check
}

type healthConfig struct {
	appSystemCode string
	appName       string
}

func newHealthService(db *bolt.DB, collections []fullColl, config healthConfig) *healthService {
	hService := &healthService{
		db: db,
		config: config,
	}
	hService.checks = []health.Check{}
	for _, collection := range collections {
		hService.checks = append(hService.checks, hService.backupImageCheck(collection.database, collection.collection))
	}
	return hService
}

func (h *healthService) backupImageCheck(database string, collection string) health.Check {
	return health.Check{
		BusinessImpact:   "Restoring the database in case of an issue will have to be done from older backups. It will take longer to restore systems to a clean state.",
		Name:             collection,
		PanicGuide:       "https://dewey.ft.com/mongo-hot-backup.html",
		Severity:         1,
		TechnicalSummary: fmt.Sprintf("A backup for database %s, collection %s has not been made in the last 26 hours.", database, collection),
		Checker:          func() (string, error) { return h.verifyExistingBackupImage(database, collection) },
	}
}

func (h *healthService) verifyExistingBackupImage(database string, collection string) (string, error) {
	err := h.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("Results"))
		v := b.Get([]byte(fmt.Sprintf("%s/%s", database, collection)))

		result := scheduledJobResult{}

		json.Unmarshal(v, &result)

		if time.Since(result.Timestamp).Hours() > 26 {
			return errors.New("Last backup more than 26 hours ago. Check backup was taken.")
		}
		if !result.Success {
			return errors.New("Backup failed. Check backup was taken.")
		}
		return nil
	})
	if err != nil {
		return err.Error(), err
	}
	return "", nil
}
