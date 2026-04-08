package handlers

// IndexResponse is returned by GET / (API root).
type IndexResponse struct {
	Service string `json:"service"`
	Version string `json:"version"`
	Docs    string `json:"swagger"`
}

// HealthResponse is returned by GET /health.
type HealthResponse struct {
	Status string `json:"status"`
}

// PingResponse is returned by GET /api/v1/ping.
type PingResponse struct {
	Message string `json:"message"`
}

// MeResponse is returned by GET /api/v1/me.
type MeResponse struct {
	UserID string   `json:"user_id"`
	Roles  []string `json:"roles"`
}

// ErrorResponse matches pkg/response JSON error envelope.
type ErrorResponse struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}
