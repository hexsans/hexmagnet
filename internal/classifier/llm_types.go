package classifier

type TorrentFile struct {
	Path      string `json:"path"`
	Size      uint64 `json:"size"`
	Extension string `json:"extension,omitempty"`
}

type LLMResult struct {
	Type      string   `json:"type"`
	BaseTitle string   `json:"base_title"`
	Date      string   `json:"date,omitempty"`
	Languages []string `json:"languages,omitempty"`
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatRequest struct {
	Model              string              `json:"model"`
	Messages           []chatMessage       `json:"messages"`
	Temperature        float64             `json:"temperature"`
	ReasoningEffort    *string             `json:"reasoning_effort,omitempty"`
	ResponseFormat     *responseFormat     `json:"response_format,omitempty"`
	ReasoningBudget    *int                `json:"reasoning_budget,omitempty"`
	ChatTemplateKwargs *chatTemplateKwargs `json:"chat_template_kwargs,omitempty"`
	Thinking           *thinking           `json:"thinking,omitempty"`
}

type responseFormat struct {
	Type string `json:"type"`
}

type chatTemplateKwargs struct {
	EnableThinking bool `json:"enable_thinking"`
}

type thinking struct {
	Type string `json:"type"`
}

type chatChoice struct {
	Message chatChoiceMessage `json:"message"`
}

type chatChoiceMessage struct {
	Content string `json:"content"`
}

type chatError struct {
	Message string `json:"message"`
}

type chatResponse struct {
	Choices []chatChoice `json:"choices"`
	Error   *chatError   `json:"error,omitempty"`
}
