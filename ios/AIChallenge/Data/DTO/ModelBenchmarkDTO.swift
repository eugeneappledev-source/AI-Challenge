struct RunModelBenchmarkRequestDTO: Encodable {
    let prompt: String
    let tier: ModelTier
}

struct ReviewModelBenchmarkRequestDTO: Encodable {
    let prompt: String
    let attempts: [ModelBenchmarkAttempt]
}
