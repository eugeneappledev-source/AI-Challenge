struct DefaultAIAgentRepository: AIAgentRepository, AgentMemoryRepository, AgentTokenRepository, ContextCompressionRepository, ContextStrategyRepository {
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
    func state(sessionID: String, strategy: ContextStrategy, branchID: String?, windowSize: Int) async throws -> ContextStrategyState {
        try await api.strategyState(sessionID: sessionID, strategy: strategy, branchID: branchID, windowSize: windowSize)
    }
    func send(message: String, sessionID: String, strategy: ContextStrategy, branchID: String?, windowSize: Int) async throws -> ContextStrategyExchange {
        try await api.sendStrategy(message: message, sessionID: sessionID, strategy: strategy, branchID: branchID, windowSize: windowSize)
    }
    func createBranches(sessionID: String) async throws -> ContextStrategyState { try await api.createStrategyBranches(sessionID: sessionID) }
    func compare(sessionID: String, windowSize: Int) async throws -> ContextStrategyComparison {
        try await api.compareContextStrategies(sessionID: sessionID, windowSize: windowSize)
    }
    func clear(sessionID: String) async throws { try await api.clearContextStrategies(sessionID: sessionID) }
}
