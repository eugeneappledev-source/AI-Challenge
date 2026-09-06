import SwiftUI

struct ReasoningComparisonScreen: View {
    @State var viewModel: ReasoningComparisonViewModel
    @FocusState private var isProblemFocused: Bool

    var body: some View {
        ScrollView {
            LazyVStack(alignment: .leading, spacing: 20) {
                introduction
                experimentRule
                problemEditor

                if viewModel.isRunning {
                    ReasoningProgressCard(
                        attempts: viewModel.attemptsByMethod,
                        isReviewing: viewModel.isReviewing
                    )
                } else if let comparison = viewModel.comparison {
                    resultSection(comparison)
                } else {
                    readyState
                }
            }
            .padding()
        }
        .background(Color(.systemGroupedBackground))
        .navigationTitle("День 3")
        .navigationBarTitleDisplayMode(.inline)
        .toolbar(.visible, for: .navigationBar)
        .onChange(of: viewModel.problem) {
            viewModel.clearResult()
        }
        .alert(
            "Эксперимент не завершён",
            isPresented: Binding(
                get: { viewModel.errorMessage != nil },
                set: { if !$0 { viewModel.errorMessage = nil } }
            )
        ) {
            Button("Понятно", role: .cancel) {
                viewModel.errorMessage = nil
            }
        } message: {
            Text(viewModel.errorMessage ?? "Неизвестная ошибка")
        }
    }

    private var introduction: some View {
        VStack(alignment: .leading, spacing: 8) {
            Text("РАЗНЫЕ СПОСОБЫ РАССУЖДЕНИЯ")
                .font(.caption.weight(.bold))
                .foregroundStyle(Color.aiCoral)
                .tracking(1.05)

            Text("Одна задача.\nЧетыре стратегии.")
                .font(.largeTitle.bold())

            Text("Сравни прямой ответ, пошаговое решение, meta-prompt и группу экспертов. После них независимый AI-арбитр проверит результат.")
                .font(.subheadline)
                .foregroundStyle(.secondary)
                .lineSpacing(3)
        }
        .padding(.vertical, 8)
    }

    private var experimentRule: some View {
        HStack(alignment: .top, spacing: 12) {
            Image(systemName: "equal.circle.fill")
                .font(.title2)
                .foregroundStyle(Color.aiForest)

            VStack(alignment: .leading, spacing: 4) {
                Text("Честное сравнение")
                    .font(.headline)
                Text("После запуска текст фиксируется и без изменений используется во всех четырёх способах. Меняются только инструкции модели.")
                    .font(.caption)
                    .foregroundStyle(.secondary)
                    .lineSpacing(2)
            }
        }
        .padding(16)
        .background(Color.aiForest.opacity(0.08), in: RoundedRectangle(cornerRadius: 20, style: .continuous))
    }

    private var problemEditor: some View {
        VStack(alignment: .leading, spacing: 14) {
            HStack {
                Label("Задача", systemImage: "function")
                    .font(.headline)

                Spacer()

                if viewModel.isRunning {
                    Text("ЗАФИКСИРОВАНА")
                        .font(.system(size: 9, weight: .bold))
                        .foregroundStyle(Color.aiCoral)
                        .padding(.horizontal, 8)
                        .padding(.vertical, 5)
                        .background(Color.aiCoral.opacity(0.1), in: Capsule())
                }
            }

            TextEditor(text: $viewModel.problem)
                .font(.system(.body, design: .serif, weight: .medium))
                .lineSpacing(4)
                .frame(minHeight: 138)
                .scrollContentBackground(.hidden)
                .padding(12)
                .background(Color(.tertiarySystemGroupedBackground), in: RoundedRectangle(cornerRadius: 14, style: .continuous))
                .focused($isProblemFocused)
                .disabled(viewModel.isRunning)

            HStack(spacing: 10) {
                Button {
                    isProblemFocused = false
                    viewModel.useNextExample()
                } label: {
                    Label("Другой пример", systemImage: "arrow.triangle.2.circlepath")
                }
                .buttonStyle(ReasoningSecondaryButtonStyle())

                Button {
                    viewModel.clearProblem()
                    isProblemFocused = true
                } label: {
                    Label("Очистить", systemImage: "xmark")
                }
                .buttonStyle(ReasoningSecondaryButtonStyle())
            }
            .disabled(viewModel.isRunning)

            Button {
                isProblemFocused = false
                Task { await viewModel.runComparison() }
            } label: {
                HStack(spacing: 10) {
                    Image(systemName: "brain.head.profile")
                    Text("Запустить 4 способа")
                        .fontWeight(.semibold)
                }
                .frame(maxWidth: .infinity)
                .frame(height: 50)
                .foregroundStyle(viewModel.canRun ? .white : Color(.tertiaryLabel))
                .background(
                    viewModel.canRun ? Color.aiForest : Color(.quaternarySystemFill),
                    in: RoundedRectangle(cornerRadius: 15, style: .continuous)
                )
            }
            .buttonStyle(.plain)
            .disabled(!viewModel.canRun)
        }
        .padding(16)
        .background(Color(.secondarySystemGroupedBackground), in: RoundedRectangle(cornerRadius: 22, style: .continuous))
        .overlay {
            RoundedRectangle(cornerRadius: 22, style: .continuous)
                .stroke(Color.primary.opacity(0.05))
        }
    }

