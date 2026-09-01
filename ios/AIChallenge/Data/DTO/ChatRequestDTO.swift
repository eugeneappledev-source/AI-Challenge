struct ChatRequestDTO: Encodable, Sendable {
    let message: String
    let mode: ResponseControlMode
}
