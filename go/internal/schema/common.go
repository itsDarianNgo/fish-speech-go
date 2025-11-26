package schema

// ErrorResponse represents a standard error payload.
type ErrorResponse struct {
	Detail string `json:"detail" msgpack:"detail"`
}

// HealthResponse represents the health check response payload.
type HealthResponse struct {
	Status string `json:"status" msgpack:"status"`
}
