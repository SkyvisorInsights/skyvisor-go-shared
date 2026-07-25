package domain

import "time"

// FlightLive is the most recent airborne position for an active flight.
type FlightLive struct {
	Latitude  float64   `json:"latitude"`
	Longitude float64   `json:"longitude"`
	Altitude  float64   `json:"altitude,omitempty"`
	Speed     float64   `json:"speed,omitempty"`
	IsGround  bool      `json:"is_ground"`
	UpdatedAt time.Time `json:"updated_at,omitzero"`
}

// InboundAircraft is the metal currently assigned to operate a departure —
// typically still inbound to the origin airport ("where's my plane").
type InboundAircraft struct {
	Registration        string    `json:"registration,omitempty"`
	FlightNumber        string    `json:"flight_number,omitempty"`
	Status              string    `json:"status,omitempty"`
	OriginIATA          string    `json:"origin_iata,omitempty"`
	DestinationIATA     string    `json:"destination_iata,omitempty"`
	EstimatedArrival    time.Time `json:"estimated_arrival,omitzero"`
	ArrivalDelayMinutes *int      `json:"arrival_delay_minutes,omitempty"`
	MinutesUntilArrival *int      `json:"minutes_until_arrival,omitempty"`
	LateRisk            bool      `json:"late_risk"`
	Message             string    `json:"message,omitempty"`
}

// Flight is the rich provider-backed flight record used by track, watches,
// and trip enrichment.
type Flight struct {
	InstanceID            string            `json:"instance_id,omitempty"`
	Number                string            `json:"number"`
	Status                string            `json:"status"`
	Airline               string            `json:"airline"`
	DepartureIATA         string            `json:"departure_iata"`
	ArrivalIATA           string            `json:"arrival_iata"`
	DepartureTerminal     string            `json:"departure_terminal,omitempty"`
	DepartureGate         string            `json:"departure_gate,omitempty"`
	DepartureDelayMinutes *int              `json:"departure_delay_minutes,omitempty"`
	ArrivalTerminal       string            `json:"arrival_terminal,omitempty"`
	ArrivalGate           string            `json:"arrival_gate,omitempty"`
	ArrivalBaggage        string            `json:"arrival_baggage,omitempty"`
	ArrivalDelayMinutes   *int              `json:"arrival_delay_minutes,omitempty"`
	ScheduledAt           time.Time         `json:"scheduled_at,omitzero"`
	EstimatedDeparture    time.Time         `json:"estimated_departure,omitzero"`
	EstimatedArrival      time.Time         `json:"estimated_arrival,omitzero"`
	ActualDeparture       time.Time         `json:"actual_departure,omitzero"`
	ActualArrival         time.Time         `json:"actual_arrival,omitzero"`
	AircraftRegistration  string            `json:"aircraft_registration,omitempty"`
	AircraftIATA          string            `json:"aircraft_iata,omitempty"`
	Live                  *FlightLive       `json:"live,omitempty"`
	Inbound               *InboundAircraft  `json:"inbound,omitempty"`
	Provenance            *SourceProvenance `json:"provenance,omitempty"`
	Freshness             *DataFreshness    `json:"freshness,omitempty"`
	UpdatedAt             time.Time         `json:"updated_at"`
}

// SourceProvenance identifies the product and observation behind a displayed
// fact. Permission flags are intentionally omitted from customer responses.
type SourceProvenance struct {
	Provider   string    `json:"provider"`
	Product    string    `json:"product,omitempty"`
	ObservedAt time.Time `json:"observed_at,omitzero"`
	IngestedAt time.Time `json:"ingested_at,omitzero"`
}

// DataFreshness makes stale or incomplete provider data explicit to clients.
type DataFreshness struct {
	Status     string    `json:"status"` // fresh | stale | unknown
	ObservedAt time.Time `json:"observed_at,omitzero"`
	AgeSeconds int       `json:"age_seconds,omitempty"`
	StaleAfter int       `json:"stale_after_seconds"`
}

