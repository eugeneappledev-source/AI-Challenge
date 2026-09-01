struct ChatResponseDTO: Decodable, Sendable {
    struct Usage: Decodable, Sendable {
        let promptTokens: Int
        let completionTokens: Int
        let totalTokens: Int
    }

    let answer: String
    let model: String
    let mode: ResponseControlMode?
    let finishReason: String?
    let usage: Usage
}
