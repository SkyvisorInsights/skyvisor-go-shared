package apiclient

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/SkyvisorInsights/skyvisor-go-shared/domain"
)

var (
	ErrNotFound        = errors.New("resource not found")
	ErrUnauthorized    = errors.New("API session is not authorized")
	ErrPaymentRequired = errors.New("plan upgrade required")
	ErrConflict        = errors.New("resource conflict")
)

// Client is a thin, typed HTTP client for skyvisor-api. Every call takes the
// caller's OIDC access token; the client itself holds no user state.
type Client struct {
	baseURL    string
	http       *http.Client
	stream     *http.Client
	clientName string // optional X-SkyVisor-Client value (e.g. "mcp")
}

func New(baseURL string) (*Client, error) {
	parsed, err := url.Parse(baseURL)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
		return nil, errors.New("skyvisor-api base URL must be an absolute HTTP(S) URL")
	}
	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		http:    &http.Client{Timeout: 15 * time.Second},
		stream:  &http.Client{},
	}, nil
}

// WithClientName returns a shallow copy that sends X-SkyVisor-Client on requests.
func (c *Client) WithClientName(name string) *Client {
	if c == nil {
		return nil
	}
	copy := *c
	copy.clientName = strings.TrimSpace(name)
	return &copy
}

// StreamEvents opens the authenticated SSE stream and returns its body.
func (c *Client) StreamEvents(ctx context.Context, accessToken string) (io.ReadCloser, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/v1/events", nil)
	if err != nil {
		return nil, fmt.Errorf("build events request: %w", err)
	}
	request.Header.Set("Authorization", "Bearer "+accessToken)
	request.Header.Set("Accept", "text/event-stream")

	response, err := c.stream.Do(request)
	if err != nil {
		return nil, fmt.Errorf("open events stream: %w", err)
	}
	if response.StatusCode == http.StatusUnauthorized {
		_ = response.Body.Close()
		return nil, ErrUnauthorized
	}
	if response.StatusCode >= 400 {
		defer func() { _ = response.Body.Close() }()
		return nil, apiError(response)
	}
	return response.Body, nil
}

// Me is the /v1/me response (account + identity + entitlements).
type Me struct {
	Account struct {
		ID                   string    `json:"id"`
		CreatedAt            time.Time `json:"created_at"`
		Plan                 string    `json:"plan"`
		WatchLimit           int       `json:"watch_limit"`
		ProTrialUsed         bool      `json:"pro_trial_used"`
		Email                string    `json:"email,omitempty"`
		EmailAlerts          bool      `json:"email_alerts"`
		StripeCustomerID     string    `json:"stripe_customer_id,omitempty"`
		StripeSubscriptionID string    `json:"stripe_subscription_id,omitempty"`
	} `json:"account"`
	Identity struct {
		Issuer  string `json:"issuer"`
		Subject string `json:"subject"`
		Email   string `json:"email"`
		Name    string `json:"name"`
	} `json:"identity"`
	Entitlements domain.Entitlements `json:"entitlements"`
}

type CheckoutSession struct {
	ID  string `json:"id"`
	URL string `json:"url"`
}

// APIError is a structured skyvisor-api error response.
type APIError struct {
	Status  int
	Code    string
	Message string
}

func (e *APIError) Error() string {
	if e == nil {
		return "skyvisor-api error"
	}
	return fmt.Sprintf("skyvisor-api %d %s: %s", e.Status, e.Code, e.Message)
}

func (c *Client) Me(ctx context.Context, accessToken string) (Me, error) {
	var me Me
	err := c.do(ctx, http.MethodGet, "/v1/me", accessToken, nil, &me)
	return me, err
}

func (c *Client) ListTrips(ctx context.Context, accessToken string) ([]domain.Trip, error) {
	var payload struct {
		Data []domain.Trip `json:"data"`
	}
	if err := c.do(ctx, http.MethodGet, "/v1/trips", accessToken, nil, &payload); err != nil {
		return nil, err
	}
	return payload.Data, nil
}

func (c *Client) CreateTrip(ctx context.Context, accessToken string, input domain.CreateTrip) (domain.Trip, error) {
	var trip domain.Trip
	err := c.do(ctx, http.MethodPost, "/v1/trips", accessToken, input, &trip)
	return trip, err
}

