import Foundation

struct ModelBenchmarkAPI: Sendable {
    private let baseURL: URL
    private let accessToken: String
    private let httpClient: any HTTPClient

    init(baseURL: URL, accessToken: String, httpClient: any HTTPClient) {
        self.baseURL = baseURL
        self.accessToken = accessToken
        self.httpClient = httpClient
    }

    func run(prompt: String, tier: ModelTier) async throws -> ModelBenchmarkAttempt {
        try await perform(
            path: "v1/models/run",
            body: RunModelBenchmarkRequestDTO(prompt: prompt, tier: tier),
            responseType: ModelBenchmarkAttempt.self
        )
    }

    func review(prompt: String, attempts: [ModelBenchmarkAttempt]) async throws -> ModelBenchmarkReview {
        try await perform(
            path: "v1/models/review",
            body: ReviewModelBenchmarkRequestDTO(prompt: prompt, attempts: attempts),
            responseType: ModelBenchmarkReview.self
        )
    }

    private func perform<Body: Encodable & Sendable, Response: Decodable & Sendable>(
        path: String,
        body: Body,
        responseType: Response.Type
    ) async throws -> Response {
        var request = URLRequest(url: baseURL.appending(path: path))
        request.httpMethod = "POST"
        request.timeoutInterval = 180
        request.setValue("application/json", forHTTPHeaderField: "Content-Type")
        request.setValue("application/json", forHTTPHeaderField: "Accept")
        if !accessToken.isEmpty {
            request.setValue("Bearer \(accessToken)", forHTTPHeaderField: "Authorization")
        }
        request.httpBody = try JSONEncoder().encode(body)

        let (data, response) = try await httpClient.data(for: request)
        guard 200..<300 ~= response.statusCode else {
            let apiError = try? JSONDecoder().decode(APIErrorDTO.self, from: data)
            throw NetworkError.httpStatus(
                code: response.statusCode,
                message: apiError?.error.message ?? "Сервер временно недоступен (\(response.statusCode))."
            )
        }
        do {
            return try JSONDecoder().decode(responseType, from: data)
        } catch {
            throw NetworkError.decoding
        }
    }
}
