import Foundation

@MainActor
enum AppContainer {
    static func makeDayOneViewModel() -> DayOneViewModel {
        DayOneViewModel(sendMessage: makeSendMessageUseCase())
    }

    static func makeChatViewModel() -> ChatViewModel {
        let sendMessage = makeSendMessageUseCase()
        let compareResponses = CompareResponsesUseCase(sendMessage: sendMessage)
        return ChatViewModel(compareResponses: compareResponses)
    }

    static func makeReasoningViewModel() -> ReasoningComparisonViewModel {
        let configuration = AppConfiguration.live()
        let httpClient = URLSessionHTTPClient(session: .shared)
        let api = ReasoningAPI(
            baseURL: configuration.baseURL,
            accessToken: configuration.accessToken,
            httpClient: httpClient
        )
        let repository = DefaultReasoningRepository(api: api)
        let compareStrategies = CompareReasoningStrategiesUseCase(repository: repository)
        return ReasoningComparisonViewModel(compareStrategies: compareStrategies)
    }

    static func makeTemperatureViewModel() -> TemperatureComparisonViewModel {
        let configuration = AppConfiguration.live()
        let httpClient = URLSessionHTTPClient(session: .shared)
        let api = TemperatureAPI(
            baseURL: configuration.baseURL,
            accessToken: configuration.accessToken,
            httpClient: httpClient
        )
        let repository = DefaultTemperatureRepository(api: api)
        let compareTemperatures = CompareTemperaturesUseCase(repository: repository)
        return TemperatureComparisonViewModel(compareTemperatures: compareTemperatures)
    }

    static func makeModelComparisonViewModel() -> ModelComparisonViewModel {
        let configuration = AppConfiguration.live()
        let api = ModelBenchmarkAPI(
            baseURL: configuration.baseURL,
            accessToken: configuration.accessToken,
            httpClient: URLSessionHTTPClient(session: .shared)
        )
        let repository = DefaultModelBenchmarkRepository(api: api)
        return ModelComparisonViewModel(compareModels: CompareModelsUseCase(repository: repository))
    }

    static func makeAgentViewModel() -> AIAgentViewModel {
        let configuration = AppConfiguration.live()
        let api = AIAgentAPI(
            baseURL: configuration.baseURL,
            accessToken: configuration.accessToken,
            httpClient: URLSessionHTTPClient(session: .shared)
        )
        return AIAgentViewModel(
            talkToAgent: TalkToAgentUseCase(repository: DefaultAIAgentRepository(api: api))
        )
    }

    static func makeAgentMemoryViewModel() -> AgentMemoryViewModel {
        let configuration = AppConfiguration.live()
        let api = AIAgentAPI(
            baseURL: configuration.baseURL,
            accessToken: configuration.accessToken,
            httpClient: URLSessionHTTPClient(session: .shared)
        )
        let key = "day07.conversationID"
        let conversationID: String
        if let saved = UserDefaults.standard.string(forKey: key) {
            conversationID = saved
        } else {
            conversationID = "ios-" + UUID().uuidString.lowercased()
            UserDefaults.standard.set(conversationID, forKey: key)
        }
        return AgentMemoryViewModel(
            useCase: ContinueAgentConversationUseCase(repository: DefaultAIAgentRepository(api: api)),
            conversationID: conversationID
        )
    }

    static func makeTokenLabViewModel() -> TokenLabViewModel {
        let configuration = AppConfiguration.live()
        let api = AIAgentAPI(
            baseURL: configuration.baseURL,
            accessToken: configuration.accessToken,
            httpClient: URLSessionHTTPClient(session: .shared)
        )
        return TokenLabViewModel(
            useCase: InspectAgentTokensUseCase(repository: DefaultAIAgentRepository(api: api)),
            conversationID: stableConversationID(for: "day08")
        )
    }

    static func makeContextCompressionViewModel() -> ContextCompressionViewModel {
        let configuration = AppConfiguration.live()
        let api = AIAgentAPI(
            baseURL: configuration.baseURL,
            accessToken: configuration.accessToken,
            httpClient: URLSessionHTTPClient(session: .shared)
        )
        return ContextCompressionViewModel(
            useCase: ManageCompressedContextUseCase(repository: DefaultAIAgentRepository(api: api)),
            conversationID: stableConversationID(for: "day09")
        )
    }

    private static func stableConversationID(for day: String) -> String {
        let key = "\(day).conversationID"
        if let saved = UserDefaults.standard.string(forKey: key) { return saved }
        let value = "ios-\(day)-" + UUID().uuidString.lowercased()
        UserDefaults.standard.set(value, forKey: key)
        return value
    }

    private static func makeSendMessageUseCase() -> SendMessageUseCase {
        let configuration = AppConfiguration.live()
        let httpClient = URLSessionHTTPClient(session: .shared)
        let api = ChatAPI(
            baseURL: configuration.baseURL,
            accessToken: configuration.accessToken,
            httpClient: httpClient
        )
        let repository = DefaultChatRepository(api: api)
        return SendMessageUseCase(repository: repository)
    }
}
