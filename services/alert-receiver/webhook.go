package main

import "encoding/json"

// AlertRef is a flattened record of one alert from an Alertmanager webhook.
type AlertRef struct {
	Alertname string `json:"alertname"`
	Severity  string `json:"severity"`
	Service   string `json:"service"`
	Status    string `json:"status"`
}

// parseAMWebhook flattens an Alertmanager webhook payload into AlertRefs.
func parseAMWebhook(body []byte) []AlertRef {
	var p struct {
		Alerts []struct {
			Status string            `json:"status"`
			Labels map[string]string `json:"labels"`
		} `json:"alerts"`
	}
	if err := json.Unmarshal(body, &p); err != nil {
		return nil
	}
	refs := make([]AlertRef, 0, len(p.Alerts))
	for _, a := range p.Alerts {
		refs = append(refs, AlertRef{
			Alertname: a.Labels["alertname"],
			Severity:  a.Labels["severity"],
			Service:   a.Labels["service"],
			Status:    a.Status,
		})
	}
	return refs
}
