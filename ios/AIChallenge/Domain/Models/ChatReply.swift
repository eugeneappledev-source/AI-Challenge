enum ResponseControlMode: String, Codable, CaseIterable, Identifiable, Sendable {
    case unrestricted
    case controlled

    var id: Self { self }
}

struct ControlledFoodAnswer: Decodable, Equatable, Sendable {
    enum Status: String, Decodable, Sendable {
        case ok
        case outOfScope = "out_of_scope"
    }

    let status: Status
    let answer: String
    let ingredients: [String]
    let steps: [String]
}

struct ChatReply: Equatable, Sendable {
    let answer: String
    let controlledAnswer: ControlledFoodAnswer?
    let model: String
    let mode: ResponseControlMode
    let finishReason: String?
    let promptTokens: Int
    let completionTokens: Int
    let totalTokens: Int

    init(
        answer: String,
        controlledAnswer: ControlledFoodAnswer? = nil,
        model: String,
        mode: ResponseControlMode,
        finishReason: String?,
        promptTokens: Int,
        completionTokens: Int,
        totalTokens: Int
    ) {
        self.answer = answer
        self.controlledAnswer = controlledAnswer
        self.model = model
        self.mode = mode
        self.finishReason = finishReason
        self.promptTokens = promptTokens
        self.completionTokens = completionTokens
        self.totalTokens = totalTokens
    }
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
