package domain

type Usage struct {
	PromptTokens     int `json:"promptTokens"`
	CompletionTokens int `json:"completionTokens"`
	TotalTokens      int `json:"totalTokens"`
}

type ChatReply struct {
	Answer string `json:"answer"`
	Model  string `json:"model"`
	Usage  Usage  `json:"usage"`
}
