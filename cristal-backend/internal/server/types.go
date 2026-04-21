package server

// ChatRequest represents a chat message from the user
type ChatRequest struct {
	Message string `json:"message"`
}

// Citation represents a cited page from the portal
type Citation struct {
	ID         int    `json:"id"`
	Title      string `json:"title"`
	Breadcrumb string `json:"breadcrumb"`
	URL        string `json:"url"`
}

// ChatResponse represents the API response
// Compatible with frontend expectations (cristal-chat-ui)
type ChatResponse struct {
	Response  string     `json:"response,omitempty"`  // Text with inline citations [text]^N
	Citations []Citation `json:"citations,omitempty"` // Array of cited pages
	Status    string     `json:"status,omitempty"`    // Deprecated: kept for backward compatibility
	Error     string     `json:"error,omitempty"`     // Error message if any
}

// HealthResponse represents health check response
type HealthResponse struct {
	Status string `json:"status"`
}
