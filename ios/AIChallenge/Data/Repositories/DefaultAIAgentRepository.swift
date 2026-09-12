struct DefaultAIAgentRepository: AIAgentRepository, AgentMemoryRepository, AgentTokenRepository, ContextCompressionRepository {
    private let api: AIAgentAPI

    init(api: AIAgentAPI) { self.api = api }

    func profile() async throws -> AIAgentProfile { try await api.profile() }
    func send(message: String) async throws -> AIAgentExchange { try await api.send(message: message) }
    func history(conversationID: String) async throws -> AIAgentConversation { try await api.history(conversationID: conversationID) }
    func send(message: String, conversationID: String) async throws -> AIAgentExchange {
        try await api.send(message: message, conversationID: conversationID)
    }
    func clearHistory(conversationID: String) async throws { try await api.clearHistory(conversationID: conversationID) }
    func tokenMetrics(conversationID: String) async throws -> AgentTokenMetrics { try await api.tokenMetrics(conversationID: conversationID) }
    func sendCompressed(message: String, conversationID: String) async throws -> AIAgentExchange {
        try await api.sendCompressed(message: message, conversationID: conversationID)
    }
    func contextState(conversationID: String) async throws -> ContextState { try await api.contextState(conversationID: conversationID) }
    func compareContexts(conversationID: String, question: String) async throws -> ContextComparison {
        try await api.compareContexts(conversationID: conversationID, question: question)
    }
}
