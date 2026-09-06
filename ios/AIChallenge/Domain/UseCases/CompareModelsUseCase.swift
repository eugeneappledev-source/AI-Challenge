struct CompareModelsUseCase: Sendable {
    private let repository: any ModelBenchmarkRepository

    init(repository: any ModelBenchmarkRepository) {
        self.repository = repository
    }

    func execute(
        prompt: String,
        onAttempt: @escaping @MainActor @Sendable (ModelBenchmarkAttempt) -> Void
    ) async throws -> ModelComparison {
        let attempts = try await withThrowingTaskGroup(
            of: ModelBenchmarkAttempt.self,
            returning: [ModelBenchmarkAttempt].self
        ) { group in
            for tier in ModelTier.allCases {
                group.addTask { try await repository.run(prompt: prompt, tier: tier) }
            }

            var results: [ModelBenchmarkAttempt] = []
            for try await attempt in group {
                results.append(attempt)
                await onAttempt(attempt)
            }
            return ModelTier.allCases.compactMap { tier in results.first { $0.tier == tier } }
        }

        let review = try await repository.review(prompt: prompt, attempts: attempts)
        return ModelComparison(prompt: prompt, attempts: attempts, review: review)
    }
}