    private var readyState: some View {
        VStack(spacing: 12) {
            Image(systemName: "rectangle.3.group.bubble.left")
                .font(.system(size: 36))
                .foregroundStyle(Color.aiCoral)
            Text("Готово к эксперименту")
                .font(.headline)
            Text("Можно использовать пример или полностью заменить его своей логической, алгоритмической или аналитической задачей.")
                .font(.caption)
                .foregroundStyle(.secondary)
                .multilineTextAlignment(.center)
        }
        .frame(maxWidth: .infinity)
        .padding(.vertical, 28)
        .padding(.horizontal, 20)
    }

    private func resultSection(_ comparison: ReasoningComparison) -> some View {
        VStack(alignment: .leading, spacing: 18) {
            HStack {
                Text("Решения")
                    .font(.title2.bold())
                Spacer()
                Text("4 ИЗ 4")
                    .font(.caption2.weight(.bold))
                    .foregroundStyle(Color.aiSuccess)
            }

            ReasoningMethodPicker(selection: $viewModel.selectedMethod)

            if let attempt = viewModel.selectedAttempt {
                ReasoningAttemptCard(
                    attempt: attempt,
                    isWinner: comparison.review.winner == attempt.method
                )
                .id(attempt.method)
                .transition(.opacity.combined(with: .scale(scale: 0.98)))
            }

            ReasoningReviewCard(review: comparison.review)

            Button {
                viewModel.clearResult()
                isProblemFocused = true
            } label: {
                Label("Новый эксперимент", systemImage: "arrow.counterclockwise")
                    .fontWeight(.semibold)
                    .frame(maxWidth: .infinity)
                    .frame(height: 48)
            }
            .buttonStyle(.bordered)
            .tint(.aiForest)
        }
        .animation(.easeInOut(duration: 0.2), value: viewModel.selectedMethod)
    }
}

private struct ReasoningProgressCard: View {
    let attempts: [ReasoningMethod: ReasoningAttempt]
    let isReviewing: Bool

    var body: some View {
        VStack(alignment: .leading, spacing: 16) {
            HStack {
                VStack(alignment: .leading, spacing: 3) {
                    Text(isReviewing ? "Арбитр сравнивает решения" : "DeepSeek решает задачу")
                        .font(.headline)
                    Text(isReviewing ? "Формируется итоговый вердикт" : "Готово способов: \(attempts.count) из 4")
                        .font(.caption)
                        .foregroundStyle(.secondary)
                }
                Spacer()
                ProgressView()
                    .tint(.aiForest)
            }

            ForEach(ReasoningMethod.allCases) { method in
                HStack(spacing: 11) {
                    Image(systemName: attempts[method] == nil ? method.systemImage : "checkmark.circle.fill")
                        .foregroundStyle(attempts[method] == nil ? method.color : Color.aiSuccess)
                        .frame(width: 22)
                    Text(method.title)
                        .font(.subheadline.weight(.medium))
                    Spacer()
                    Text(attempts[method] == nil ? "В работе" : "Готово")
                        .font(.caption2.weight(.bold))
                        .foregroundStyle(attempts[method] == nil ? .secondary : Color.aiSuccess)
                }
            }

            if isReviewing {
                Divider()
                Label("Пятый запрос: независимая проверка", systemImage: "checkmark.bubble")
                    .font(.caption.weight(.semibold))
                    .foregroundStyle(Color.aiForest)
            }
        }
        .padding(18)
        .background(Color(.secondarySystemGroupedBackground), in: RoundedRectangle(cornerRadius: 22, style: .continuous))
    }
}

