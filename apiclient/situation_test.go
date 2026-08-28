package apiclient

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func newTestClient(t *testing.T, baseURL string) *Client {
	t.Helper()
	client, err := New(baseURL)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return client
}

func TestSituationLayersReadsTheCatalog(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/situation/layers" {
			t.Errorf("path = %q", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer token-123" {
			t.Errorf("Authorization = %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":[{"id":"sigmet.intl","label":"SIGMET","entitled":true,"available":true,"attribution":"NOAA","licence":"public-domain"}]}`))
	}))
	defer server.Close()

	layers, err := newTestClient(t, server.URL).SituationLayers(context.Background(), "token-123")
	if err != nil {
		t.Fatalf("SituationLayers: %v", err)
	}
	if len(layers) != 1 {
		t.Fatalf("got %d layers, want 1", len(layers))
	}
	if layers[0].ID != "sigmet.intl" || !layers[0].Entitled {
		t.Errorf("layer = %+v", layers[0])
	}
}

func TestSituationLayerGeoJSONSurvivesAPayloadOverTheDecodeLimit(t *testing.T) {
	// The shared do() helper decodes through a 1 MiB LimitReader, so a busy
	// layer would fail there as a JSON syntax error rather than an HTTP error —
	// a genuinely confusing thing to debug. This path must not go through it.
	big := `{"type":"FeatureCollection","features":[` +
		strings.Repeat(`{"type":"Feature","geometry":{"type":"Point","coordinates":[0,0]},"properties":{"layer":"sigmet.intl"}},`, 12000)
	big = strings.TrimSuffix(big, ",") + `]}`
	if len(big) <= 1<<20 {
		t.Fatalf("test payload is %d bytes, needs to exceed 1 MiB", len(big))
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/geo+json")
		_, _ = w.Write([]byte(big))
	}))
	defer server.Close()

	body, err := newTestClient(t, server.URL).SituationLayerGeoJSON(context.Background(), "token-123", "sigmet.intl", SituationQuery{})
	if err != nil {
		t.Fatalf("SituationLayerGeoJSON: %v", err)
	}
	if len(body) != len(big) {
		t.Errorf("got %d bytes, want %d", len(body), len(big))
	}
}

func TestSituationLayerGeoJSONPassesTheViewport(t *testing.T) {
	var gotQuery string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		if !strings.HasSuffix(r.URL.Path, "/sigmet.intl.geojson") {
			t.Errorf("path = %q", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"type":"FeatureCollection","features":[]}`))
	}))
	defer server.Close()

	_, err := newTestClient(t, server.URL).SituationLayerGeoJSON(context.Background(), "token-123", "sigmet.intl",
		SituationQuery{BBox: "-10,35,30,60", Limit: 500})
	if err != nil {
		t.Fatalf("SituationLayerGeoJSON: %v", err)
	}
	if !strings.Contains(gotQuery, "bbox=") || !strings.Contains(gotQuery, "limit=500") {
		t.Errorf("query = %q", gotQuery)
	}
}

func TestSituationLayerGeoJSONMapsPaymentRequired(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusPaymentRequired)
		_, _ = w.Write([]byte(`{"error":{"code":"situation_layer_requires_upgrade"}}`))
	}))
	defer server.Close()

	_, err := newTestClient(t, server.URL).SituationLayerGeoJSON(context.Background(), "token-123", "fire.firms", SituationQuery{})
	// The web BFF renders an upgrade prompt off this sentinel, so a gated layer
	// must not surface as a generic transport failure.
	if err == nil {
		t.Fatal("a 402 was reported as success")
	}
	if !strings.Contains(err.Error(), ErrPaymentRequired.Error()) {
		t.Errorf("err = %v, want it to wrap ErrPaymentRequired", err)
	}
}

func TestSituationNewsPassesFiltersAndCursor(t *testing.T) {
	var gotQuery string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		if r.URL.Path != "/v1/situation/news" {
			t.Errorf("path = %q", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":[{"id":"n1","headline":"Wire story","attribution":"The GDELT Project"}],"next_cursor":"abc"}`))
	}))
	defer server.Close()

	page, err := newTestClient(t, server.URL).SituationNews(context.Background(), "token-123", SituationNewsQuery{
		Languages: []string{"en", "fr"},
		Countries: []string{"US"},
		Limit:     25,
		Cursor:    "prev",
	})
	if err != nil {
		t.Fatalf("SituationNews: %v", err)
	}

	for _, want := range []string{"languages=en%2Cfr", "countries=US", "limit=25", "cursor=prev"} {
		if !strings.Contains(gotQuery, want) {
			t.Errorf("query %q is missing %q", gotQuery, want)
		}
	}
	if len(page.Data) != 1 || page.Data[0].Headline != "Wire story" {
		t.Errorf("page = %+v", page.Data)
	}
	if page.NextCursor != "abc" {
		t.Errorf("NextCursor = %q", page.NextCursor)
	}
}

func TestSituationNewsOmitsEmptyFilters(t *testing.T) {
	var gotQuery string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		_, _ = w.Write([]byte(`{"data":[]}`))
	}))
	defer server.Close()

	if _, err := newTestClient(t, server.URL).SituationNews(context.Background(), "token-123", SituationNewsQuery{}); err != nil {
		t.Fatalf("SituationNews: %v", err)
	}
	// An empty filter must not travel as "languages=", which the server would
	// have to distinguish from absent.
	if strings.Contains(gotQuery, "languages=") || strings.Contains(gotQuery, "cursor=") {
		t.Errorf("query = %q, want empty filters omitted", gotQuery)
	}
}

func TestSituationPointSendsCoordinatesOrICAO(t *testing.T) {
	var gotQuery string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		_, _ = w.Write([]byte(`{"latitude":38.78,"longitude":-9.13,"icao":"LPPT"}`))
	}))
	defer server.Close()

	client := newTestClient(t, server.URL)

	if _, err := client.SituationPoint(context.Background(), "t", SituationPointQuery{ICAO: "LPPT"}); err != nil {
		t.Fatalf("by icao: %v", err)
	}
	// An ICAO query must not also send a meaningless 0,0 position, which the
	// server would read as the Gulf of Guinea.
	if !strings.Contains(gotQuery, "icao=LPPT") || strings.Contains(gotQuery, "lat=") {
		t.Errorf("query = %q", gotQuery)
	}

	if _, err := client.SituationPoint(context.Background(), "t", SituationPointQuery{Latitude: 38.78, Longitude: -9.13}); err != nil {
		t.Fatalf("by coordinates: %v", err)
	}
	if !strings.Contains(gotQuery, "lat=38.78") || !strings.Contains(gotQuery, "lon=-9.13") {
		t.Errorf("query = %q", gotQuery)
	}
}
