struct APIErrorDTO: Decodable, Sendable {
    struct Detail: Decodable, Sendable {
        let code: String
        let message: String
    }

    let error: Detail
}
