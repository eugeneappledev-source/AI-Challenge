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
