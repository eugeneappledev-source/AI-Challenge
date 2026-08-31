struct ChatResponseDTO: Decodable, Sendable {
    struct Usage: Decodable, Sendable {
        let promptTokens: Int
        let completionTokens: Int
        let totalTokens: Int
    }

    let answer: String
    let model: String
    let usage: Usage
}
