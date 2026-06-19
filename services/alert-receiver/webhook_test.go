package main

import "testing"

func TestParseAMWebhook(t *testing.T) {
	body := []byte(`{"version":"4","status":"firing","alerts":[
		{"status":"firing","labels":{"alertname":"FastBurn","severity":"page","service":"orders-api"}},
		{"status":"resolved","labels":{"alertname":"ServiceDown","severity":"page","job":"payments-worker"}}
	]}`)
	refs := parseAMWebhook(body)
	if len(refs) != 2 {
		t.Fatalf("want 2 refs, got %d", len(refs))
	}
	if refs[0].Alertname != "FastBurn" || refs[0].Severity != "page" || refs[0].Service != "orders-api" || refs[0].Status != "firing" {
		t.Fatalf("bad first ref: %+v", refs[0])
	}
	if refs[1].Alertname != "ServiceDown" || refs[1].Status != "resolved" {
		t.Fatalf("bad second ref: %+v", refs[1])
	}
}

func TestParseAMWebhookGarbage(t *testing.T) {
	if refs := parseAMWebhook([]byte("not json")); refs != nil {
		t.Fatalf("garbage should yield nil, got %v", refs)
	}
}
