package schema

// AddReferenceRequest represents a request to add a new voice reference.
type AddReferenceRequest struct {
	ID    string `json:"id" msgpack:"id"`
	Audio []byte `json:"audio" msgpack:"audio"`
	Text  string `json:"text" msgpack:"text"`
}

// AddReferenceResponse represents the response after adding a voice reference.
type AddReferenceResponse struct {
	Success     bool   `json:"success" msgpack:"success"`
	Message     string `json:"message" msgpack:"message"`
	ReferenceID string `json:"reference_id" msgpack:"reference_id"`
}

// ListReferencesResponse represents the response for listing voice references.
type ListReferencesResponse struct {
	Success      bool     `json:"success" msgpack:"success"`
	ReferenceIDs []string `json:"reference_ids" msgpack:"reference_ids"`
	Message      string   `json:"message" msgpack:"message"`
}

// DeleteReferenceResponse represents the response when deleting a voice reference.
type DeleteReferenceResponse struct {
	Success     bool   `json:"success" msgpack:"success"`
	Message     string `json:"message" msgpack:"message"`
	ReferenceID string `json:"reference_id" msgpack:"reference_id"`
}