private struct ReasoningMethodPicker: View {
    @Binding var selection: ReasoningMethod

    var body: some View {
        ScrollView(.horizontal) {
            HStack(spacing: 8) {
                ForEach(ReasoningMethod.allCases) { method in
                    Button {
                        selection = method
                    } label: {
                        Label(method.shortTitle, systemImage: method.systemImage)
                            .font(.caption.weight(.semibold))
                            .foregroundStyle(selection == method ? .white : method.color)
                            .padding(.horizontal, 12)
                            .padding(.vertical, 9)
                            .background(selection == method ? method.color : method.color.opacity(0.1), in: Capsule())
                    }
                    .buttonStyle(.plain)
                }
            }
        }
        .scrollIndicators(.hidden)
    }
}

private struct ReasoningAttemptCard: View {
    let attempt: ReasoningAttempt
    let isWinner: Bool

    var body: some View {
        VStack(alignment: .leading, spacing: 18) {
            HStack(spacing: 12) {
                Image(systemName: attempt.method.systemImage)
                    .font(.headline)
                    .foregroundStyle(.white)
                    .frame(width: 42, height: 42)
                    .background(attempt.method.color, in: RoundedRectangle(cornerRadius: 13, style: .continuous))

                VStack(alignment: .leading, spacing: 2) {
                    Text(attempt.method.title)
                        .font(.headline)
                    Text(attempt.method.requestDescription)
                        .font(.caption2)
                        .foregroundStyle(.secondary)
                }

                Spacer()

                if isWinner {
                    Label("ЛУЧШИЙ", systemImage: "trophy.fill")
                        .font(.system(size: 9, weight: .bold))
                        .foregroundStyle(Color.aiCoral)
                        .padding(.horizontal, 8)
                        .padding(.vertical, 6)
                        .background(Color.aiCoral.opacity(0.1), in: Capsule())
                }
            }

            if let generatedPrompt = attempt.generatedPrompt, !generatedPrompt.isEmpty {
                GeneratedPromptView(prompt: generatedPrompt)
            }

            if !attempt.experts.isEmpty {
                VStack(alignment: .leading, spacing: 12) {
                    Text("МНЕНИЯ ЭКСПЕРТОВ")
                        .font(.caption2.weight(.bold))
                        .foregroundStyle(attempt.method.color)
                        .tracking(0.7)

                    ForEach(Array(attempt.experts.enumerated()), id: \.offset) { _, expert in
                        ExpertSolutionView(expert: expert)
                    }
                }

                Divider()

                Text("ОБЩИЙ ВЫВОД")
                    .font(.caption2.weight(.bold))
                    .foregroundStyle(attempt.method.color)
                    .tracking(0.7)
            }

            ModelAnswerText(attempt.answer)

            Divider()

            HStack {
                Label(attempt.model, systemImage: "cpu")
                Spacer()
                Text("\(attempt.usage.totalTokens) токенов")
                Text("·")
                Text(attempt.finishReason)
            }
            .font(.caption2)
            .foregroundStyle(.secondary)
        }
        .padding(18)
        .background(Color(.secondarySystemGroupedBackground), in: RoundedRectangle(cornerRadius: 22, style: .continuous))
        .overlay {
            RoundedRectangle(cornerRadius: 22, style: .continuous)
                .stroke(isWinner ? Color.aiCoral.opacity(0.35) : Color.primary.opacity(0.05))
        }
    }
}

private struct GeneratedPromptView: View {
    let prompt: String
    @State private var isExpanded = false

