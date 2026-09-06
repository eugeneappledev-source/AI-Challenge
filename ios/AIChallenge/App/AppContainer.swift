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
