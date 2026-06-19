package main

import "testing"

const alertsBody = `{"data":{"alerts":[
  {"labels":{"alertname":"FastBurn","severity":"page","service":"payments-worker"},
   "annotations":{"summary":"payments-worker fast burn","runbook_url":"docs/runbooks/fast-burn.md"},
   "state":"firing","activeAt":"2026-06-18T10:00:00Z"},
  {"labels":{"alertname":"ServiceDown","severity":"page","job":"orders-api"},
   "annotations":{"summary":"orders-api is down","runbook_url":"docs/runbooks/service-down.md"},
   "state":"pending","activeAt":"2026-06-18T10:05:00Z"}
]}}`

func TestParseAlerts(t *testing.T) {
	alerts := parseAlerts([]byte(alertsBody))
	if len(alerts) != 2 {
		t.Fatalf("want 2 alerts, got %d", len(alerts))
	}
	// sorted by activeAt desc => ServiceDown (10:05) first
	if alerts[0].Alertname != "ServiceDown" || alerts[0].Service != "orders-api" {
		t.Fatalf("bad first alert: %+v", alerts[0])
	}
	if alerts[1].Alertname != "FastBurn" || alerts[1].RunbookURL != "docs/runbooks/fast-burn.md" {
		t.Fatalf("bad second alert: %+v", alerts[1])
	}
}

func TestDeriveIncidentsOnlyFiring(t *testing.T) {
	incidents := deriveIncidents(parseAlerts([]byte(alertsBody)))
	// only FastBurn is firing; ServiceDown is pending => excluded
	if len(incidents) != 1 {
		t.Fatalf("want 1 incident, got %d", len(incidents))
	}
	inc := incidents[0]
	if inc.Service != "payments-worker" || inc.Severity != "page" || inc.AlertCount != 1 {
		t.Fatalf("bad incident: %+v", inc)
	}
	if inc.Title != "payments-worker fast burn" {
		t.Fatalf("title: %q", inc.Title)
	}
}