    var body: some View {
        DisclosureGroup(isExpanded: $isExpanded) {
            Text(prompt)
                .font(.system(.caption, design: .monospaced))
                .textSelection(.enabled)
                .frame(maxWidth: .infinity, alignment: .leading)
                .padding(.top, 10)
        } label: {
            Label("Промпт, созданный первым запросом", systemImage: "wand.and.stars")
                .font(.subheadline.weight(.semibold))
        }
        .tint(.primary)
        .padding(14)
        .background(Color.aiPurple.opacity(0.09), in: RoundedRectangle(cornerRadius: 14, style: .continuous))
    }
}

private struct ExpertSolutionView: View {
    let expert: ExpertSolution

    var body: some View {
        VStack(alignment: .leading, spacing: 7) {
            Label(expert.role, systemImage: expert.systemImage)
                .font(.subheadline.weight(.bold))
                .foregroundStyle(Color.aiForest)
            ModelAnswerText(expert.answer, font: .subheadline)
        }
        .padding(14)
        .background(Color.aiForest.opacity(0.07), in: RoundedRectangle(cornerRadius: 14, style: .continuous))
    }
}

private struct ReasoningReviewCard: View {
    let review: ReasoningReview

    var body: some View {
        VStack(alignment: .leading, spacing: 20) {
            HStack(spacing: 13) {
                Image(systemName: "checkmark.bubble.fill")
                    .font(.title2)
                    .foregroundStyle(.white)
                    .frame(width: 48, height: 48)
                    .background(Color.aiForest, in: RoundedRectangle(cornerRadius: 15, style: .continuous))

                VStack(alignment: .leading, spacing: 3) {
                    Text("Вердикт AI-арбитра")
                        .font(.headline)
                    Text("Победитель: \(review.winner.title)")
                        .font(.caption.weight(.semibold))
                        .foregroundStyle(Color.aiCoral)
                }
            }

            ModelAnswerText(review.verdict)

            VStack(alignment: .leading, spacing: 9) {
                Label("Эталонная проверка", systemImage: "checkmark.seal.fill")
                    .font(.subheadline.weight(.bold))
                    .foregroundStyle(Color.aiForest)
                ModelAnswerText(review.referenceAnswer, font: .subheadline)
            }
            .padding(14)
            .background(Color.aiForest.opacity(0.08), in: RoundedRectangle(cornerRadius: 14, style: .continuous))

            VStack(alignment: .leading, spacing: 10) {
                Text("КЛЮЧЕВЫЕ ОТЛИЧИЯ")
                    .font(.caption2.weight(.bold))
                    .foregroundStyle(Color.aiCoral)
                    .tracking(0.7)

                ForEach(Array(review.differences.enumerated()), id: \.offset) { index, difference in
                    HStack(alignment: .top, spacing: 10) {
                        Text("\(index + 1)")
                            .font(.caption2.weight(.bold))
                            .foregroundStyle(.white)
                            .frame(width: 23, height: 23)
                            .background(Color.aiCoral, in: Circle())
                        Text(difference)
                            .font(.subheadline)
                            .frame(maxWidth: .infinity, alignment: .leading)
                    }
                }
            }

            VStack(alignment: .leading, spacing: 12) {
                Text("ОЦЕНКИ")
                    .font(.caption2.weight(.bold))
                    .foregroundStyle(Color.aiCoral)
                    .tracking(0.7)

                ForEach(ReasoningMethod.allCases) { method in
                    if let score = review.scores.first(where: { $0.method == method }) {
                        ReasoningScoreView(score: score, isWinner: review.winner == method)
                    }
                }
            }

            HStack {
                Label(review.model, systemImage: "cpu")
                Spacer()
                Text("\(review.usage.totalTokens) токенов")
            }
            .font(.caption2)
            .foregroundStyle(.secondary)

            Label("Арбитр — ещё одна LLM-оценка, а не формальное доказательство. Для объективности лучше выбирать задачи с проверяемым ответом.", systemImage: "info.circle")
                .font(.caption2)
                .foregroundStyle(.secondary)
        }
        .padding(18)
        .background(Color(.secondarySystemGroupedBackground), in: RoundedRectangle(cornerRadius: 22, style: .continuous))
        .overlay {
            RoundedRectangle(cornerRadius: 22, style: .continuous)
                .stroke(Color.aiForest.opacity(0.2))
        }
    }
}

private struct ReasoningScoreView: View {
    let score: ReasoningScore
    let isWinner: Bool

