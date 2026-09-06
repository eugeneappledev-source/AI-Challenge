struct DefaultModelBenchmarkRepository: ModelBenchmarkRepository {
    private let api: ModelBenchmarkAPI

    init(api: ModelBenchmarkAPI) {
        self.api = api
    }

    func run(prompt: String, tier: ModelTier) async throws -> ModelBenchmarkAttempt {
        try await api.run(prompt: prompt, tier: tier)
    }

    func review(prompt: String, attempts: [ModelBenchmarkAttempt]) async throws -> ModelBenchmarkReview {
        try await api.review(prompt: prompt, attempts: attempts)
    }
}
