import SwiftUI

@main
struct AIChallengeApp: App {
    @State private var viewModel = AppContainer.makeChatViewModel()

    var body: some Scene {
        WindowGroup {
            ChatScreen(viewModel: viewModel)
        }
    }
}
