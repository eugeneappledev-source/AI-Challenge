struct DefaultReasoningRepository: ReasoningRepository {
    private let api: ReasoningAPI

    init(api: ReasoningAPI) {
        self.api = api
    }

    func run(problem: String, method: ReasoningMethod) async throws -> ReasoningAttempt {
        try await api.run(problem: problem, method: method)
    }

    func review(problem: String, attempts: [ReasoningAttempt]) async throws -> ReasoningReview {
        try await api.review(problem: problem, attempts: attempts)
    }
}
