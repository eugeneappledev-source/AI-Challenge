struct ChatReply: Equatable, Sendable {
    let answer: String
    let model: String
    let totalTokens: Int
}
