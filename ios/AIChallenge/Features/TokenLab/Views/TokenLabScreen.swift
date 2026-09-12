import SwiftUI

struct TokenLabScreen: View {
    @State var viewModel: TokenLabViewModel

    var body: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: 18) {
                header
                inputCard
                if let answer = viewModel.lastAnswer { answerCard(answer) }
                if let metrics = viewModel.metrics {
                    exactMetrics(metrics)
                    contextGauge(metrics)
                    scenarioComparison(metrics.scenarios)
                    methodology
                }
            }.padding()
        }
        .background(Color(.systemGroupedBackground))
        .navigationTitle("День 8")
        .navigationBarTitleDisplayMode(.inline)
        .task { await viewModel.load() }
        .alert("Token Lab", isPresented: Binding(get: { viewModel.errorMessage != nil }, set: { if !$0 { viewModel.errorMessage = nil } })) {
            Button("Понятно", role: .cancel) {}
        } message: { Text(viewModel.errorMessage ?? "Неизвестная ошибка") }
    }

    private var header: some View {
        VStack(alignment: .leading, spacing: 8) {
            Text("TOKEN LAB").font(.caption.bold()).tracking(1).foregroundStyle(Color.aiBlue)
            Text("Цена каждого сообщения.").font(.largeTitle.bold())
            Text("Отделяем точные usage-метрики DeepSeek от локальной оценки ещё не отправленного текста.")
                .font(.subheadline).foregroundStyle(.secondary).lineSpacing(3)
        }.padding(.vertical, 8)
    }

    private var inputCard: some View {
        VStack(alignment: .leading, spacing: 12) {
            TextField("Введите сообщение…", text: $viewModel.input, axis: .vertical)
                .lineLimit(2...6).padding(13).background(Color(.tertiarySystemGroupedBackground), in: RoundedRectangle(cornerRadius: 14))
                .disabled(viewModel.isLoading)
            HStack {
                Button("Короткий") { viewModel.useExample("Объясни API одним предложением.") }
                Button("Развёрнутый") { viewModel.useExample("Составь подробный план изучения AI-агентов с примерами и критериями проверки каждого шага.") }
                Spacer()
                Button { Task { await viewModel.send() } } label: {
                    Group { if viewModel.isLoading { ProgressView().tint(.white) } else { Label("Отправить", systemImage: "paperplane.fill") } }
                        .font(.caption.bold()).padding(.horizontal, 13).frame(height: 38).foregroundStyle(.white)
                        .background(viewModel.canSend ? Color.aiBlue : Color(.tertiarySystemFill), in: Capsule())
                }.disabled(!viewModel.canSend)
            }.buttonStyle(.borderless).font(.caption)
        }.padding(16).background(Color(.secondarySystemGroupedBackground), in: RoundedRectangle(cornerRadius: 20))
    }

    private func answerCard(_ answer: String) -> some View {
        VStack(alignment: .leading, spacing: 8) {
            Label("Последний ответ", systemImage: "sparkles").font(.headline).foregroundStyle(Color.aiBlue)
            Text(answer).font(.body).lineSpacing(3).textSelection(.enabled)
        }.padding(16).background(Color(.secondarySystemGroupedBackground), in: RoundedRectangle(cornerRadius: 20))
    }

    private func exactMetrics(_ metrics: AgentTokenMetrics) -> some View {
        VStack(alignment: .leading, spacing: 13) {
            HStack { Label("Метрики провайдера", systemImage: "checkmark.seal.fill").font(.headline); Spacer(); Text("ТОЧНЫЕ").font(.caption2.bold()).foregroundStyle(Color.aiSuccess) }
            LazyVGrid(columns: [.init(.flexible()), .init(.flexible())], spacing: 10) {
                metric("Текущий input + context", metrics.lastContextPromptTokens, "prompt_tokens")
                metric("Последний ответ", metrics.lastResponseTokens, "completion_tokens")
                metric("Вся история вызовов", metrics.cumulativeTotalTokens, "cumulative")
                metric("Расчётная стоимость", String(format: "$%.6f", metrics.estimatedCostUSD), "peak rate")
            }
        }.padding(16).background(Color(.secondarySystemGroupedBackground), in: RoundedRectangle(cornerRadius: 20))
    }

    private func contextGauge(_ metrics: AgentTokenMetrics) -> some View {
        VStack(alignment: .leading, spacing: 10) {
            HStack { Label("Текущий контекст", systemImage: "gauge.with.dots.needle.50percent").font(.headline); Spacer(); Text("ОЦЕНКА").font(.caption2.bold()).foregroundStyle(Color.aiCoral) }
            ProgressView(value: min(Double(metrics.historyEstimatedTokens) / Double(metrics.contextWindowTokens), 1)).tint(Color.aiBlue)
            HStack { Text("≈ \(metrics.historyEstimatedTokens) токенов"); Spacer(); Text("лимит \(metrics.contextWindowTokens.formatted())") }.font(.caption).foregroundStyle(.secondary)
            Text("Последнее сообщение: ≈ \(metrics.currentMessageEstimatedTokens) токенов · сообщений: \(metrics.messageCount)")
                .font(.caption).foregroundStyle(.secondary)
        }.padding(16).background(Color.aiBlue.opacity(0.07), in: RoundedRectangle(cornerRadius: 20))
    }

    private func scenarioComparison(_ scenarios: [TokenScenario]) -> some View {
        VStack(alignment: .leading, spacing: 12) {
            Text("Три сценария").font(.title3.bold())
            ForEach(scenarios) { scenario in
                VStack(alignment: .leading, spacing: 7) {
                    HStack {
                        Image(systemName: scenario.accepted ? "checkmark.circle.fill" : "xmark.octagon.fill").foregroundStyle(scenario.accepted ? Color.aiSuccess : Color.red)
                        Text(scenario.title).font(.subheadline.bold()); Spacer()
                        Text("≈ \(scenario.estimatedTokens.formatted())").font(.caption.bold().monospaced())
                    }
                    Text("\(scenario.messageCount) сообщений · \(scenario.outcome)").font(.caption).foregroundStyle(.secondary).lineSpacing(2)
                }.padding(14).background(Color(.secondarySystemGroupedBackground), in: RoundedRectangle(cornerRadius: 16))
            }
        }
    }

    private var methodology: some View {
        VStack(alignment: .leading, spacing: 8) {
            Label("Что ломается", systemImage: "exclamationmark.triangle.fill").font(.headline).foregroundStyle(Color.aiCoral)
            Text("При превышении context window запрос нельзя корректно обработать: API вернёт ошибку длины. Демонстрация делает безопасный preflight и блокирует заведомо переполненный payload до платного вызова.")
                .font(.caption).foregroundStyle(.secondary).lineSpacing(3)
        }.padding(16).background(Color.aiCoral.opacity(0.08), in: RoundedRectangle(cornerRadius: 18))
    }

    private func metric(_ title: String, _ value: Int, _ note: String) -> some View { metric(title, value.formatted(), note) }
    private func metric(_ title: String, _ value: String, _ note: String) -> some View {
        VStack(alignment: .leading, spacing: 4) {
            Text(title).font(.caption2).foregroundStyle(.secondary)
            Text(value).font(.title3.bold().monospaced()).minimumScaleFactor(0.7)
            Text(note).font(.system(size: 9)).foregroundStyle(.tertiary)
        }.frame(maxWidth: .infinity, minHeight: 72, alignment: .leading).padding(11)
            .background(Color(.tertiarySystemGroupedBackground), in: RoundedRectangle(cornerRadius: 13))
    }
}
