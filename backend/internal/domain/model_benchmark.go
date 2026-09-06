package domain

type ModelTier string

const (
	ModelTierBasic    ModelTier = "basic"
	ModelTierExtended ModelTier = "extended"
	ModelTierStrong   ModelTier = "strong"
)

func (t ModelTier) IsValid() bool {
	return t == ModelTierBasic || t == ModelTierExtended || t == ModelTierStrong
}

type ModelBenchmarkAttempt struct {
	Tier                ModelTier `json:"tier"`
	Model               string    `json:"model"`
	Answer              string    `json:"answer"`
	LatencyMilliseconds int64     `json:"latencyMilliseconds"`
	Usage               Usage     `json:"usage"`
	InputCostUSD        float64   `json:"inputCostUSD"`
	OutputCostUSD       float64   `json:"outputCostUSD"`
	EstimatedCostUSD    float64   `json:"estimatedCostUSD"`
	PricingPeriod       string    `json:"pricingPeriod"`
	FinishReason        string    `json:"finishReason"`
}

type ModelQualityScore struct {
	Tier         ModelTier `json:"tier"`
	Accuracy     int       `json:"accuracy"`
	Completeness int       `json:"completeness"`
	Clarity      int       `json:"clarity"`
	Feedback     string    `json:"feedback"`
}

type ModelRecommendation struct {
	Tier     ModelTier `json:"tier"`
	BestFor  []string  `json:"bestFor"`
	Tradeoff string    `json:"tradeoff"`
}

type ModelBenchmarkReview struct {
	QualityWinner   ModelTier             `json:"qualityWinner"`
	Fastest         ModelTier             `json:"fastest"`
	Cheapest        ModelTier             `json:"cheapest"`
	Summary         string                `json:"summary"`
	Differences     []string              `json:"differences"`
	Scores          []ModelQualityScore   `json:"scores"`
	Recommendations []ModelRecommendation `json:"recommendations"`
	ReviewerModel   string                `json:"reviewerModel"`
	ReviewerUsage   Usage                 `json:"reviewerUsage"`
}
