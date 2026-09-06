import SwiftUI

@main
struct AIChallengeApp: App {
    @State private var dayOneViewModel = AppContainer.makeDayOneViewModel()
    @State private var dayTwoViewModel = AppContainer.makeChatViewModel()
    @State private var dayThreeViewModel = AppContainer.makeReasoningViewModel()

    var body: some Scene {
        WindowGroup {
            ChallengeCatalogScreen(
                dayOneViewModel: dayOneViewModel,
                dayTwoViewModel: dayTwoViewModel,
                dayThreeViewModel: dayThreeViewModel
            )
        }
    }
}
