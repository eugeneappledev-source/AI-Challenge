package domain

const (
	ControlledAnswerWordLimit = 80
	FoodOutOfScopeMessage     = "Я отвечаю только на вопросы о еде и приготовлении. Пожалуйста, задайте вопрос по теме."
)

type ControlledAnswerStatus string

const (
	ControlledAnswerStatusOK         ControlledAnswerStatus = "ok"
	ControlledAnswerStatusOutOfScope ControlledAnswerStatus = "out_of_scope"
)

type ControlledAnswer struct {
	Status      ControlledAnswerStatus `json:"status"`
	Answer      string                 `json:"answer"`
	Ingredients []string               `json:"ingredients"`
	Steps       []string               `json:"steps"`
}

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
