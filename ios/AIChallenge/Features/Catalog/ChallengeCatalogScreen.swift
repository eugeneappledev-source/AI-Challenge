import SwiftUI

struct ChallengeCatalogScreen: View {
    let dayOneViewModel: DayOneViewModel
    let dayTwoViewModel: ChatViewModel
    let dayThreeViewModel: ReasoningComparisonViewModel
    let dayFourViewModel: TemperatureComparisonViewModel
    let dayFiveViewModel: ModelComparisonViewModel
    let daySixViewModel: AIAgentViewModel
    let daySevenViewModel: AgentMemoryViewModel
    let dayEightViewModel: TokenLabViewModel
    let dayNineViewModel: ContextCompressionViewModel

    private let challenges = ChallengeCatalogItem.all

    var body: some View {
        NavigationStack {
            ScrollView {
                LazyVStack(alignment: .leading, spacing: 22) {
                    header
                    progressCard

                    VStack(alignment: .leading, spacing: 14) {
                        Text("Задания недели")
                            .font(.title2.bold())

                        ForEach(challenges) { challenge in
                            NavigationLink(value: challenge.route) {
                                ChallengeCard(challenge: challenge)
                            }
                            .buttonStyle(.plain)
                        }
                    }
                }
                .padding(.horizontal, 18)
                .padding(.top, 12)
                .padding(.bottom, 32)
            }
            .background(CatalogBackground())
            .toolbar(.hidden, for: .navigationBar)
            .navigationDestination(for: ChallengeRoute.self) { route in
                switch route {
                case .dayOne:
                    DayOneScreen(viewModel: dayOneViewModel)
                case .dayTwo:
                    ChatScreen(viewModel: dayTwoViewModel)
                case .dayThree:
                    ReasoningComparisonScreen(viewModel: dayThreeViewModel)
                case .dayFour:
                    TemperatureComparisonScreen(viewModel: dayFourViewModel)
                case .dayFive:
                    ModelComparisonScreen(viewModel: dayFiveViewModel)
                case .daySix:
                    AIAgentScreen(viewModel: daySixViewModel)
                case .daySeven:
                    AgentMemoryScreen(viewModel: daySevenViewModel)
                case .dayEight:
                    TokenLabScreen(viewModel: dayEightViewModel)
                case .dayNine:
                    ContextCompressionScreen(viewModel: dayNineViewModel)
                }
            }
        }
        .tint(.aiForest)
    }

    private var header: some View {
        VStack(alignment: .leading, spacing: 9) {
            HStack(spacing: 8) {
                Image(systemName: "sparkles")
                Text("AI ADVENT CHALLENGE · ПОТОК 9")
                    .tracking(1.1)
            }
            .font(.caption2.weight(.bold))
            .foregroundStyle(Color.aiCoral)

            Text("Учимся управлять\nязыковой моделью")
                .font(.system(.largeTitle, design: .rounded, weight: .bold))
                .minimumScaleFactor(0.82)

            Text("Одно приложение развивается вместе с заданиями — от первого API-запроса до сравнения параметров и версий моделей.")
                .font(.subheadline)
                .foregroundStyle(.secondary)
                .lineSpacing(3)
        }
    }

    private var progressCard: some View {
        HStack(spacing: 16) {
            ZStack {
                Circle()
                    .stroke(Color.aiForest.opacity(0.12), lineWidth: 7)
                Circle()
                    .trim(from: 0, to: 1)
                    .stroke(Color.aiForest, style: StrokeStyle(lineWidth: 7, lineCap: .round))
                    .rotationEffect(.degrees(-90))
                Text("9/9")
                    .font(.caption.weight(.bold))
            }
            .frame(width: 54, height: 54)

            VStack(alignment: .leading, spacing: 4) {
                Text("Общий прогресс")
                    .font(.headline)
                Text("Первая неделя завершена. Начат блок про AI-агентов.")
                    .font(.caption)
                    .foregroundStyle(.secondary)
            }

            Spacer(minLength: 0)
        }
        .padding(16)
        .background(.regularMaterial, in: RoundedRectangle(cornerRadius: 22, style: .continuous))
        .overlay {
            RoundedRectangle(cornerRadius: 22, style: .continuous)
                .stroke(Color.primary.opacity(0.05))
        }
    }
}

private struct ChallengeCard: View {
    let challenge: ChallengeCatalogItem

    var body: some View {
        HStack(spacing: 15) {
            ZStack {
                RoundedRectangle(cornerRadius: 17, style: .continuous)
                    .fill(challenge.tint.opacity(0.12))
                Image(systemName: challenge.systemImage)
                    .font(.title2.weight(.semibold))
                    .foregroundStyle(challenge.tint)
            }
            .frame(width: 58, height: 58)

            VStack(alignment: .leading, spacing: 7) {
                HStack(spacing: 8) {
                    Text("ДЕНЬ \(challenge.number)")
                        .font(.caption2.weight(.bold))
                        .tracking(0.8)
                        .foregroundStyle(challenge.tint)

                    ChallengeStatusBadge(status: challenge.status)
                }

                Text(challenge.title)
                    .font(.headline)
                    .foregroundStyle(.primary)

                Text(challenge.subtitle)
                    .font(.caption)
                    .foregroundStyle(.secondary)
                    .lineLimit(2)
            }

            Spacer(minLength: 4)

            Image(systemName: "chevron.right")
                .font(.caption.weight(.bold))
                .foregroundStyle(.tertiary)
        }
        .padding(16)
        .background(Color(.secondarySystemGroupedBackground), in: RoundedRectangle(cornerRadius: 22, style: .continuous))
        .overlay {
            RoundedRectangle(cornerRadius: 22, style: .continuous)
                .stroke(challenge.status == .current ? challenge.tint.opacity(0.35) : Color.primary.opacity(0.05))
        }
        .shadow(color: Color.black.opacity(0.035), radius: 12, y: 5)
    }
}

