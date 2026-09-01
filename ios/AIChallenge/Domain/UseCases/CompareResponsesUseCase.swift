struct CompareResponsesUseCase: Sendable {
    private let sendMessage: SendMessageUseCase

    init(sendMessage: SendMessageUseCase) {
        self.sendMessage = sendMessage
    }

    func execute(message: String) async throws -> ResponseComparison {
        async let unrestricted = sendMessage.execute(message: message, mode: .unrestricted)
        async let controlled = sendMessage.execute(message: message, mode: .controlled)

        return try await ResponseComparison(
            prompt: message,
            unrestricted: unrestricted,
            controlled: controlled
        )
    }
}
