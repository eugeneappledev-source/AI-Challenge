struct AIAgentMessageRequestDTO: Encodable {
    let message: String
    let conversationId: String?

    init(message: String, conversationId: String? = nil) {
        self.message = message
        self.conversationId = conversationId
    }
}
