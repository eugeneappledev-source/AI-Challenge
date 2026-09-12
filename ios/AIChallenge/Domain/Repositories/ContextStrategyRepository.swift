protocol ContextStrategyRepository: Sendable {
    func state(sessionID: String, strategy: ContextStrategy, branchID: String?) async throws -> ContextStrategyState
    func send(message: String, sessionID: String, strategy: ContextStrategy, branchID: String?) async throws -> ContextStrategyExchange
    func createBranches(sessionID: String) async throws -> ContextStrategyState
    func compare(sessionID: String) async throws -> ContextStrategyComparison
    func clear(sessionID: String) async throws
}