func (c *Client) ImportTrip(ctx context.Context, accessToken, text string) (domain.Trip, error) {
	var trip domain.Trip
	err := c.do(ctx, http.MethodPost, "/v1/trips/import", accessToken, map[string]string{"text": text}, &trip)
	return trip, err
}

func (c *Client) ImportTripPDF(ctx context.Context, accessToken, filename string, pdf io.Reader) (domain.Trip, error) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", filename)
	if err != nil {
		return domain.Trip{}, fmt.Errorf("build upload: %w", err)
	}
	if _, err = io.Copy(part, pdf); err != nil {
		return domain.Trip{}, fmt.Errorf("copy upload: %w", err)
	}
	if err = writer.Close(); err != nil {
		return domain.Trip{}, fmt.Errorf("finalize upload: %w", err)
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/v1/trips/import/pdf", body)
	if err != nil {
		return domain.Trip{}, fmt.Errorf("build request: %w", err)
	}
	request.Header.Set("Authorization", "Bearer "+accessToken)
	request.Header.Set("Content-Type", writer.FormDataContentType())
	request.Header.Set("Accept", "application/json")

	response, err := c.http.Do(request)
	if err != nil {
		return domain.Trip{}, fmt.Errorf("call skyvisor-api: %w", err)
	}
	defer func() {
		_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, 1<<16))
		_ = response.Body.Close()
	}()
	switch {
	case response.StatusCode == http.StatusUnauthorized:
		return domain.Trip{}, ErrUnauthorized
	case response.StatusCode == http.StatusRequestEntityTooLarge:
		return domain.Trip{}, &APIError{Status: response.StatusCode, Code: "file_too_large", Message: "The PDF must be at most 10MB"}
	case response.StatusCode >= 400:
		return domain.Trip{}, apiError(response)
	}
	var trip domain.Trip
	if err := json.NewDecoder(io.LimitReader(response.Body, 1<<20)).Decode(&trip); err != nil {
		return domain.Trip{}, fmt.Errorf("decode skyvisor-api response: %w", err)
	}
	return trip, nil
}

func (c *Client) DeleteTrip(ctx context.Context, accessToken, id string) error {
	return c.do(ctx, http.MethodDelete, "/v1/trips/"+url.PathEscape(id), accessToken, nil, nil)
}

func (c *Client) GetTrip(ctx context.Context, accessToken, id string) (domain.Trip, error) {
	var trip domain.Trip
	err := c.do(ctx, http.MethodGet, "/v1/trips/"+url.PathEscape(id), accessToken, nil, &trip)
	return trip, err
}

func (c *Client) GetFlight(ctx context.Context, accessToken, number string) (domain.Flight, error) {
	var flight domain.Flight
	err := c.do(ctx, http.MethodGet, "/v1/flights/"+url.PathEscape(number), accessToken, nil, &flight)
	return flight, err
}

// ListLiveFlights returns flights matching optional filters (Phase 3 tracker).
func (c *Client) ListLiveFlights(ctx context.Context, accessToken string, q LiveFlightQuery) ([]domain.Flight, error) {
	values := url.Values{}
	if q.Airline != "" {
		values.Set("airline", q.Airline)
	}
	if q.DepIATA != "" {
		values.Set("dep_iata", q.DepIATA)
	}
	if q.ArrIATA != "" {
		values.Set("arr_iata", q.ArrIATA)
	}
	if q.Status != "" {
		values.Set("status", q.Status)
	}
	if q.Limit > 0 {
		values.Set("limit", fmt.Sprintf("%d", q.Limit))
	}
	path := "/v1/flights"
	if encoded := values.Encode(); encoded != "" {
		path += "?" + encoded
	}
	var payload struct {
		Data []domain.Flight `json:"data"`
	}
	if err := c.do(ctx, http.MethodGet, path, accessToken, nil, &payload); err != nil {
		return nil, err
	}
	return payload.Data, nil
}

// LiveFlightQuery filters for GET /v1/flights.
type LiveFlightQuery struct {
	Airline string
	DepIATA string
	ArrIATA string
	Status  string
	Limit   int
}

func (c *Client) ListWatches(ctx context.Context, accessToken string) ([]domain.Watch, error) {
	var payload struct {
		Data []domain.Watch `json:"data"`
	}
	if err := c.do(ctx, http.MethodGet, "/v1/watches", accessToken, nil, &payload); err != nil {
		return nil, err
	}
	return payload.Data, nil
}

