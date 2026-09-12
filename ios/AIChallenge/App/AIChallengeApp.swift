import SwiftUI

@main
struct AIChallengeApp: App {
    @State private var dayOneViewModel = AppContainer.makeDayOneViewModel()
    @State private var dayTwoViewModel = AppContainer.makeChatViewModel()
    @State private var dayThreeViewModel = AppContainer.makeReasoningViewModel()
    @State private var dayFourViewModel = AppContainer.makeTemperatureViewModel()
    @State private var dayFiveViewModel = AppContainer.makeModelComparisonViewModel()
    @State private var daySixViewModel = AppContainer.makeAgentViewModel()
    @State private var daySevenViewModel = AppContainer.makeAgentMemoryViewModel()
    @State private var dayEightViewModel = AppContainer.makeTokenLabViewModel()
    @State private var dayNineViewModel = AppContainer.makeContextCompressionViewModel()

    var body: some Scene {
        WindowGroup {
            ChallengeCatalogScreen(
                dayOneViewModel: dayOneViewModel,
                dayTwoViewModel: dayTwoViewModel,
                dayThreeViewModel: dayThreeViewModel,
                dayFourViewModel: dayFourViewModel,
                dayFiveViewModel: dayFiveViewModel,
                daySixViewModel: daySixViewModel,
                daySevenViewModel: daySevenViewModel,
                dayEightViewModel: dayEightViewModel,
                dayNineViewModel: dayNineViewModel
            )
        }
    }
}
