enum ResponseControlMode: String, Codable, CaseIterable, Identifiable, Sendable {
    case unrestricted
    case controlled

    var id: Self { self }
}

struct ChatReply: Equatable, Sendable {
    let answer: String
    let model: String
    let mode: ResponseControlMode
    let finishReason: String?
    let promptTokens: Int
    let completionTokens: Int
    let totalTokens: Int
}

struct ResponseComparison: Equatable, Sendable {
    let prompt: String
    let unrestricted: ChatReply
    let controlled: ChatReply

    func reply(for mode: ResponseControlMode) -> ChatReply {
        switch mode {
        case .unrestricted:
            unrestricted
        case .controlled:
            controlled
        }
    }
}