// ShareLink is a public, revocable pickup page for a watched flight.
type ShareLink struct {
	Token     string    `json:"token"`
	WatchID   string    `json:"watch_id"`
	URLPath   string    `json:"url_path"`
	ExpiresAt time.Time `json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
}

// PublicShare is the unauthenticated view of a shared flight.
type PublicShare struct {
	FlightNumber string    `json:"flight_number"`
	Flight       *Flight   `json:"flight,omitempty"`
	ExpiresAt    time.Time `json:"expires_at"`
	Label        string    `json:"label,omitempty"`
}

// PushDevice is a mobile endpoint for APNs/FCM delivery.
type PushDevice struct {
	ID        string    `json:"id"`
	Platform  string    `json:"platform"`
	Token     string    `json:"token"`
	CreatedAt time.Time `json:"created_at"`
}

type RegisterDevice struct {
	Platform string `json:"platform"`
	Token    string `json:"token"`
}

const (
	RiskOK         = "ok"
	RiskTight      = "tight"
	RiskMissLikely = "miss_likely"
	RiskUnknown    = "unknown"
)

// ConnectionRisk describes the transfer between two consecutive segments.
type ConnectionRisk struct {
	FromFlight     string `json:"from_flight"`
	ToFlight       string `json:"to_flight"`
	FromIATA       string `json:"from_iata,omitempty"`
	ToIATA         string `json:"to_iata,omitempty"`
	LayoverMinutes int    `json:"layover_minutes,omitempty"`
	Risk           string `json:"risk"`
	Reason         string `json:"reason"`
}

// TripSegment is one leg of a trip: a flight plus the timeline details the
// traveler cares about. Only the flight number is mandatory. Live is filled
// on read when enrichment is available and is never persisted.
type TripSegment struct {
	FlightNumber     string    `json:"flight_number"`
	DepartureIATA    string    `json:"departure_iata,omitempty"`
	ArrivalIATA      string    `json:"arrival_iata,omitempty"`
	DepartsAt        time.Time `json:"departs_at,omitzero"`
	ArrivesAt        time.Time `json:"arrives_at,omitzero"`
	BookingReference string    `json:"booking_reference,omitempty"`
	Notes            string    `json:"notes,omitempty"`
	Live             *Flight   `json:"live,omitempty"`
	ConnectionRisk   string    `json:"connection_risk,omitempty"`
}

type Trip struct {
	ID          string           `json:"id"`
	Name        string           `json:"name"`
	Segments    []TripSegment    `json:"segments"`
	StartsAt    time.Time        `json:"starts_at,omitzero"`
	CreatedAt   time.Time        `json:"created_at"`
	Connections []ConnectionRisk `json:"connections,omitempty"`
	AutoWatches *AutoWatchResult `json:"auto_watches,omitempty"`
}

// AutoWatchResult reports watches started when a trip was created or imported.
type AutoWatchResult struct {
	Started []string           `json:"started"`
	Skipped []AutoWatchSkipped `json:"skipped,omitempty"`
}

type AutoWatchSkipped struct {
	FlightNumber string `json:"flight_number"`
	Reason       string `json:"reason"`
}

type CreateTrip struct {
	Name     string        `json:"name"`
	Segments []TripSegment `json:"segments"`
	StartsAt time.Time     `json:"starts_at,omitzero"`
}

type AssistantRequest struct {
	Question string `json:"question"`
	TripID   string `json:"trip_id,omitempty"`
}

type ImportTripRequest struct {
	Text string `json:"text"`
}

type AssistantResponse struct {
	Answer   string `json:"answer"`
	Model    string `json:"model"`
	TripID   string `json:"trip_id,omitempty"`
	Grounded bool   `json:"grounded"`
}

// Plan identifies the commercial tier of an account.
type Plan string

const (
	PlanFree     Plan = "free"
	PlanPro      Plan = "pro"
	PlanBusiness Plan = "business"
)

// FreeWatchLimit is concurrent active watches allowed on the free plan.
const FreeWatchLimit = 5

// FreeAnalyticsWindowDays is the max lookback Free plan may request.
const FreeAnalyticsWindowDays = 7

// ProAnalyticsWindowDays is the max lookback Pro/Business may request.
const ProAnalyticsWindowDays = 90

// BusinessTeamSeatLimit is max members (including owner) on a Business team.
const BusinessTeamSeatLimit = 5

// Daily usage limits (-1 = unlimited). UTC calendar day window.
const (
	FreeMCPDailyReadLimit       = 100
	FreeMCPDailyActionLimit     = 0 // Free MCP is read-only
	FreeAssistantDailyLimit     = 10
	ProMCPDailyReadLimit        = 2000
	ProMCPDailyActionLimit      = 200
	ProAssistantDailyLimit      = 100
	BusinessMCPDailyReadLimit   = 10000
	BusinessMCPDailyActionLimit = 2000
	BusinessAssistantDailyLimit = 500
)

// Usage kinds for metering.
const (
	UsageKindMCPRead   = "mcp_read"
	UsageKindMCPAction = "mcp_action"
	UsageKindAssistant = "assistant"
)

// ClientHeaderMCP is sent by skyvisor-mcp so the API can meter agent traffic.
const ClientHeaderMCP = "mcp"
const ClientHeaderName = "X-SkyVisor-Client"

// IsPaidPlan reports whether the plan unlocks Pro-level watches/analytics.
func IsPaidPlan(plan Plan) bool {
	return plan == PlanPro || plan == PlanBusiness
}

// Entitlements are the product gates for an account.
type Entitlements struct {
	Plan                Plan `json:"plan"`
	WatchLimit          int  `json:"watch_limit"` // -1 means unlimited
	ActiveWatches       int  `json:"active_watches"`
	CanCreateWatch      bool `json:"can_create_watch"`
	ProTrialEligible    bool `json:"pro_trial_eligible"`
	EmailAlerts         bool `json:"email_alerts"`
	AnalyticsWindowDays int  `json:"analytics_window_days"`
	CanExportAnalytics  bool `json:"can_export_analytics"`
	CanAccessLogistics  bool `json:"can_access_logistics"`
	CanTeamShare        bool `json:"can_team_share"`
	TeamSeatLimit       int  `json:"team_seat_limit,omitempty"`
	MCPDailyReadLimit   int  `json:"mcp_daily_read_limit"`   // -1 unlimited
	MCPDailyActionLimit int  `json:"mcp_daily_action_limit"` // -1 unlimited; 0 = no actions
	AssistantDailyLimit int  `json:"assistant_daily_limit"`  // -1 unlimited
}

// UsageSnapshot is the caller's UTC-day counters vs plan limits.
type UsageSnapshot struct {
	Day                 string       `json:"day"` // YYYY-MM-DD UTC
	MCPReads            int          `json:"mcp_reads"`
	MCPActions          int          `json:"mcp_actions"`
	AssistantCalls      int          `json:"assistant_calls"`
	MCPDailyReadLimit   int          `json:"mcp_daily_read_limit"`
	MCPDailyActionLimit int          `json:"mcp_daily_action_limit"`
	AssistantDailyLimit int          `json:"assistant_daily_limit"`
	Entitlements        Entitlements `json:"entitlements"`
}

// Watch is a continuously monitored flight for one account.
type Watch struct {
	ID            string     `json:"id"`
	FlightNumber  string     `json:"flight_number"`
	TripID        string     `json:"trip_id,omitempty"`
	TeamID        string     `json:"team_id,omitempty"`
	Status        string     `json:"status"`
	Flight        *Flight    `json:"flight,omitempty"`
	LastEventType string     `json:"last_event_type,omitempty"`
	LastEventAt   time.Time  `json:"last_event_at,omitzero"`
	Share         *ShareLink `json:"share,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

type CreateWatch struct {
	FlightNumber  string `json:"flight_number"`
	TripID        string `json:"trip_id,omitempty"`
	ShareWithTeam bool   `json:"share_with_team,omitempty"`
	TeamID        string `json:"-"` // set by API when ShareWithTeam is true
}

// Preferences updates notification settings for an account.
type Preferences struct {
	EmailAlerts *bool `json:"email_alerts,omitempty"`
}

// StreamEvent is pushed on the authenticated SSE channel.
type StreamEvent struct {
	Type      string    `json:"type"`
	AccountID string    `json:"-"`
	WatchID   string    `json:"watch_id,omitempty"`
	Flight    *Flight   `json:"flight,omitempty"`
	Message   string    `json:"message,omitempty"`
	At        time.Time `json:"at"`
}

// AirportBoard is arrivals and/or departures for one IATA airport.
type AirportBoard struct {
	IATA        string    `json:"iata"`
	Arrivals    []Flight  `json:"arrivals"`
	Departures  []Flight  `json:"departures"`
	GeneratedAt time.Time `json:"generated_at"`
}

// AnalyticsQuery is the filter set for performance analytics.
type AnalyticsQuery struct {
	Airline    string `json:"airline,omitempty"`
	Airport    string `json:"airport,omitempty"`
	WindowDays int    `json:"window_days,omitempty"`
	Limit      int    `json:"limit,omitempty"`
}

// AnalyticsSummary rolls up delay / on-time performance.
type AnalyticsSummary struct {
	FlightCount              int     `json:"flight_count"`
	OnTimeCount              int     `json:"on_time_count"`
	DelayedCount             int     `json:"delayed_count"`
	CancelledCount           int     `json:"cancelled_count"`
	OnTimePercent            float64 `json:"on_time_percent"`
	AvgDepartureDelayMinutes float64 `json:"avg_departure_delay_minutes"`
	AvgArrivalDelayMinutes   float64 `json:"avg_arrival_delay_minutes"`
}

// RouteStat ranks a departure→arrival pair by volume.
type RouteStat struct {
	DepartureIATA   string  `json:"departure_iata"`
	ArrivalIATA     string  `json:"arrival_iata"`
	Flights         int     `json:"flights"`
	AvgDelayMinutes float64 `json:"avg_delay_minutes"`
	OnTimePercent   float64 `json:"on_time_percent"`
}

// AirportStat ranks airports by traffic in the sample.
type AirportStat struct {
	IATA            string  `json:"iata"`
	Departures      int     `json:"departures"`
	Arrivals        int     `json:"arrivals"`
	AvgDelayMinutes float64 `json:"avg_delay_minutes"`
}

// AnalyticsReport is the JSON analytics payload for web, API, and MCP.
type AnalyticsReport struct {
	WindowDays   int              `json:"window_days"`
	WindowCapped bool             `json:"window_capped"`
	Plan         Plan             `json:"plan"`
	CanExport    bool             `json:"can_export"`
	Airline      string           `json:"airline,omitempty"`
	Airport      string           `json:"airport,omitempty"`
	Summary      AnalyticsSummary `json:"summary"`
	Routes       []RouteStat      `json:"routes"`
	Airports     []AirportStat    `json:"airports"`
	SampleSize   int              `json:"sample_size"`
	GeneratedAt  time.Time        `json:"generated_at"`
}

// WhatIfRequest applies a delay scenario to one trip segment, then re-scores
// connections. Optional AskAssistant asks the grounded co-pilot about the result.
type WhatIfRequest struct {
	SegmentIndex          int    `json:"segment_index"`
	DelayDepartureMinutes int    `json:"delay_departure_minutes,omitempty"`
	DelayArrivalMinutes   int    `json:"delay_arrival_minutes,omitempty"`
	Question              string `json:"question,omitempty"`
}

// WhatIfResult is the scored trip after applying a scenario (not persisted).
type WhatIfResult struct {
	Trip        Trip             `json:"trip"`
	Baseline    []ConnectionRisk `json:"baseline_connections,omitempty"`
	Scenario    WhatIfRequest    `json:"scenario"`
	Summary     string           `json:"summary"`
	Assistant   string           `json:"assistant_answer,omitempty"`
	Model       string           `json:"model,omitempty"`
	GeneratedAt time.Time        `json:"generated_at"`
}

// LogisticsQuery filters the Business cargo / disruption overview.
type LogisticsQuery struct {
	Airline string `json:"airline,omitempty"`
	Airport string `json:"airport,omitempty"`
	Limit   int    `json:"limit,omitempty"`
}

// LogisticsDisruption is one delayed/cancelled cargo-relevant flight.
type LogisticsDisruption struct {
	Flight  Flight `json:"flight"`
	Kind    string `json:"kind"` // delayed | cancelled | diverted | late_inbound
	Message string `json:"message"`
}

// LogisticsOverview is the Business logistics workspace snapshot.
type LogisticsOverview struct {
	CargoFlights []Flight              `json:"cargo_flights"`
	Disruptions  []LogisticsDisruption `json:"disruptions"`
	Airline      string                `json:"airline,omitempty"`
	Airport      string                `json:"airport,omitempty"`
	SampleSize   int                   `json:"sample_size"`
	GeneratedAt  time.Time             `json:"generated_at"`
}

// Team is a Business workspace for shared watches (up to BusinessTeamSeatLimit seats).
type Team struct {
	ID             string    `json:"id"`
	Name           string    `json:"name"`
	OwnerAccountID string    `json:"owner_account_id"`
	InviteCode     string    `json:"invite_code"`
	MemberCount    int       `json:"member_count"`
	SeatLimit      int       `json:"seat_limit"`
	CreatedAt      time.Time `json:"created_at"`
}

type CreateTeam struct {
	Name string `json:"name"`
}

type JoinTeam struct {
	InviteCode string `json:"invite_code"`
}

// OperationsDashboard is the decision-focused signed-in landing payload. It
// only reports values derived from records owned by the authenticated account.
type OperationsDashboard struct {
	Summary     OperationsSummary      `json:"summary"`
	Attention   []OperationsAttention  `json:"attention"`
	Watches     []OperationsWatch      `json:"watches"`
	Connections []OperationsConnection `json:"connections"`
	Freshness   OperationsFreshness    `json:"freshness"`
	GeneratedAt time.Time              `json:"generated_at"`
}

type OperationsSummary struct {
	ActiveWatches      int `json:"active_watches"`
	FlightsAtRisk      int `json:"flights_at_risk"`
	CancelledFlights   int `json:"cancelled_flights"`
	LateInboundFlights int `json:"late_inbound_flights"`
	Trips              int `json:"trips"`
	AtRiskConnections  int `json:"at_risk_connections"`
}

type OperationsAttention struct {
	ID           string    `json:"id"`
	Kind         string    `json:"kind"`
	Severity     string    `json:"severity"`
	Title        string    `json:"title"`
	FlightNumber string    `json:"flight_number,omitempty"`
	Route        string    `json:"route,omitempty"`
	Status       string    `json:"status,omitempty"`
	Reason       string    `json:"reason"`
	UpdatedAt    time.Time `json:"updated_at,omitzero"`
}

type OperationsWatch struct {
	WatchID               string    `json:"watch_id"`
	FlightNumber          string    `json:"flight_number"`
	Route                 string    `json:"route,omitempty"`
	Status                string    `json:"status,omitempty"`
	DepartureGate         string    `json:"departure_gate,omitempty"`
	ArrivalGate           string    `json:"arrival_gate,omitempty"`
	DepartureDelayMinutes *int      `json:"departure_delay_minutes,omitempty"`
	ArrivalDelayMinutes   *int      `json:"arrival_delay_minutes,omitempty"`
	InboundLateRisk       bool      `json:"inbound_late_risk"`
	FreshnessStatus       string    `json:"freshness_status"`
	UpdatedAt             time.Time `json:"updated_at,omitzero"`
}

type OperationsConnection struct {
	TripID         string `json:"trip_id"`
	TripName       string `json:"trip_name"`
	FromFlight     string `json:"from_flight"`
	ToFlight       string `json:"to_flight"`
	Airport        string `json:"airport,omitempty"`
	LayoverMinutes int    `json:"layover_minutes,omitempty"`
	Risk           string `json:"risk"`
	Reason         string `json:"reason"`
}

type OperationsFreshness struct {
	Provider            string    `json:"provider"`
	Status              string    `json:"status"`
	FreshRecords        int       `json:"fresh_records"`
	StaleRecords        int       `json:"stale_records"`
	UnknownRecords      int       `json:"unknown_records"`
	LatestObservationAt time.Time `json:"latest_observation_at,omitzero"`
	OldestObservationAt time.Time `json:"oldest_observation_at,omitzero"`
	StaleAfterSeconds   int       `json:"stale_after_seconds"`
}

// OperationalCase binds a shipment, passenger journey, or other customer
// dependency to flights, deadlines, financial exposure, owners, and actions.
type OperationalCase struct {
	ID                 string                   `json:"id"`
	Kind               string                   `json:"kind"` // shipment | passenger | crew | aircraft | other
	Reference          string                   `json:"reference"`
	Title              string                   `json:"title"`
	Description        string                   `json:"description,omitempty"`
	TripID             string                   `json:"trip_id,omitempty"`
	FlightNumbers      []string                 `json:"flight_numbers"`
	Status             string                   `json:"status"`   // open | monitoring | action_required | resolved | closed
	Severity           string                   `json:"severity"` // low | medium | high | critical
	Owner              string                   `json:"owner,omitempty"`
	EscalationPolicy   string                   `json:"escalation_policy,omitempty"`
	HandoffDeadline    *time.Time               `json:"handoff_deadline,omitempty"`
	SLADeadline        *time.Time               `json:"sla_deadline,omitempty"`
	SLACurrency        string                   `json:"sla_currency,omitempty"`
	SLAValueMinor      int64                    `json:"sla_value_minor,omitempty"`
	FailureCostMinor   int64                    `json:"failure_cost_minor,omitempty"`
	ApprovedAlternates []OperationalAlternative `json:"approved_alternatives"`
	CreatedAt          time.Time                `json:"created_at"`
	UpdatedAt          time.Time                `json:"updated_at"`
	ClosedAt           *time.Time               `json:"closed_at,omitempty"`
}

type OperationalAlternative struct {
	ID       string `json:"id"`
	Kind     string `json:"kind"`
	Label    string `json:"label"`
	Details  string `json:"details,omitempty"`
	Approved bool   `json:"approved"`
}

type CreateOperationalCase struct {
	Kind               string                   `json:"kind"`
	Reference          string                   `json:"reference"`
	Title              string                   `json:"title"`
	Description        string                   `json:"description,omitempty"`
	TripID             string                   `json:"trip_id,omitempty"`
	FlightNumbers      []string                 `json:"flight_numbers,omitempty"`
	Severity           string                   `json:"severity,omitempty"`
	Owner              string                   `json:"owner,omitempty"`
	EscalationPolicy   string                   `json:"escalation_policy,omitempty"`
	HandoffDeadline    *time.Time               `json:"handoff_deadline,omitempty"`
	SLADeadline        *time.Time               `json:"sla_deadline,omitempty"`
	SLACurrency        string                   `json:"sla_currency,omitempty"`
	SLAValueMinor      int64                    `json:"sla_value_minor,omitempty"`
	FailureCostMinor   int64                    `json:"failure_cost_minor,omitempty"`
	ApprovedAlternates []OperationalAlternative `json:"approved_alternatives,omitempty"`
}

type UpdateOperationalCase struct {
	Status             *string                   `json:"status,omitempty"`
	Severity           *string                   `json:"severity,omitempty"`
	Owner              *string                   `json:"owner,omitempty"`
	EscalationPolicy   *string                   `json:"escalation_policy,omitempty"`
	HandoffDeadline    *time.Time                `json:"handoff_deadline,omitempty"`
	SLADeadline        *time.Time                `json:"sla_deadline,omitempty"`
	ApprovedAlternates *[]OperationalAlternative `json:"approved_alternatives,omitempty"`
}

// DecisionRecord is the durable signal → prediction → recommendation → action
// → outcome chain. Every transition is also appended to the case audit log.
type DecisionRecord struct {
	ID                   string           `json:"id"`
	CaseID               string           `json:"case_id"`
	SignalType           string           `json:"signal_type"`
	SignalSource         string           `json:"signal_source"`
	SignalObservedAt     time.Time        `json:"signal_observed_at"`
	Evidence             map[string]any   `json:"evidence,omitempty"`
	PredictionType       string           `json:"prediction_type"`
	PredictedProbability *float64         `json:"predicted_probability,omitempty"`
	Confidence           *float64         `json:"confidence,omitempty"`
	HorizonMinutes       int              `json:"horizon_minutes,omitempty"`
	ModelVersion         string           `json:"model_version,omitempty"`
	ScopeAirport         string           `json:"scope_airport,omitempty"`
	ScopeAirline         string           `json:"scope_airline,omitempty"`
	ScopeRoute           string           `json:"scope_route,omitempty"`
	RecommendationType   string           `json:"recommendation_type"`
	Recommendation       string           `json:"recommendation"`
	Rationale            string           `json:"rationale,omitempty"`
	ProposedAction       string           `json:"proposed_action,omitempty"`
	ApprovalRequired     bool             `json:"approval_required"`
	State                string           `json:"state"` // proposed | approved | rejected | executed | evaluated
	Action               *DecisionAction  `json:"action,omitempty"`
	Outcome              *DecisionOutcome `json:"outcome,omitempty"`
	CreatedAt            time.Time        `json:"created_at"`
	UpdatedAt            time.Time        `json:"updated_at"`
}

type CreateDecisionRecord struct {
	SignalType           string         `json:"signal_type"`
	SignalSource         string         `json:"signal_source"`
	SignalObservedAt     time.Time      `json:"signal_observed_at"`
	Evidence             map[string]any `json:"evidence,omitempty"`
	PredictionType       string         `json:"prediction_type"`
	PredictedProbability *float64       `json:"predicted_probability,omitempty"`
	Confidence           *float64       `json:"confidence,omitempty"`
	HorizonMinutes       int            `json:"horizon_minutes,omitempty"`
	ModelVersion         string         `json:"model_version,omitempty"`
	ScopeAirport         string         `json:"scope_airport,omitempty"`
	ScopeAirline         string         `json:"scope_airline,omitempty"`
	ScopeRoute           string         `json:"scope_route,omitempty"`
	RecommendationType   string         `json:"recommendation_type"`
	Recommendation       string         `json:"recommendation"`
	Rationale            string         `json:"rationale,omitempty"`
	ProposedAction       string         `json:"proposed_action,omitempty"`
	ApprovalRequired     bool           `json:"approval_required"`
}

type DecisionAction struct {
	Decision    string    `json:"decision"` // approved | rejected | executed
	ActionTaken string    `json:"action_taken,omitempty"`
	ActorID     string    `json:"actor_id"`
	ActorType   string    `json:"actor_type"` // human | agent
	Notes       string    `json:"notes,omitempty"`
	ExternalRef string    `json:"external_ref,omitempty"`
	OccurredAt  time.Time `json:"occurred_at"`
}

type RecordDecisionAction struct {
	Decision    string `json:"decision"`
	ActionTaken string `json:"action_taken,omitempty"`
	Notes       string `json:"notes,omitempty"`
	ExternalRef string `json:"external_ref,omitempty"`
}

type DecisionOutcome struct {
	PredictionResult string         `json:"prediction_result"` // occurred | not_occurred | unknown
	ActionResult     string         `json:"action_result"`     // succeeded | failed | no_action | unknown
	Actual           map[string]any `json:"actual,omitempty"`
	AvoidedCostMinor int64          `json:"avoided_cost_minor,omitempty"`
	Notes            string         `json:"notes,omitempty"`
	OccurredAt       time.Time      `json:"occurred_at"`
}

type RecordDecisionOutcome struct {
	PredictionResult string         `json:"prediction_result"`
	ActionResult     string         `json:"action_result"`
	Actual           map[string]any `json:"actual,omitempty"`
	AvoidedCostMinor int64          `json:"avoided_cost_minor,omitempty"`
	Notes            string         `json:"notes,omitempty"`
}

type CaseAuditEvent struct {
	ID        string         `json:"id"`
	CaseID    string         `json:"case_id"`
	Type      string         `json:"type"`
	ActorID   string         `json:"actor_id,omitempty"`
	ActorType string         `json:"actor_type,omitempty"`
	Payload   map[string]any `json:"payload,omitempty"`
	CreatedAt time.Time      `json:"created_at"`
}

type OperationalCaseDetail struct {
	Case      OperationalCase  `json:"case"`
	Decisions []DecisionRecord `json:"decisions"`
	Audit     []CaseAuditEvent `json:"audit"`
}

type DecisionTrustMetrics struct {
	TotalPredictions      int              `json:"total_predictions"`
	EvaluatedPredictions  int              `json:"evaluated_predictions"`
	SuccessfulPredictions int              `json:"successful_predictions"`
	FalsePositives        int              `json:"false_positives"`
	Precision             float64          `json:"precision"`
	FalsePositiveRate     float64          `json:"false_positive_rate"`
	MedianLeadTimeMinutes float64          `json:"median_lead_time_minutes"`
	ExecutedActions       int              `json:"executed_actions"`
	SuccessfulActions     int              `json:"successful_actions"`
	ActionSuccessRate     float64          `json:"action_success_rate"`
	AvoidedCostMinor      int64            `json:"avoided_cost_minor"`
	Currency              string           `json:"currency,omitempty"`
	Breakdowns            []TrustBreakdown `json:"breakdowns"`
	GeneratedAt           time.Time        `json:"generated_at"`
}

type TrustBreakdown struct {
	Dimension         string  `json:"dimension"` // airport | airline | route
	Value             string  `json:"value"`
	Evaluated         int     `json:"evaluated"`
	Precision         float64 `json:"precision"`
	FalsePositiveRate float64 `json:"false_positive_rate"`
}

// Trust share bounds. A trust share always expires.
const (
	TrustShareDefaultTTLHours = 168
	TrustShareMaxTTLHours     = 720
	// TrustLowSampleThreshold is the evaluated-prediction count below which a
	// ratio is presented as low confidence rather than as a headline number.
	TrustLowSampleThreshold = 20
)

// CreateTrustShare is the request body for publishing a trust snapshot.
type CreateTrustShare struct {
	Label             string `json:"label,omitempty"`
	IncludeFinancials bool   `json:"include_financials"`
	TTLHours          int    `json:"ttl_hours,omitempty"`
}

// TrustShareLink is a public, revocable, point-in-time trust report.
type TrustShareLink struct {
	Token             string    `json:"token"`
	Label             string    `json:"label,omitempty"`
	IncludeFinancials bool      `json:"include_financials"`
	URLPath           string    `json:"url_path"`
	GeneratedAt       time.Time `json:"generated_at"`
	ExpiresAt         time.Time `json:"expires_at"`
	CreatedAt         time.Time `json:"created_at"`
}

// PublicTrustReport is the unauthenticated, redacted view of a trust snapshot.
// AvoidedCostMinor is a pointer on purpose: nil means the publisher withheld
// financials, whereas 0 means nothing was avoided. Collapsing the two would
// publish "zero avoided" about an account that merely opted out.
type PublicTrustReport struct {
	Label                 string           `json:"label,omitempty"`
	EvaluatedPredictions  int              `json:"evaluated_predictions"`
	ExecutedActions       int              `json:"executed_actions"`
	Precision             float64          `json:"precision"`
	FalsePositiveRate     float64          `json:"false_positive_rate"`
	MedianLeadTimeMinutes float64          `json:"median_lead_time_minutes"`
	ActionSuccessRate     float64          `json:"action_success_rate"`
	AvoidedCostMinor      *int64           `json:"avoided_cost_minor,omitempty"`
	Currency              string           `json:"currency,omitempty"`
	Breakdowns            []TrustBreakdown `json:"breakdowns,omitempty"`
	GeneratedAt           time.Time        `json:"generated_at"`
	ExpiresAt             time.Time        `json:"expires_at"`
}

type WebhookIntegration struct {
	ID                 string     `json:"id"`
	Name               string     `json:"name"`
	URL                string     `json:"url"`
	Events             []string   `json:"events"`
	Enabled            bool       `json:"enabled"`
	LastDeliveryStatus string     `json:"last_delivery_status,omitempty"`
	LastDeliveryAt     *time.Time `json:"last_delivery_at,omitempty"`
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
}

type CreateWebhookIntegration struct {
	Name   string   `json:"name"`
	URL    string   `json:"url"`
	Events []string `json:"events"`
}

type WebhookIntegrationCreated struct {
	Integration   WebhookIntegration `json:"integration"`
	SigningSecret string             `json:"signing_secret"`
}

type WebhookDelivery struct {
	ID            string    `json:"id"`
	IntegrationID string    `json:"integration_id"`
	EventType     string    `json:"event_type"`
	Status        string    `json:"status"`
	StatusCode    int       `json:"status_code,omitempty"`
	Error         string    `json:"error,omitempty"`
	AttemptedAt   time.Time `json:"attempted_at"`
}

// PAT is the metadata for a long-lived MCP connector token. The plaintext is
// never retrievable after creation.
type PAT struct {
	ID         string     `json:"id"`
	Name       string     `json:"name"`
	CreatedAt  time.Time  `json:"created_at"`
	ExpiresAt  *time.Time `json:"expires_at,omitempty"`
	LastUsedAt *time.Time `json:"last_used_at,omitempty"`
	RevokedAt  *time.Time `json:"revoked_at,omitempty"`
}

// CreatedPAT carries the one-time plaintext next to the stored metadata.
type CreatedPAT struct {
	Token     PAT    `json:"token"`
	Plaintext string `json:"plaintext"`
}
