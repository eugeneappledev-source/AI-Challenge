protocol ContextCompressionRepository: Sendable {
    func history(conversationID: String) async throws -> AIAgentConversation
    func sendCompressed(message: String, conversationID: String) async throws -> AIAgentExchange
    func contextState(conversationID: String) async throws -> ContextState
    func compareContexts(conversationID: String, question: String) async throws -> ContextComparison
    func clearHistory(conversationID: String) async throws
}
