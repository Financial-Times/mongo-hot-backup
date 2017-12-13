package main

import (
	"net/http"

	health "github.com/Financial-Times/go-fthealth/v1_1"
	log "github.com/Sirupsen/logrus"
	"github.com/gorilla/mux"
)

type httpService interface {
	ScheduleAndServe(colls []dbColl, cronExpr string, runAtStart bool)
}

type scheduleHTTPService struct {
	scheduler     scheduler
	healthService *healthService
}

func newScheduleHTTPService(scheduler scheduler, healthService *healthService) *scheduleHTTPService {
	return &scheduleHTTPService{scheduler, healthService}
}

func (h *scheduleHTTPService) ScheduleAndServe(colls []dbColl, cronExpr string, runAtStart bool) {
	h.scheduler.SheduleBackups(colls, cronExpr, runAtStart)

	hc := health.HealthCheck{
		SystemCode:  h.healthService.config.appSystemCode,
		Name:        h.healthService.config.appName,
		Description: "Creates periodic backups of mongodb.",
		Checks:      h.healthService.checks,
	}

	r := mux.NewRouter()
	r.HandleFunc("/__health", http.HandlerFunc(health.Handler(hc)))
	http.Handle("/", r)

	log.Fatal(http.ListenAndServe(":8080", nil))
}
