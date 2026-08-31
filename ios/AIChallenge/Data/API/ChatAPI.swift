import Foundation

struct ChatAPI: Sendable {
    private let baseURL: URL
    private let accessToken: String
    private let httpClient: any HTTPClient

    init(baseURL: URL, accessToken: String, httpClient: any HTTPClient) {
        self.baseURL = baseURL
        self.accessToken = accessToken
        self.httpClient = httpClient
    }

    func send(message: String) async throws -> ChatResponseDTO {
        let endpoint = baseURL.appending(path: "v1/chat")
        var request = URLRequest(url: endpoint)
        request.httpMethod = "POST"
        request.timeoutInterval = 65
        request.setValue("application/json", forHTTPHeaderField: "Content-Type")
        request.setValue("application/json", forHTTPHeaderField: "Accept")
        if !accessToken.isEmpty {
            request.setValue("Bearer \(accessToken)", forHTTPHeaderField: "Authorization")
        }
        request.httpBody = try JSONEncoder().encode(ChatRequestDTO(message: message))

        let (data, response) = try await httpClient.data(for: request)
        guard 200..<300 ~= response.statusCode else {
            let apiError = try? JSONDecoder().decode(APIErrorDTO.self, from: data)
            let message = apiError?.error.message ?? "Сервер временно недоступен (\(response.statusCode))."
            throw NetworkError.httpStatus(code: response.statusCode, message: message)
        }

        do {
            return try JSONDecoder().decode(ChatResponseDTO.self, from: data)
        } catch {
            throw NetworkError.decoding
        }
    }
}
