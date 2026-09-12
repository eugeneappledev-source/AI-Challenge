struct AIAgentMessageRequestDTO: Encodable {
    let message: String
    let conversationId: String?
    let compression: Bool

    init(message: String, conversationId: String? = nil, compression: Bool = false) {
        self.message = message
        self.conversationId = conversationId
        self.compression = compression
    }
}

struct ContextComparisonRequestDTO: Encodable {
    let conversationId: String
    let question: String
}

struct ContextStrategyMessageRequestDTO: Encodable {
    let sessionId: String
    let strategy: ContextStrategy
    let branchId: String?
    let windowSize: Int
    let message: String
}

struct ContextStrategySessionRequestDTO: Encodable {
    let sessionId: String
    let windowSize: Int?

    init(sessionId: String, windowSize: Int? = nil) {
        self.sessionId = sessionId
        self.windowSize = windowSize
    }
}