func (c *Client) CreateWatch(ctx context.Context, accessToken string, input domain.CreateWatch) (domain.Watch, error) {
	var watch domain.Watch
	err := c.do(ctx, http.MethodPost, "/v1/watches", accessToken, input, &watch)
	return watch, err
}

func (c *Client) DeleteWatch(ctx context.Context, accessToken, id string) error {
	return c.do(ctx, http.MethodDelete, "/v1/watches/"+url.PathEscape(id), accessToken, nil, nil)
}

func (c *Client) CreateShare(ctx context.Context, accessToken, watchID string) (domain.ShareLink, error) {
	var link domain.ShareLink
	err := c.do(ctx, http.MethodPost, "/v1/watches/"+url.PathEscape(watchID)+"/share", accessToken, nil, &link)
	return link, err
}

func (c *Client) RevokeShare(ctx context.Context, accessToken, token string) error {
	return c.do(ctx, http.MethodDelete, "/v1/shares/"+url.PathEscape(token), accessToken, nil, nil)
}

func (c *Client) RegisterDevice(ctx context.Context, accessToken string, input domain.RegisterDevice) (domain.PushDevice, error) {
	var device domain.PushDevice
	err := c.do(ctx, http.MethodPost, "/v1/devices", accessToken, input, &device)
	return device, err
}

