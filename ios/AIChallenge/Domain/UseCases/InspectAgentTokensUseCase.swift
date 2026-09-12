struct InspectAgentTokensUseCase: Sendable {
    private let repository: any AgentTokenRepository

    init(repository: any AgentTokenRepository) { self.repository = repository }

    func send(message: String, conversationID: String) async throws -> (AIAgentExchange, AgentTokenMetrics) {
        let exchange = try await repository.send(message: message, conversationID: conversationID)
        let metrics = try await repository.tokenMetrics(conversationID: conversationID)
        return (exchange, metrics)
    }

    func load(conversationID: String) async throws -> AgentTokenMetrics {
        try await repository.tokenMetrics(conversationID: conversationID)
    }

    func clear(conversationID: String) async throws {
        try await repository.clearHistory(conversationID: conversationID)
    }
}
