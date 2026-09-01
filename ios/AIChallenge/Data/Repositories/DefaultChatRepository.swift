struct DefaultChatRepository: ChatRepository {
    private let api: ChatAPI

    init(api: ChatAPI) {
        self.api = api
    }

    func send(message: String, mode: ResponseControlMode) async throws -> ChatReply {
        let response = try await api.send(message: message, mode: mode)
        return ChatReply(
            answer: response.answer,
            model: response.model,
            mode: response.mode ?? mode,
            finishReason: response.finishReason,
            promptTokens: response.usage.promptTokens,
            completionTokens: response.usage.completionTokens,
            totalTokens: response.usage.totalTokens
        )
    }
}
