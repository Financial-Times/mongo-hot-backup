package main

import (
	"net/http"
	"time"

	health "github.com/Financial-Times/go-fthealth/v1_1"
	status "github.com/Financial-Times/service-status-go/httphandlers"

	log "github.com/Sirupsen/logrus"
	"github.com/gorilla/handlers"
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

	hc := health.TimedHealthCheck{
		HealthCheck: health.HealthCheck{
			SystemCode:  h.healthService.config.appSystemCode,
			Name:        h.healthService.config.appName,
			Description: "Creates periodic backups of mongodb.",
			Checks:      h.healthService.checks,
		},
		Timeout: 10 * time.Second,
	}

	r := mux.NewRouter()
	r.HandleFunc("/__health", http.HandlerFunc(health.Handler(hc)))
	r.Path("/__gtg").Handler(handlers.MethodHandler{"GET": http.HandlerFunc(status.NewGoodToGoHandler(h.healthService.GTG))})
	http.Handle("/", r)

	log.Fatal(http.ListenAndServe(":8080", nil))
}
