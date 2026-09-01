import Foundation

struct DefaultChatRepository: ChatRepository {
    private let api: ChatAPI

    init(api: ChatAPI) {
        self.api = api
    }

    func send(message: String, mode: ResponseControlMode) async throws -> ChatReply {
        let response = try await api.send(message: message, mode: mode)
        let responseMode = response.mode ?? mode
        let controlledAnswer = try decodeControlledAnswer(
            response.answer,
            mode: responseMode
        )

        return ChatReply(
            answer: response.answer,
            controlledAnswer: controlledAnswer,
            model: response.model,
            mode: responseMode,
            finishReason: response.finishReason,
            promptTokens: response.usage.promptTokens,
            completionTokens: response.usage.completionTokens,
            totalTokens: response.usage.totalTokens
        )
    }

    private func decodeControlledAnswer(
        _ answer: String,
        mode: ResponseControlMode
    ) throws -> ControlledFoodAnswer? {
        guard mode == .controlled else { return nil }
        guard let data = answer.data(using: .utf8) else {
            throw NetworkError.decoding
        }

        do {
            return try JSONDecoder().decode(ControlledFoodAnswer.self, from: data)
        } catch {
            throw NetworkError.decoding
        }
    }
}
