import Foundation

struct ReasoningAPI: Sendable {
    private let baseURL: URL
    private let accessToken: String
    private let httpClient: any HTTPClient

    init(baseURL: URL, accessToken: String, httpClient: any HTTPClient) {
        self.baseURL = baseURL
        self.accessToken = accessToken
        self.httpClient = httpClient
    }

    func run(problem: String, method: ReasoningMethod) async throws -> ReasoningAttempt {
        try await perform(
            path: "v1/reasoning/run",
            body: RunReasoningRequestDTO(problem: problem, method: method),
            responseType: ReasoningAttempt.self
        )
    }

    func review(problem: String, attempts: [ReasoningAttempt]) async throws -> ReasoningReview {
        try await perform(
            path: "v1/reasoning/review",
            body: ReviewReasoningRequestDTO(problem: problem, attempts: attempts),
            responseType: ReasoningReview.self
        )
    }

    private func perform<Body: Encodable & Sendable, Response: Decodable & Sendable>(
        path: String,
        body: Body,
        responseType: Response.Type
    ) async throws -> Response {
        let endpoint = baseURL.appending(path: path)
        var request = URLRequest(url: endpoint)
        request.httpMethod = "POST"
        request.timeoutInterval = 135
        request.setValue("application/json", forHTTPHeaderField: "Content-Type")
        request.setValue("application/json", forHTTPHeaderField: "Accept")
        if !accessToken.isEmpty {
            request.setValue("Bearer \(accessToken)", forHTTPHeaderField: "Authorization")
        }
        request.httpBody = try JSONEncoder().encode(body)

        let (data, response) = try await httpClient.data(for: request)
        guard 200..<300 ~= response.statusCode else {
            let apiError = try? JSONDecoder().decode(APIErrorDTO.self, from: data)
            let message = apiError?.error.message ?? "Сервер временно недоступен (\(response.statusCode))."
            throw NetworkError.httpStatus(code: response.statusCode, message: message)
        }

        do {
            return try JSONDecoder().decode(responseType, from: data)
        } catch {
            throw NetworkError.decoding
        }
    }
}
