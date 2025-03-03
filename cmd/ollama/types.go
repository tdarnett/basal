package ollama

// Request represents a request to the Ollama API
type Request struct {
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
	Stream   bool      `json:"stream"`
}

// Message represents a single message in the conversation
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// Response represents a response from the Ollama API
type Response struct {
	Message Message `json:"message"`
}
