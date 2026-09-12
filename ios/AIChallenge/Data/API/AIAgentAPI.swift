import Foundation

struct AIAgentAPI: Sendable {
    private let baseURL: URL
    private let accessToken: String
    private let httpClient: any HTTPClient

    init(baseURL: URL, accessToken: String, httpClient: any HTTPClient) {
        self.baseURL = baseURL
        self.accessToken = accessToken
        self.httpClient = httpClient
    }

    func profile() async throws -> AIAgentProfile {
        var request = URLRequest(url: baseURL.appending(path: "v1/agent"))
        request.timeoutInterval = 30
        request.setValue("application/json", forHTTPHeaderField: "Accept")
        authorize(&request)
        return try await perform(request, as: AIAgentProfile.self)
    }

    func send(message: String) async throws -> AIAgentExchange {
        try await send(message: message, conversationID: nil)
    }

    func send(message: String, conversationID: String) async throws -> AIAgentExchange {
        try await send(message: message, conversationID: Optional(conversationID), compression: false)
    }

    func sendCompressed(message: String, conversationID: String) async throws -> AIAgentExchange {
        try await send(message: message, conversationID: Optional(conversationID), compression: true)
    }

    func history(conversationID: String) async throws -> AIAgentConversation {
        var components = URLComponents(url: baseURL.appending(path: "v1/agent/history"), resolvingAgainstBaseURL: false)
        components?.queryItems = [URLQueryItem(name: "conversationId", value: conversationID)]
        guard let url = components?.url else { throw NetworkError.invalidResponse }
        var request = URLRequest(url: url)
        request.timeoutInterval = 30
        request.setValue("application/json", forHTTPHeaderField: "Accept")
        authorize(&request)
        return try await perform(request, as: AIAgentConversation.self)
    }

    func clearHistory(conversationID: String) async throws {
        var components = URLComponents(url: baseURL.appending(path: "v1/agent/history"), resolvingAgainstBaseURL: false)
        components?.queryItems = [URLQueryItem(name: "conversationId", value: conversationID)]
        guard let url = components?.url else { throw NetworkError.invalidResponse }
        var request = URLRequest(url: url)
        request.httpMethod = "DELETE"
        request.timeoutInterval = 30
        authorize(&request)
        let (_, response) = try await httpClient.data(for: request)
        guard response.statusCode == 204 else {
            throw NetworkError.httpStatus(code: response.statusCode, message: "Не удалось очистить историю.")
        }
    }

    func tokenMetrics(conversationID: String) async throws -> AgentTokenMetrics {
        var components = URLComponents(url: baseURL.appending(path: "v1/agent/tokens"), resolvingAgainstBaseURL: false)
        components?.queryItems = [URLQueryItem(name: "conversationId", value: conversationID)]
        guard let url = components?.url else { throw NetworkError.invalidResponse }
        var request = URLRequest(url: url)
        request.timeoutInterval = 30
        request.setValue("application/json", forHTTPHeaderField: "Accept")
        authorize(&request)
        return try await perform(request, as: AgentTokenMetrics.self)
    }

    func contextState(conversationID: String) async throws -> ContextState {
        var components = URLComponents(url: baseURL.appending(path: "v1/agent/context"), resolvingAgainstBaseURL: false)
        components?.queryItems = [URLQueryItem(name: "conversationId", value: conversationID)]
        guard let url = components?.url else { throw NetworkError.invalidResponse }
        var request = URLRequest(url: url)
        request.timeoutInterval = 90
        request.setValue("application/json", forHTTPHeaderField: "Accept")
        authorize(&request)
        return try await perform(request, as: ContextState.self)
    }

    func compareContexts(conversationID: String, question: String) async throws -> ContextComparison {
        var request = URLRequest(url: baseURL.appending(path: "v1/agent/context/compare"))
        request.httpMethod = "POST"
        request.timeoutInterval = 180
        request.setValue("application/json", forHTTPHeaderField: "Content-Type")
        request.setValue("application/json", forHTTPHeaderField: "Accept")
        authorize(&request)
        request.httpBody = try JSONEncoder().encode(ContextComparisonRequestDTO(conversationId: conversationID, question: question))
        return try await perform(request, as: ContextComparison.self)
    }

    func strategyState(sessionID: String, strategy: ContextStrategy, branchID: String?) async throws -> ContextStrategyState {
        var components = URLComponents(url: baseURL.appending(path: "v1/agent/strategies/state"), resolvingAgainstBaseURL: false)
        components?.queryItems = [
            URLQueryItem(name: "sessionId", value: sessionID),
            URLQueryItem(name: "strategy", value: strategy.rawValue),
            URLQueryItem(name: "branchId", value: branchID),
        ]
        guard let url = components?.url else { throw NetworkError.invalidResponse }
        var request = URLRequest(url: url)
        request.timeoutInterval = 60
        request.setValue("application/json", forHTTPHeaderField: "Accept")
        authorize(&request)
        return try await perform(request, as: ContextStrategyState.self)
    }

