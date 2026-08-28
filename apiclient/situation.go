package apiclient

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/SkyvisorInsights/skyvisor-go-shared/domain"
)

// situationLayerLimit bounds a GeoJSON download. Generous, because a busy
// hazard layer is legitimately large; bounded, because an unbounded read of a
// remote body is how one bad response exhausts memory.
const situationLayerLimit = 16 << 20

// SituationQuery narrows a situation layer read.
//
// BBox is the raw "minLon,minLat,maxLon,maxLat" string rather than a struct:
// it comes straight off a map viewport on one side and goes straight into a
// query parameter on the other, and parsing it twice would only add a place to
// disagree with the server.
type SituationQuery struct {
	BBox  string
	Since string
	Limit int
}

func (q SituationQuery) values() url.Values {
	values := url.Values{}
	if q.BBox != "" {
		values.Set("bbox", q.BBox)
	}
	if q.Since != "" {
		values.Set("since", q.Since)
	}
	if q.Limit > 0 {
		values.Set("limit", strconv.Itoa(q.Limit))
	}
	return values
}

// SituationLayers reads the layer catalogue for the caller's plan.
func (c *Client) SituationLayers(ctx context.Context, accessToken string) ([]domain.SituationLayer, error) {
	var payload struct {
		Data []domain.SituationLayer `json:"data"`
	}
	if err := c.do(ctx, http.MethodGet, "/v1/situation/layers", accessToken, nil, &payload); err != nil {
		return nil, err
	}
	return payload.Data, nil
}

// SituationLayerGeoJSON downloads one layer as a GeoJSON FeatureCollection.
//
// It deliberately does not go through do(): that helper decodes through a
// 1 MiB LimitReader, and a busy layer exceeding it would fail as a JSON syntax
// error rather than an HTTP one. The body is also passed through untouched,
// because the caller hands it to a map renderer rather than inspecting it.
func (c *Client) SituationLayerGeoJSON(ctx context.Context, accessToken, layerID string, query SituationQuery) ([]byte, error) {
	path := "/v1/situation/layers/" + url.PathEscape(layerID) + ".geojson"
	if encoded := query.values().Encode(); encoded != "" {
		path += "?" + encoded
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return nil, err
	}
	request.Header.Set("Authorization", "Bearer "+accessToken)
	request.Header.Set("Accept", "application/geo+json, application/json")

	response, err := c.http.Do(request)
	if err != nil {
		return nil, err
	}
	defer func() { _ = response.Body.Close() }()

	body, err := io.ReadAll(io.LimitReader(response.Body, situationLayerLimit))
	if err != nil {
		return nil, fmt.Errorf("read situation layer %s: %w", layerID, err)
	}

	switch {
	case response.StatusCode == http.StatusUnauthorized:
		return nil, ErrUnauthorized
	case response.StatusCode == http.StatusPaymentRequired:
		return nil, fmt.Errorf("%w: %v", ErrPaymentRequired, &APIError{
			Status:  response.StatusCode,
			Code:    "situation_layer_requires_upgrade",
			Message: string(body),
		})
	case response.StatusCode == http.StatusNotFound:
		return nil, ErrNotFound
	case response.StatusCode >= 400:
		return nil, &APIError{Status: response.StatusCode, Message: string(body)}
	}
	return body, nil
}

// SituationNewsQuery selects a page of the situation news rail.
type SituationNewsQuery struct {
	Categories []string
	Countries  []string
	Languages  []string
	Cursor     string
	Limit      int
}

func (q SituationNewsQuery) values() url.Values {
	values := url.Values{}
	// Empty filters are omitted rather than sent blank, so the server never has
	// to tell "no constraint" from "match nothing".
	if len(q.Categories) > 0 {
		values.Set("categories", strings.Join(q.Categories, ","))
	}
	if len(q.Countries) > 0 {
		values.Set("countries", strings.Join(q.Countries, ","))
	}
	if len(q.Languages) > 0 {
		values.Set("languages", strings.Join(q.Languages, ","))
	}
	if q.Cursor != "" {
		values.Set("cursor", q.Cursor)
	}
	if q.Limit > 0 {
		values.Set("limit", strconv.Itoa(q.Limit))
	}
	return values
}

// SituationNews reads one keyset-paged slice of the news rail.
func (c *Client) SituationNews(ctx context.Context, accessToken string, query SituationNewsQuery) (domain.SituationNewsPage, error) {
	path := "/v1/situation/news"
	if encoded := query.values().Encode(); encoded != "" {
		path += "?" + encoded
	}

	var page domain.SituationNewsPage
	err := c.do(ctx, http.MethodGet, path, accessToken, nil, &page)
	return page, err
}

// SituationPointQuery names a place, either by coordinates or by aerodrome.
type SituationPointQuery struct {
	Latitude  float64
	Longitude float64
	ICAO      string
}

// SituationPoint reads current conditions for one place.
//
// Blocks are absent when the caller's plan does not include them, and present
// with an unknown freshness when the upstream could not be reached — two
// different answers a client should render differently.
func (c *Client) SituationPoint(ctx context.Context, accessToken string, query SituationPointQuery) (domain.SituationPoint, error) {
	values := url.Values{}
	if icao := strings.TrimSpace(query.ICAO); icao != "" {
		values.Set("icao", icao)
	} else {
		values.Set("lat", strconv.FormatFloat(query.Latitude, 'f', 4, 64))
		values.Set("lon", strconv.FormatFloat(query.Longitude, 'f', 4, 64))
	}

	var point domain.SituationPoint
	err := c.do(ctx, http.MethodGet, "/v1/situation/point?"+values.Encode(), accessToken, nil, &point)
	return point, err
}
