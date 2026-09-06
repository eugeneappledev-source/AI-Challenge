struct RunReasoningRequestDTO: Encodable {
    let problem: String
    let method: ReasoningMethod
}

struct ReviewReasoningRequestDTO: Encodable {
    let problem: String
    let attempts: [ReasoningAttempt]
}
