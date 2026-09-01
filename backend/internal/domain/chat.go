package domain

type ResponseMode string

const (
	ResponseModeUnrestricted ResponseMode = "unrestricted"
	ResponseModeControlled   ResponseMode = "controlled"
)

func (m ResponseMode) IsValid() bool {
	return m == ResponseModeUnrestricted || m == ResponseModeControlled
}

type CompletionRequest struct {
	Message string
	Mode    ResponseMode
}

type Usage struct {
	PromptTokens     int `json:"promptTokens"`
	CompletionTokens int `json:"completionTokens"`
	TotalTokens      int `json:"totalTokens"`
}

type ChatReply struct {
	Answer       string       `json:"answer"`
	Model        string       `json:"model"`
	Mode         ResponseMode `json:"mode"`
	FinishReason string       `json:"finishReason"`
	Usage        Usage        `json:"usage"`
}
