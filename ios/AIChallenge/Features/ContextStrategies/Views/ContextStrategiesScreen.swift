import SwiftUI

struct ContextStrategiesScreen: View {
    @State var viewModel: ContextStrategiesViewModel

    var body: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: 18) {
                header
                strategyPicker
                windowSizeControl
                strategyOverview
                if let state = viewModel.currentState { stateCard(state) }
                composer
                comparisonLauncher
                if let comparison = viewModel.comparison { comparisonSection(comparison) }
            }
            .padding()
        }
        .background(Color(.systemGroupedBackground))
        .navigationTitle("День 10")
        .navigationBarTitleDisplayMode(.inline)
        .toolbar {
            ToolbarItem(placement: .topBarTrailing) {
                Button("Очистить", role: .destructive) { Task { await viewModel.clear() } }
                    .disabled(viewModel.isLoading)
            }
        }
        .task { await viewModel.loadSelected() }
        .onChange(of: viewModel.selectedStrategy) { _, strategy in
            if strategy == .branching, viewModel.activeBranchID.isEmpty { viewModel.activeBranchID = "main" }
            Task { await viewModel.loadSelected() }
        }
        .onChange(of: viewModel.selectedWindowSize) { _, _ in
            Task { await viewModel.windowSizeDidChange() }
        }
        .alert("Context Strategies Lab", isPresented: Binding(
            get: { viewModel.errorMessage != nil },
            set: { if !$0 { viewModel.errorMessage = nil } }
        )) {
            Button("Понятно", role: .cancel) {}
        } message: { Text(viewModel.errorMessage ?? "Неизвестная ошибка") }
    }

    private var header: some View {
        VStack(alignment: .leading, spacing: 8) {
            Text("CONTEXT STRATEGIES LAB").font(.caption.bold()).tracking(1).foregroundStyle(Color.aiCoral)
            Text("Одна память — три поведения.").font(.largeTitle.bold())
            Text("Без summary: ограничиваем окно, выносим факты или создаём независимые ветки от checkpoint.")
                .font(.subheadline).foregroundStyle(.secondary).lineSpacing(3)
        }
        .padding(.vertical, 8)
    }

    private var strategyPicker: some View {
        Picker("Стратегия", selection: $viewModel.selectedStrategy) {
            ForEach(ContextStrategy.allCases) { strategy in Text(strategy.title).tag(strategy) }
        }
        .pickerStyle(.segmented)
    }

    private var windowSizeControl: some View {
        VStack(alignment: .leading, spacing: 10) {
            HStack {
                Label("Размер окна N", systemImage: "slider.horizontal.3")
                    .font(.headline)
                Spacer()
                Text("N = \(viewModel.selectedWindowSize)")
                    .font(.caption.bold().monospaced())
                    .foregroundStyle(Color.aiCoral)
            }
            Picker("Размер окна", selection: $viewModel.selectedWindowSize) {
                ForEach(viewModel.windowSizeOptions, id: \.self) { size in
                    Text("\(size) msg").tag(size)
                }
            }
            .pickerStyle(.segmented)
            .disabled(viewModel.isLoading)

            Text("N применяется к Sliding Window и Sticky Facts, а общий тест заново прогоняет обе стратегии с выбранным значением. Увеличение N не возвращает уже отброшенные сообщения — для чистого ручного опыта нажмите «Очистить».")
                .font(.caption2)
                .foregroundStyle(.secondary)
                .lineSpacing(2)
        }
        .padding(15)
        .background(Color(.secondarySystemGroupedBackground), in: RoundedRectangle(cornerRadius: 19))
    }

    private var strategyOverview: some View {
        HStack(alignment: .top, spacing: 13) {
            Image(systemName: strategyIcon(viewModel.selectedStrategy))
                .font(.title2).foregroundStyle(strategyColor(viewModel.selectedStrategy))
                .frame(width: 42, height: 42)
                .background(strategyColor(viewModel.selectedStrategy).opacity(0.1), in: RoundedRectangle(cornerRadius: 12))
            VStack(alignment: .leading, spacing: 5) {
                Text(viewModel.selectedStrategy.fullTitle).font(.headline)
                Text(viewModel.selectedStrategy.explanation).font(.caption).foregroundStyle(.secondary).lineSpacing(3)
            }
        }
        .padding(15)
        .background(Color(.secondarySystemGroupedBackground), in: RoundedRectangle(cornerRadius: 19))
    }

    private func stateCard(_ state: ContextStrategyState) -> some View {
        VStack(alignment: .leading, spacing: 13) {
            HStack {
                Label("Текущее состояние", systemImage: "memorychip").font(.headline)
                Spacer()
                Text("\(state.messages.count) / \(state.strategy == .branching ? "∞" : String(state.windowSize)) msg")
                    .font(.caption.bold().monospaced()).foregroundStyle(strategyColor(state.strategy))
            }

            if state.strategy == .stickyFacts {
                if state.facts.isEmpty {
                    Text("Facts пока пусты — отправьте первое содержательное сообщение.").font(.caption).foregroundStyle(.secondary)
                } else {
                    VStack(alignment: .leading, spacing: 7) {
                        Text("KEY-VALUE MEMORY").font(.system(size: 9, weight: .bold)).foregroundStyle(.secondary)
                        ForEach(state.facts) { fact in
                            HStack(alignment: .top) {
                                Text(fact.key).font(.caption2.bold().monospaced()).foregroundStyle(Color.aiPurple)
                                Spacer(minLength: 12)
                                Text(fact.value).font(.caption).multilineTextAlignment(.trailing)
                            }
                        }
                    }
                    .padding(12).background(Color.aiPurple.opacity(0.07), in: RoundedRectangle(cornerRadius: 14))
                }
            }

            if state.strategy == .branching { branchControls(state) }

            if !state.messages.isEmpty {
                VStack(alignment: .leading, spacing: 7) {
                    Text(state.strategy == .branching ? "СООБЩЕНИЯ АКТИВНОЙ ВЕТКИ" : "СОХРАНЁННОЕ ОКНО")
                        .font(.system(size: 9, weight: .bold)).foregroundStyle(.secondary)
                    ForEach(state.messages.suffix(state.strategy == .branching ? 6 : state.windowSize)) { message in
                        HStack(alignment: .top, spacing: 8) {
                            Circle().fill(message.role == .user ? Color.aiCoral : Color.aiForest).frame(width: 6, height: 6).padding(.top, 5)
                            Text(message.content).font(.caption).lineLimit(3)
                        }
                    }
                }
            }
        }
        .padding(16)
        .background(Color(.secondarySystemGroupedBackground), in: RoundedRectangle(cornerRadius: 20))
    }

    private func branchControls(_ state: ContextStrategyState) -> some View {
        VStack(alignment: .leading, spacing: 10) {
            HStack(spacing: 7) {
                ForEach(state.branches) { branch in
                    Button { Task { await viewModel.selectBranch(branch.id) } } label: {
                        VStack(spacing: 2) {
                            Text(branch.title).font(.caption.bold())
                            Text("\(branch.messageCount) msg").font(.system(size: 9).monospaced())
                        }
                        .frame(maxWidth: .infinity).padding(.vertical, 8)
                        .background(viewModel.activeBranchID == branch.id ? Color.aiBlue : Color.aiBlue.opacity(0.08), in: RoundedRectangle(cornerRadius: 11))
                        .foregroundStyle(viewModel.activeBranchID == branch.id ? .white : Color.aiBlue)
                    }
                    .buttonStyle(.plain)
                }
            }
            if state.branches.count == 1 {
                Button { Task { await viewModel.createCheckpoint() } } label: {
                    Label("Checkpoint → MVP + Growth", systemImage: "arrow.triangle.branch").frame(maxWidth: .infinity)
                }
                .buttonStyle(.bordered).tint(Color.aiBlue).disabled(!viewModel.canCreateCheckpoint)
            } else if let branch = state.branches.first(where: { $0.id == viewModel.activeBranchID }) {
                Text("Общая точка: \(branch.checkpointMessages) сообщений · дальше ветки независимы")
                    .font(.caption2).foregroundStyle(.secondary)
            }
        }
    }

    private var composer: some View {
        VStack(alignment: .leading, spacing: 10) {
            TextField("Введите своё сообщение…", text: $viewModel.input, axis: .vertical)
                .lineLimit(2...5).padding(13)
                .background(Color(.tertiarySystemGroupedBackground), in: RoundedRectangle(cornerRadius: 14))
            Button { Task { await viewModel.send() } } label: {
                Label("Отправить через \(viewModel.selectedStrategy.fullTitle)", systemImage: "paperplane.fill").frame(maxWidth: .infinity)
            }
            .buttonStyle(.borderedProminent).tint(strategyColor(viewModel.selectedStrategy)).disabled(!viewModel.canSend)
            if viewModel.isLoading {
                HStack(spacing: 9) { ProgressView(); Text(viewModel.progressText).font(.caption).foregroundStyle(.secondary) }
            }
        }
        .padding(15)
        .background(Color(.secondarySystemGroupedBackground), in: RoundedRectangle(cornerRadius: 19))
    }

    private var comparisonLauncher: some View {
        VStack(alignment: .leading, spacing: 10) {
            Label("Одинаковый тест для всех", systemImage: "scale.3d").font(.headline)
            Text("12 сообщений о ТЗ PulsePlan при N = \(viewModel.selectedWindowSize): ранние ограничения, свежие требования и две версии продукта. Результат оценивает отдельный LLM-рецензент.")
                .font(.caption).foregroundStyle(.secondary).lineSpacing(3)
            Button { Task { await viewModel.runComparison() } } label: {
                Label("Прогнать сценарий и сравнить", systemImage: "play.fill").frame(maxWidth: .infinity)
            }
            .buttonStyle(.borderedProminent).tint(Color.aiCoral).disabled(viewModel.isLoading)
        }
        .padding(16).background(Color.aiCoral.opacity(0.08), in: RoundedRectangle(cornerRadius: 20))
    }

    private func comparisonSection(_ comparison: ContextStrategyComparison) -> some View {
        VStack(alignment: .leading, spacing: 14) {
            HStack {
                Text("Результаты").font(.title2.bold())
                Spacer()
                Text("N = \(comparison.windowSize)")
                    .font(.caption.bold().monospaced())
                    .foregroundStyle(Color.aiCoral)
            }
            ForEach(comparison.results) { result in resultCard(result) }
            reviewCard(comparison.review)
        }
    }

    private func resultCard(_ result: ContextStrategyResult) -> some View {
        VStack(alignment: .leading, spacing: 11) {
            HStack {
                Label(result.strategy.fullTitle, systemImage: strategyIcon(result.strategy)).font(.headline).foregroundStyle(strategyColor(result.strategy))
                Spacer()
                Text("\(result.usage.totalTokens) tokens").font(.caption.bold().monospaced())
            }
            HStack(spacing: 8) {
                metric("Контекст", "\(result.messagesKept) msg")
                metric("Facts", "\(result.factsKept)")
                metric("Prompt", "\(result.usage.promptTokens)")
            }
            Text(result.behavior).font(.caption).foregroundStyle(.secondary).lineSpacing(3)
            if result.branches.isEmpty {
                Text(result.answer).font(.caption).lineSpacing(3).textSelection(.enabled)
            } else {
                ForEach(result.branches) { branch in
                    VStack(alignment: .leading, spacing: 5) {
                        Text(branch.title.uppercased()).font(.caption2.bold()).foregroundStyle(Color.aiBlue)
                        Text(branch.answer).font(.caption).lineSpacing(3).textSelection(.enabled)
                    }
                    .padding(11).background(Color.aiBlue.opacity(0.06), in: RoundedRectangle(cornerRadius: 13))
                }
            }
        }
        .padding(15).background(Color(.secondarySystemGroupedBackground), in: RoundedRectangle(cornerRadius: 19))
    }

    private func reviewCard(_ review: ContextStrategyReview) -> some View {
        VStack(alignment: .leading, spacing: 12) {
            HStack {
                Label("Независимый рецензент", systemImage: "checkmark.seal.fill").font(.headline)
                Spacer()
                Text("WINNER · \(review.winner.title.uppercased())").font(.system(size: 9, weight: .bold)).foregroundStyle(strategyColor(review.winner))
            }
            Text(review.verdict).font(.subheadline.bold())
            ForEach(review.scores) { score in
                VStack(alignment: .leading, spacing: 6) {
                    HStack {
                        Text(score.strategy.fullTitle).font(.caption.bold())
                        Spacer()
                        Text("Q \(score.quality) · S \(score.stability) · T \(score.tokenEfficiency) · UX \(score.usability)").font(.caption2.monospaced())
                    }
                    Text(score.feedback).font(.caption).foregroundStyle(.secondary)
                }
                Divider()
            }
            ForEach(review.differences, id: \.self) { Text("• " + $0).font(.caption).foregroundStyle(.secondary) }
            ForEach(review.recommendations, id: \.self) { Text($0).font(.caption.bold()).foregroundStyle(Color.aiPurple) }
        }
        .padding(16).background(Color.aiPurple.opacity(0.08), in: RoundedRectangle(cornerRadius: 20))
    }

    private func metric(_ title: String, _ value: String) -> some View {
        VStack(spacing: 3) {
            Text(value).font(.caption.bold().monospaced())
            Text(title).font(.system(size: 9)).foregroundStyle(.secondary)
        }
        .frame(maxWidth: .infinity).padding(.vertical, 8)
        .background(Color.primary.opacity(0.04), in: RoundedRectangle(cornerRadius: 10))
    }

    private func strategyIcon(_ strategy: ContextStrategy) -> String {
        switch strategy {
        case .slidingWindow: "rectangle.stack"
        case .stickyFacts: "pin.fill"
        case .branching: "arrow.triangle.branch"
        }
    }

    private func strategyColor(_ strategy: ContextStrategy) -> Color {
        switch strategy {
        case .slidingWindow: .aiCoral
        case .stickyFacts: .aiPurple
        case .branching: .aiBlue
        }
    }
}