private struct ChallengeStatusBadge: View {
    let status: ChallengeStatus

    var body: some View {
        Label(status.title, systemImage: status.systemImage)
            .font(.system(size: 9, weight: .bold))
            .foregroundStyle(status.color)
            .padding(.horizontal, 7)
            .padding(.vertical, 4)
            .background(status.color.opacity(0.1), in: Capsule())
    }
}

private struct CatalogBackground: View {
    var body: some View {
        ZStack {
            Color(.systemGroupedBackground)

            Circle()
                .fill(Color.aiCoral.opacity(0.08))
                .frame(width: 260, height: 260)
                .blur(radius: 2)
                .offset(x: 160, y: -310)

            Circle()
                .fill(Color.aiForest.opacity(0.07))
                .frame(width: 310, height: 310)
                .offset(x: -180, y: 360)
        }
        .ignoresSafeArea()
    }
}

private enum ChallengeStatus: Hashable {
    case completed
    case current

    var title: String {
        switch self {
        case .completed: "ГОТОВО"
        case .current: "СЛЕДУЮЩИЙ"
        }
    }

    var systemImage: String {
        switch self {
        case .completed: "checkmark.circle.fill"
        case .current: "arrow.right.circle.fill"
        }
    }

    var color: Color {
        switch self {
        case .completed: .aiSuccess
        case .current: .aiCoral
        }
    }
}

private enum ChallengeRoute: Hashable {
    case dayOne
    case dayTwo
    case dayThree
    case dayFour
    case dayFive
    case daySix
    case daySeven
    case dayEight
    case dayNine
}

private struct ChallengeCatalogItem: Identifiable {
    let number: String
    let title: String
    let subtitle: String
    let systemImage: String
    let tint: Color
    let status: ChallengeStatus
    let route: ChallengeRoute

    var id: String { number }

    static let all: [ChallengeCatalogItem] = [
        ChallengeCatalogItem(
            number: "01",
            title: "Первый запрос к LLM",
            subtitle: "Отправка запроса через API и вывод ответа модели.",
            systemImage: "paperplane.fill",
            tint: .blue,
            status: .completed,
            route: .dayOne
        ),
        ChallengeCatalogItem(
            number: "02",
            title: "Формат ответа",
            subtitle: "Свободный ответ против JSON с форматом и лимитами.",
            systemImage: "slider.horizontal.3",
            tint: .aiForest,
            status: .completed,
            route: .dayTwo
        ),
        ChallengeCatalogItem(
            number: "03",
            title: "Способы рассуждения",
            subtitle: "Четыре стратегии решения и независимая оценка результата.",
            systemImage: "brain.head.profile",
            tint: .aiCoral,
            status: .completed,
            route: .dayThree
        ),
        ChallengeCatalogItem(
            number: "04",
            title: "Температура",
            subtitle: "Один запрос при 0, 0.7 и 1.2 с оценкой точности и креативности.",
            systemImage: "thermometer.medium",
            tint: .aiPurple,
            status: .completed,
            route: .dayFour
        ),
        ChallengeCatalogItem(
            number: "05",
            title: "Версии моделей",
            subtitle: "Три реальные модели: качество, скорость, токены и стоимость.",
            systemImage: "cpu",
            tint: .aiBlue,
            status: .completed,
            route: .dayFive
        ),
        ChallengeCatalogItem(
            number: "06",
            title: "Первый агент",
            subtitle: "Отдельная сущность инкапсулирует настройки и вызов LLM.",
            systemImage: "shippingbox.fill",
            tint: .aiCoral,
            status: .completed,
            route: .daySix
        ),
        ChallengeCatalogItem(
            number: "07",
            title: "Сохранение контекста",
            subtitle: "История в SQLite переживает перезапуск приложения и агента.",
            systemImage: "memorychip.fill",
            tint: .aiPurple,
            status: .completed,
            route: .daySeven
        ),
        ChallengeCatalogItem(
            number: "08",
            title: "Работа с токенами",
            subtitle: "Usage запроса, истории и ответа, стоимость и переполнение.",
            systemImage: "number.square.fill",
            tint: .aiBlue,
            status: .completed,
            route: .dayEight
        ),
        ChallengeCatalogItem(
            number: "09",
            title: "Сжатие истории",
            subtitle: "Summary + последние сообщения против полного контекста.",
            systemImage: "archivebox.fill",
            tint: .aiForest,
            status: .completed,
            route: .dayNine
        ),
    ]
}