func (c *Client) PublicShare(ctx context.Context, token string) (domain.PublicShare, error) {
	var view domain.PublicShare
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/public/shares/"+url.PathEscape(token), nil)
	if err != nil {
		return domain.PublicShare{}, err
	}
	request.Header.Set("Accept", "application/json")
	response, err := c.http.Do(request)
	if err != nil {
		return domain.PublicShare{}, err
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode == http.StatusNotFound {
		return domain.PublicShare{}, ErrNotFound
	}
	if response.StatusCode == http.StatusGone {
		return domain.PublicShare{}, fmt.Errorf("share link expired")
	}
	if response.StatusCode >= 400 {
		return domain.PublicShare{}, apiError(response)
	}
	if err := json.NewDecoder(io.LimitReader(response.Body, 1<<20)).Decode(&view); err != nil {
		return domain.PublicShare{}, err
	}
	return view, nil
}

func (c *Client) CreateCheckout(ctx context.Context, accessToken string) (CheckoutSession, error) {
	var session CheckoutSession
	err := c.do(ctx, http.MethodPost, "/v1/billing/checkout", accessToken, map[string]any{}, &session)
	return session, err
}

func (c *Client) CreateBillingPortal(ctx context.Context, accessToken string) (CheckoutSession, error) {
	var session CheckoutSession
	err := c.do(ctx, http.MethodPost, "/v1/billing/portal", accessToken, map[string]any{}, &session)
	return session, err
}

func (c *Client) DevActivatePro(ctx context.Context, accessToken string) error {
	return c.do(ctx, http.MethodPost, "/v1/billing/dev-activate-pro", accessToken, nil, &map[string]any{})
}

func (c *Client) SetEmailAlerts(ctx context.Context, accessToken string, enabled bool) error {
	return c.do(ctx, http.MethodPatch, "/v1/me/preferences", accessToken, map[string]any{"email_alerts": enabled}, &map[string]any{})
}

func (c *Client) AskAssistant(ctx context.Context, accessToken, question, tripID string) (domain.AssistantResponse, error) {
	var response domain.AssistantResponse
	body := map[string]string{"question": question}
	if tripID != "" {
		body["trip_id"] = tripID
	}
	err := c.do(ctx, http.MethodPost, "/v1/assistant", accessToken, body, &response)
	return response, err
}

// AirportBoardQuery filters GET /v1/airports/{iata}/board.
type AirportBoardQuery struct {
	Direction string
	Status    string
	Limit     int
}

func (c *Client) AirportBoard(ctx context.Context, accessToken, iata string, q AirportBoardQuery) (domain.AirportBoard, error) {
	values := url.Values{}
	if q.Direction != "" {
		values.Set("direction", q.Direction)
	}
	if q.Status != "" {
		values.Set("status", q.Status)
	}
	if q.Limit > 0 {
		values.Set("limit", fmt.Sprintf("%d", q.Limit))
	}
	path := "/v1/airports/" + url.PathEscape(strings.ToUpper(strings.TrimSpace(iata))) + "/board"
	if encoded := values.Encode(); encoded != "" {
		path += "?" + encoded
	}
	var board domain.AirportBoard
	err := c.do(ctx, http.MethodGet, path, accessToken, nil, &board)
	return board, err
}

// AnalyticsQuery filters GET /v1/analytics.
type AnalyticsQuery struct {
	Airline    string
	Airport    string
	WindowDays int
	Limit      int
}

func (c *Client) Analytics(ctx context.Context, accessToken string, q AnalyticsQuery) (domain.AnalyticsReport, error) {
	values := url.Values{}
	if q.Airline != "" {
		values.Set("airline", q.Airline)
	}
	if q.Airport != "" {
		values.Set("airport", q.Airport)
	}
	if q.WindowDays > 0 {
		values.Set("window_days", fmt.Sprintf("%d", q.WindowDays))
	}
	if q.Limit > 0 {
		values.Set("limit", fmt.Sprintf("%d", q.Limit))
	}
	path := "/v1/analytics"
	if encoded := values.Encode(); encoded != "" {
		path += "?" + encoded
	}
	var report domain.AnalyticsReport
	err := c.do(ctx, http.MethodGet, path, accessToken, nil, &report)
	return report, err
}

// AnalyticsExportCSV downloads the Pro-only CSV export body.
func (c *Client) AnalyticsExportCSV(ctx context.Context, accessToken string, q AnalyticsQuery) ([]byte, error) {
	values := url.Values{}
	if q.Airline != "" {
		values.Set("airline", q.Airline)
	}
	if q.Airport != "" {
		values.Set("airport", q.Airport)
	}
	if q.WindowDays > 0 {
		values.Set("window_days", fmt.Sprintf("%d", q.WindowDays))
	}
	if q.Limit > 0 {
		values.Set("limit", fmt.Sprintf("%d", q.Limit))
	}
	path := "/v1/analytics/export.csv"
	if encoded := values.Encode(); encoded != "" {
		path += "?" + encoded
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return nil, err
	}
	request.Header.Set("Authorization", "Bearer "+accessToken)
	request.Header.Set("Accept", "text/csv")
	response, err := c.http.Do(request)
	if err != nil {
		return nil, err
	}
	defer func() { _ = response.Body.Close() }()
	body, _ := io.ReadAll(io.LimitReader(response.Body, 2<<20))
	switch {
	case response.StatusCode == http.StatusUnauthorized:
		return nil, ErrUnauthorized
	case response.StatusCode == http.StatusPaymentRequired:
		return nil, fmt.Errorf("%w: %v", ErrPaymentRequired, &APIError{Status: response.StatusCode, Code: "export_requires_pro", Message: string(body)})
	case response.StatusCode >= 400:
		return nil, &APIError{Status: response.StatusCode, Message: string(body)}
	}
	return body, nil
}

func (c *Client) TripWhatIf(ctx context.Context, accessToken, tripID string, req domain.WhatIfRequest) (domain.WhatIfResult, error) {
	var result domain.WhatIfResult
	err := c.do(ctx, http.MethodPost, "/v1/trips/"+url.PathEscape(tripID)+"/what-if", accessToken, req, &result)
	return result, err
}

func (c *Client) LogisticsOverview(ctx context.Context, accessToken string, q domain.LogisticsQuery) (domain.LogisticsOverview, error) {
	values := url.Values{}
	if q.Airline != "" {
		values.Set("airline", q.Airline)
	}
	if q.Airport != "" {
		values.Set("airport", q.Airport)
	}
	if q.Limit > 0 {
		values.Set("limit", fmt.Sprintf("%d", q.Limit))
	}
	path := "/v1/logistics/overview"
	if encoded := values.Encode(); encoded != "" {
		path += "?" + encoded
	}
	var overview domain.LogisticsOverview
	err := c.do(ctx, http.MethodGet, path, accessToken, nil, &overview)
	return overview, err
}

func (c *Client) GetTeam(ctx context.Context, accessToken string) (domain.Team, error) {
	var team domain.Team
	err := c.do(ctx, http.MethodGet, "/v1/team", accessToken, nil, &team)
	return team, err
}

func (c *Client) CreateTeam(ctx context.Context, accessToken string, name string) (domain.Team, error) {
	var team domain.Team
	err := c.do(ctx, http.MethodPost, "/v1/team", accessToken, domain.CreateTeam{Name: name}, &team)
	return team, err
}

func (c *Client) JoinTeam(ctx context.Context, accessToken, inviteCode string) (domain.Team, error) {
	var team domain.Team
	err := c.do(ctx, http.MethodPost, "/v1/team/join", accessToken, domain.JoinTeam{InviteCode: inviteCode}, &team)
	return team, err
}

func (c *Client) DevActivateBusiness(ctx context.Context, accessToken string) error {
	return c.do(ctx, http.MethodPost, "/v1/billing/dev-activate-business", accessToken, nil, nil)
}

func (c *Client) GetUsage(ctx context.Context, accessToken string) (domain.UsageSnapshot, error) {
	var snap domain.UsageSnapshot
	err := c.do(ctx, http.MethodGet, "/v1/usage", accessToken, nil, &snap)
	return snap, err
}

// OperationsDashboard returns the account-scoped operational command center.
func (c *Client) OperationsDashboard(ctx context.Context, accessToken string) (domain.OperationsDashboard, error) {
	var dashboard domain.OperationsDashboard
	err := c.do(ctx, http.MethodGet, "/v1/operations/dashboard", accessToken, nil, &dashboard)
	return dashboard, err
}

func (c *Client) ListOperationalCases(ctx context.Context, accessToken, status string) ([]domain.OperationalCase, error) {
	path := "/v1/operations/cases"
	if status = strings.TrimSpace(status); status != "" {
		path += "?status=" + url.QueryEscape(status)
	}
	var payload struct {
		Data []domain.OperationalCase `json:"data"`
	}
	err := c.do(ctx, http.MethodGet, path, accessToken, nil, &payload)
	return payload.Data, err
}

func (c *Client) CreateOperationalCase(ctx context.Context, accessToken string, input domain.CreateOperationalCase) (domain.OperationalCase, error) {
	var item domain.OperationalCase
	err := c.do(ctx, http.MethodPost, "/v1/operations/cases", accessToken, input, &item)
	return item, err
}

func (c *Client) GetOperationalCase(ctx context.Context, accessToken, caseID string) (domain.OperationalCaseDetail, error) {
	var item domain.OperationalCaseDetail
	err := c.do(ctx, http.MethodGet, "/v1/operations/cases/"+url.PathEscape(caseID), accessToken, nil, &item)
	return item, err
}

func (c *Client) UpdateOperationalCase(ctx context.Context, accessToken, caseID string, input domain.UpdateOperationalCase) (domain.OperationalCase, error) {
	var item domain.OperationalCase
	err := c.do(ctx, http.MethodPatch, "/v1/operations/cases/"+url.PathEscape(caseID), accessToken, input, &item)
	return item, err
}

func (c *Client) CreateDecisionRecord(ctx context.Context, accessToken, caseID string, input domain.CreateDecisionRecord) (domain.DecisionRecord, error) {
	var item domain.DecisionRecord
	err := c.do(ctx, http.MethodPost, "/v1/operations/cases/"+url.PathEscape(caseID)+"/decisions", accessToken, input, &item)
	return item, err
}

func (c *Client) RecordDecisionAction(ctx context.Context, accessToken, caseID, decisionID string, input domain.RecordDecisionAction) (domain.DecisionRecord, error) {
	var item domain.DecisionRecord
	err := c.do(ctx, http.MethodPost, "/v1/operations/cases/"+url.PathEscape(caseID)+"/decisions/"+url.PathEscape(decisionID)+"/action", accessToken, input, &item)
	return item, err
}

func (c *Client) RecordDecisionOutcome(ctx context.Context, accessToken, caseID, decisionID string, input domain.RecordDecisionOutcome) (domain.DecisionRecord, error) {
	var item domain.DecisionRecord
	err := c.do(ctx, http.MethodPost, "/v1/operations/cases/"+url.PathEscape(caseID)+"/decisions/"+url.PathEscape(decisionID)+"/outcome", accessToken, input, &item)
	return item, err
}

func (c *Client) DecisionTrustMetrics(ctx context.Context, accessToken string) (domain.DecisionTrustMetrics, error) {
	var metrics domain.DecisionTrustMetrics
	err := c.do(ctx, http.MethodGet, "/v1/operations/trust", accessToken, nil, &metrics)
	return metrics, err
}

func (c *Client) CreateTrustShare(ctx context.Context, accessToken string, input domain.CreateTrustShare) (domain.TrustShareLink, error) {
	var link domain.TrustShareLink
	err := c.do(ctx, http.MethodPost, "/v1/operations/trust/share", accessToken, input, &link)
	return link, err
}

func (c *Client) TrustShares(ctx context.Context, accessToken string) ([]domain.TrustShareLink, error) {
	var payload struct {
		Data []domain.TrustShareLink `json:"data"`
	}
	err := c.do(ctx, http.MethodGet, "/v1/operations/trust/shares", accessToken, nil, &payload)
	return payload.Data, err
}

func (c *Client) RevokeTrustShare(ctx context.Context, accessToken, token string) error {
	return c.do(ctx, http.MethodDelete, "/v1/trust-shares/"+url.PathEscape(token), accessToken, nil, nil)
}

func (c *Client) PublicTrustReport(ctx context.Context, token string) (domain.PublicTrustReport, error) {
	var report domain.PublicTrustReport
	err := c.do(ctx, http.MethodGet, "/public/trust-shares/"+url.PathEscape(token), "", nil, &report)
	return report, err
}

func (c *Client) ListWebhookIntegrations(ctx context.Context, accessToken string) ([]domain.WebhookIntegration, error) {
	var payload struct {
		Data []domain.WebhookIntegration `json:"data"`
	}
	err := c.do(ctx, http.MethodGet, "/v1/integrations/webhooks", accessToken, nil, &payload)
	return payload.Data, err
}

func (c *Client) CreateWebhookIntegration(ctx context.Context, accessToken string, input domain.CreateWebhookIntegration) (domain.WebhookIntegrationCreated, error) {
	var created domain.WebhookIntegrationCreated
	err := c.do(ctx, http.MethodPost, "/v1/integrations/webhooks", accessToken, input, &created)
	return created, err
}

func (c *Client) DeleteWebhookIntegration(ctx context.Context, accessToken, integrationID string) error {
	return c.do(ctx, http.MethodDelete, "/v1/integrations/webhooks/"+url.PathEscape(integrationID), accessToken, nil, nil)
}

func (c *Client) TestWebhookIntegration(ctx context.Context, accessToken, integrationID string) (domain.WebhookDelivery, error) {
	var delivery domain.WebhookDelivery
	err := c.do(ctx, http.MethodPost, "/v1/integrations/webhooks/"+url.PathEscape(integrationID)+"/test", accessToken, nil, &delivery)
	return delivery, err
}

func (c *Client) do(ctx context.Context, method, path, accessToken string, body, result any) error {
	var reader io.Reader
	if body != nil {
		payload, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("encode request: %w", err)
		}
		reader = bytes.NewReader(payload)
	}
	request, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, reader)
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	request.Header.Set("Authorization", "Bearer "+accessToken)
	request.Header.Set("Accept", "application/json")
	if body != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	if c.clientName != "" {
		request.Header.Set(domain.ClientHeaderName, c.clientName)
	}

	response, err := c.http.Do(request)
	if err != nil {
		return fmt.Errorf("call skyvisor-api: %w", err)
	}
	defer func() {
		_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, 1<<16))
		_ = response.Body.Close()
	}()

	switch {
	case response.StatusCode == http.StatusUnauthorized:
		return ErrUnauthorized
	case response.StatusCode == http.StatusNotFound:
		return ErrNotFound
	case response.StatusCode == http.StatusPaymentRequired:
		return fmt.Errorf("%w: %v", ErrPaymentRequired, apiError(response))
	case response.StatusCode == http.StatusConflict:
		return fmt.Errorf("%w: %v", ErrConflict, apiError(response))
	case response.StatusCode >= 400:
		return apiError(response)
	}
	if result == nil || response.StatusCode == http.StatusNoContent {
		return nil
	}
	if err := json.NewDecoder(io.LimitReader(response.Body, 1<<20)).Decode(result); err != nil {
		return fmt.Errorf("decode skyvisor-api response: %w", err)
	}
	return nil
}

func apiError(response *http.Response) error {
	var payload struct {
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.NewDecoder(io.LimitReader(response.Body, 1<<16)).Decode(&payload); err == nil && payload.Error.Message != "" {
		return &APIError{Status: response.StatusCode, Code: payload.Error.Code, Message: payload.Error.Message}
	}
	return fmt.Errorf("skyvisor-api returned status %d", response.StatusCode)
}
