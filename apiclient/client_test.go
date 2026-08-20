package apiclient

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/SkyvisorInsights/skyvisor-go-shared/domain"
)

func TestTrustShareClientMethods(t *testing.T) {
	var gotPath, gotMethod string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath, gotMethod = r.URL.Path, r.Method
		switch r.Method {
		case http.MethodPost:
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"token":"abc","url_path":"/trust/abc","include_financials":false}`))
		case http.MethodDelete:
			w.WriteHeader(http.StatusNoContent)
		default:
			_, _ = w.Write([]byte(`{"data":[{"token":"abc"}]}`))
		}
	}))
	defer server.Close()

	client, err := New(server.URL)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	link, err := client.CreateTrustShare(context.Background(), "tok", domain.CreateTrustShare{TTLHours: 24})
	if err != nil || link.Token != "abc" {
		t.Fatalf("create = %#v err=%v", link, err)
	}
	if gotPath != "/v1/operations/trust/share" || gotMethod != http.MethodPost {
		t.Errorf("create hit %s %s", gotMethod, gotPath)
	}

	links, err := client.TrustShares(context.Background(), "tok")
	if err != nil || len(links) != 1 {
		t.Fatalf("list = %#v err=%v", links, err)
	}
	if gotPath != "/v1/operations/trust/shares" {
		t.Errorf("list hit %s", gotPath)
	}

	if err := client.RevokeTrustShare(context.Background(), "tok", "abc"); err != nil {
		t.Fatalf("revoke: %v", err)
	}
	if gotPath != "/v1/trust-shares/abc" {
		t.Errorf("revoke hit %s", gotPath)
	}
}

func TestPublicTrustReportClientMethod(t *testing.T) {
	var gotPath, gotAuthHeader string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotAuthHeader = r.Header.Get("Authorization")
		_, _ = w.Write([]byte(`{"evaluated_predictions":42,"precision":0.78}`))
	}))
	defer server.Close()

	client, err := New(server.URL)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	report, err := client.PublicTrustReport(context.Background(), "abc")
	if err != nil {
		t.Fatalf("PublicTrustReport: %v", err)
	}
	if report.EvaluatedPredictions != 42 {
		t.Errorf("report = %#v", report)
	}
	if gotPath != "/public/trust-shares/abc" {
		t.Errorf("public report hit %s", gotPath)
	}
	// c.do always sets "Authorization: Bearer "+accessToken, even with an
	// empty access token. HTTP header parsing trims the trailing optional
	// whitespace in transit, so the server observes "Bearer" (no trailing
	// space) here. The public route is registered outside bearer-auth
	// middleware and ignores this header entirely, so it's harmless.
	if gotAuthHeader != "Bearer" {
		t.Errorf("Authorization header = %q, want %q", gotAuthHeader, "Bearer")
	}
}
