protocol AgentTokenRepository: Sendable {
    func send(message: String, conversationID: String) async throws -> AIAgentExchange
    func tokenMetrics(conversationID: String) async throws -> AgentTokenMetrics
    func clearHistory(conversationID: String) async throws
}
