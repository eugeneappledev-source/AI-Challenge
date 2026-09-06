protocol ReasoningRepository: Sendable {
    func run(problem: String, method: ReasoningMethod) async throws -> ReasoningAttempt
    func review(problem: String, attempts: [ReasoningAttempt]) async throws -> ReasoningReview
}
