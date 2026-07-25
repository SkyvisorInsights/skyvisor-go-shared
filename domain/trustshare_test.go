package domain

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestPublicTrustReportOmitsRedactedMoney(t *testing.T) {
	report := PublicTrustReport{
		EvaluatedPredictions: 42,
		Precision:            0.78,
		GeneratedAt:          time.Unix(0, 0).UTC(),
		ExpiresAt:            time.Unix(0, 0).UTC(),
	}
	blob, err := json.Marshal(report)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if strings.Contains(string(blob), "avoided_cost_minor") {
		t.Errorf("redacted report must omit avoided_cost_minor, got %s", blob)
	}
	if strings.Contains(string(blob), "breakdowns") {
		t.Errorf("redacted report must omit breakdowns, got %s", blob)
	}
}

func TestPublicTrustReportKeepsZeroMoneyWhenShared(t *testing.T) {
	zero := int64(0)
	report := PublicTrustReport{AvoidedCostMinor: &zero, Currency: "GBP"}
	blob, err := json.Marshal(report)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if !strings.Contains(string(blob), `"avoided_cost_minor":0`) {
		t.Errorf("shared zero money must serialise as 0, got %s", blob)
	}
}

func TestTrustShareTTLBounds(t *testing.T) {
	if TrustShareDefaultTTLHours != 168 {
		t.Errorf("default TTL = %d, want 168", TrustShareDefaultTTLHours)
	}
	if TrustShareMaxTTLHours != 720 {
		t.Errorf("max TTL = %d, want 720", TrustShareMaxTTLHours)
	}
	if TrustLowSampleThreshold != 20 {
		t.Errorf("low sample threshold = %d, want 20", TrustLowSampleThreshold)
	}
}

func TestDecisionTrustMetricsHasCurrency(t *testing.T) {
	metrics := DecisionTrustMetrics{Currency: "EUR"}
	blob, err := json.Marshal(metrics)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if !strings.Contains(string(blob), `"currency":"EUR"`) {
		t.Errorf("metrics must carry currency, got %s", blob)
	}
}
