struct DefaultChatRepository: ChatRepository {
    private let api: ChatAPI

    init(api: ChatAPI) {
        self.api = api
    }

    func send(message: String) async throws -> ChatReply {
        let response = try await api.send(message: message)
        return ChatReply(
            answer: response.answer,
            model: response.model,
            totalTokens: response.usage.totalTokens
        )
    }
}
