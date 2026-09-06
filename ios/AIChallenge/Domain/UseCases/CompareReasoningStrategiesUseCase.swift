struct CompareReasoningStrategiesUseCase: Sendable {
    private let repository: any ReasoningRepository

    init(repository: any ReasoningRepository) {
        self.repository = repository
    }

    func execute(
        problem: String,
        onAttempt: @escaping @MainActor @Sendable (ReasoningAttempt) -> Void
    ) async throws -> ReasoningComparison {
        let attempts = try await withThrowingTaskGroup(
            of: ReasoningAttempt.self,
            returning: [ReasoningAttempt].self
        ) { group in
            for method in ReasoningMethod.allCases {
                group.addTask {
                    try await repository.run(problem: problem, method: method)
                }
            }

            var results: [ReasoningAttempt] = []
            for try await attempt in group {
                results.append(attempt)
                await onAttempt(attempt)
            }
            return ReasoningMethod.allCases.compactMap { method in
                results.first { $0.method == method }
            }
        }

        let review = try await repository.review(problem: problem, attempts: attempts)
        return ReasoningComparison(problem: problem, attempts: attempts, review: review)
    }
}
