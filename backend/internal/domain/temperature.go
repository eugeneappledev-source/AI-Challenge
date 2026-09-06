package domain

type Temperature float64

const (
	TemperaturePrecise  Temperature = 0
	TemperatureBalanced Temperature = 0.7
	TemperatureCreative Temperature = 1.2
)

func (t Temperature) IsValid() bool {
	return t == TemperaturePrecise || t == TemperatureBalanced || t == TemperatureCreative
}

type TemperatureAttempt struct {
	Temperature  Temperature `json:"temperature"`
	Answer       string      `json:"answer"`
	Model        string      `json:"model"`
	FinishReason string      `json:"finishReason"`
	Usage        Usage       `json:"usage"`
}

type TemperatureScore struct {
	Temperature Temperature `json:"temperature"`
	Accuracy    int         `json:"accuracy"`
	Creativity  int         `json:"creativity"`
	Diversity   int         `json:"diversity"`
	Feedback    string      `json:"feedback"`
}

type TemperatureRecommendation struct {
	Temperature Temperature `json:"temperature"`
	BestFor     []string    `json:"bestFor"`
	Caution     string      `json:"caution"`
}

type TemperatureReview struct {
	Summary         string                      `json:"summary"`
	BestAccuracy    Temperature                 `json:"bestAccuracy"`
	BestCreativity  Temperature                 `json:"bestCreativity"`
	BestDiversity   Temperature                 `json:"bestDiversity"`
	Differences     []string                    `json:"differences"`
	Scores          []TemperatureScore          `json:"scores"`
	Recommendations []TemperatureRecommendation `json:"recommendations"`
	Model           string                      `json:"model"`
	Usage           Usage                       `json:"usage"`
}
