protocol ModelBenchmarkRepository: Sendable {
    func run(prompt: String, tier: ModelTier) async throws -> ModelBenchmarkAttempt
    func review(prompt: String, attempts: [ModelBenchmarkAttempt]) async throws -> ModelBenchmarkReview
}