    func sendStrategy(message: String, sessionID: String, strategy: ContextStrategy, branchID: String?) async throws -> ContextStrategyExchange {
        var request = URLRequest(url: baseURL.appending(path: "v1/agent/strategies/message"))
        request.httpMethod = "POST"
        request.timeoutInterval = 150
        request.setValue("application/json", forHTTPHeaderField: "Content-Type")
        request.setValue("application/json", forHTTPHeaderField: "Accept")
        authorize(&request)
        request.httpBody = try JSONEncoder().encode(ContextStrategyMessageRequestDTO(sessionId: sessionID, strategy: strategy, branchId: branchID, message: message))
        return try await perform(request, as: ContextStrategyExchange.self)
    }

    func createStrategyBranches(sessionID: String) async throws -> ContextStrategyState {
        var request = URLRequest(url: baseURL.appending(path: "v1/agent/strategies/branches"))
        request.httpMethod = "POST"
        request.timeoutInterval = 60
        request.setValue("application/json", forHTTPHeaderField: "Content-Type")
        request.setValue("application/json", forHTTPHeaderField: "Accept")
        authorize(&request)
        request.httpBody = try JSONEncoder().encode(ContextStrategySessionRequestDTO(sessionId: sessionID))
        return try await perform(request, as: ContextStrategyState.self)
    }

    func compareContextStrategies(sessionID: String) async throws -> ContextStrategyComparison {
        var request = URLRequest(url: baseURL.appending(path: "v1/agent/strategies/compare"))
        request.httpMethod = "POST"
        request.timeoutInterval = 300
        request.setValue("application/json", forHTTPHeaderField: "Content-Type")
        request.setValue("application/json", forHTTPHeaderField: "Accept")
        authorize(&request)
        request.httpBody = try JSONEncoder().encode(ContextStrategySessionRequestDTO(sessionId: sessionID))
        return try await perform(request, as: ContextStrategyComparison.self)
    }

    func clearContextStrategies(sessionID: String) async throws {
        var components = URLComponents(url: baseURL.appending(path: "v1/agent/strategies"), resolvingAgainstBaseURL: false)
        components?.queryItems = [URLQueryItem(name: "sessionId", value: sessionID)]
        guard let url = components?.url else { throw NetworkError.invalidResponse }
        var request = URLRequest(url: url)
        request.httpMethod = "DELETE"
        request.timeoutInterval = 60
        authorize(&request)
        let (_, response) = try await httpClient.data(for: request)
        guard response.statusCode == 204 else {
            throw NetworkError.httpStatus(code: response.statusCode, message: "Не удалось очистить Context Lab.")
        }
    }

    private func send(message: String, conversationID: String?, compression: Bool = false) async throws -> AIAgentExchange {
        var request = URLRequest(url: baseURL.appending(path: "v1/agent/message"))
        request.httpMethod = "POST"
        request.timeoutInterval = 135
        request.setValue("application/json", forHTTPHeaderField: "Content-Type")
        request.setValue("application/json", forHTTPHeaderField: "Accept")
        authorize(&request)
        request.httpBody = try JSONEncoder().encode(AIAgentMessageRequestDTO(message: message, conversationId: conversationID, compression: compression))
        return try await perform(request, as: AIAgentExchange.self)
    }

    private func authorize(_ request: inout URLRequest) {
        if !accessToken.isEmpty {
            request.setValue("Bearer \(accessToken)", forHTTPHeaderField: "Authorization")
        }
    }

    private func perform<Response: Decodable & Sendable>(_ request: URLRequest, as type: Response.Type) async throws -> Response {
        let (data, response) = try await httpClient.data(for: request)
        guard 200..<300 ~= response.statusCode else {
            let apiError = try? JSONDecoder().decode(APIErrorDTO.self, from: data)
            throw NetworkError.httpStatus(
                code: response.statusCode,
                message: apiError?.error.message ?? "Сервер временно недоступен (\(response.statusCode))."
            )
        }
        do {
            let decoder = JSONDecoder()
            decoder.dateDecodingStrategy = .flexibleISO8601
            return try decoder.decode(type, from: data)
        } catch {
            throw NetworkError.decoding
        }
    }
}

private extension JSONDecoder.DateDecodingStrategy {
    static var flexibleISO8601: JSONDecoder.DateDecodingStrategy {
        .custom { decoder in
            let container = try decoder.singleValueContainer()
            let value = try container.decode(String.self)
            let withFraction = ISO8601DateFormatter()
            withFraction.formatOptions = [.withInternetDateTime, .withFractionalSeconds]
            if let date = withFraction.date(from: value) { return date }
            let plain = ISO8601DateFormatter()
            if let date = plain.date(from: value) { return date }
            throw DecodingError.dataCorruptedError(in: container, debugDescription: "Invalid ISO-8601 date")
        }
    }
}
