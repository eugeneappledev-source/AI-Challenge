import Foundation

@MainActor
enum AppContainer {
    static func makeChatViewModel() -> ChatViewModel {
        let configuration = AppConfiguration.live()
        let httpClient = URLSessionHTTPClient(session: .shared)
        let api = ChatAPI(
            baseURL: configuration.baseURL,
            accessToken: configuration.accessToken,
            httpClient: httpClient
        )
        let repository = DefaultChatRepository(api: api)
        let sendMessage = SendMessageUseCase(repository: repository)
        return ChatViewModel(sendMessage: sendMessage)
    }
}