    var body: some View {
        VStack(alignment: .leading, spacing: 10) {
            HStack {
                Label(score.method.shortTitle, systemImage: score.method.systemImage)
                    .font(.subheadline.weight(.bold))
                    .foregroundStyle(score.method.color)
                Spacer()
                Text(String(format: "%.1f", score.average))
                    .font(.headline.monospacedDigit())
                Text("/ 10")
                    .font(.caption)
                    .foregroundStyle(.secondary)
            }

            HStack(spacing: 7) {
                ScoreMetric(title: "Точность", value: score.correctness)
                ScoreMetric(title: "Ясность", value: score.clarity)
                ScoreMetric(title: "Проверка", value: score.verification)
            }

            Text(score.feedback)
                .font(.caption)
                .foregroundStyle(.secondary)
                .lineSpacing(2)
        }
        .padding(13)
        .background(
            isWinner ? Color.aiCoral.opacity(0.08) : Color(.tertiarySystemGroupedBackground),
            in: RoundedRectangle(cornerRadius: 14, style: .continuous)
        )
    }
}

private struct ScoreMetric: View {
    let title: String
    let value: Int

    var body: some View {
        VStack(spacing: 3) {
            Text("\(value)")
                .font(.caption.weight(.bold).monospacedDigit())
            Text(title)
                .font(.system(size: 9))
                .foregroundStyle(.secondary)
        }
        .frame(maxWidth: .infinity)
        .padding(.vertical, 7)
        .background(Color.primary.opacity(0.04), in: RoundedRectangle(cornerRadius: 9, style: .continuous))
    }
}

private struct ModelAnswerText: View {
    private let text: String
    private let font: Font

    init(_ text: String, font: Font = .body) {
        self.text = text
        self.font = font
    }

    var body: some View {
        if let attributed = try? AttributedString(markdown: text) {
            Text(attributed)
                .font(font)
                .lineSpacing(4)
                .textSelection(.enabled)
                .frame(maxWidth: .infinity, alignment: .leading)
        } else {
            Text(text)
                .font(font)
                .lineSpacing(4)
                .textSelection(.enabled)
                .frame(maxWidth: .infinity, alignment: .leading)
        }
    }
}

private struct ReasoningSecondaryButtonStyle: ButtonStyle {
    func makeBody(configuration: Configuration) -> some View {
        configuration.label
            .font(.caption.weight(.semibold))
            .foregroundStyle(Color.aiForest)
            .frame(maxWidth: .infinity)
            .padding(.vertical, 10)
            .background(Color.aiForest.opacity(configuration.isPressed ? 0.15 : 0.08), in: RoundedRectangle(cornerRadius: 11, style: .continuous))
    }
}

private extension ReasoningMethod {
    var title: String {
        switch self {
        case .direct: "Прямой ответ"
        case .stepByStep: "Пошаговое решение"
        case .metaPrompt: "Промпт от модели"
        case .expertPanel: "Группа экспертов"
        }
    }

    var shortTitle: String {
        switch self {
        case .direct: "Прямой"
        case .stepByStep: "По шагам"
        case .metaPrompt: "Meta-prompt"
        case .expertPanel: "Эксперты"
        }
    }

    var requestDescription: String {
        switch self {
        case .direct: "1 API-вызов · без дополнительной инструкции"
        case .stepByStep: "1 API-вызов · инструкция «решай пошагово»"
        case .metaPrompt: "2 API-вызова · создание промпта и решение"
        case .expertPanel: "1 API-вызов · аналитик, инженер и критик"
        }
    }

    var systemImage: String {
        switch self {
        case .direct: "bolt.fill"
        case .stepByStep: "list.number"
        case .metaPrompt: "wand.and.stars"
        case .expertPanel: "person.3.fill"
        }
    }

    var color: Color {
        switch self {
        case .direct: .aiBlue
        case .stepByStep: .aiPurple
        case .metaPrompt: .aiCoral
        case .expertPanel: .aiForest
        }
    }
}

private extension ExpertSolution {
    var systemImage: String {
        switch role.lowercased() {
        case let value where value.contains("аналитик"): "chart.xyaxis.line"
        case let value where value.contains("инженер"): "gearshape.2.fill"
        case let value where value.contains("критик"): "checkmark.magnifyingglass"
        default: "person.fill"
        }
    }
}
