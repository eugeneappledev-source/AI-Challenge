import Foundation
import Observation

@MainActor
@Observable
final class ContextCompressionViewModel {
    var input = ""
    var comparisonQuestion = "Какие мои цели и предпочтения ты запомнил?"
    private(set) var messages: [AIAgentMessage] = []
    private(set) var state: ContextState?
    private(set) var comparison: ContextComparison?
    private(set) var isLoading = false
    private(set) var progressText = ""
    var errorMessage: String?

    let conversationID: String
    private let useCase: ManageCompressedContextUseCase

    init(useCase: ManageCompressedContextUseCase, conversationID: String) {
        self.useCase = useCase
        self.conversationID = conversationID
    }

    var canSend: Bool { !input.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty && !isLoading }
    var canCompare: Bool { state?.compressionActive == true && !comparisonQuestion.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty && !isLoading }

    func load() async {
        guard state == nil, !isLoading else { return }
        await refresh()
    }

    func send() async {
        let value = input.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !value.isEmpty, !isLoading else { return }
        input = ""
        await send(value, progress: "Агент обновляет контекст…")
    }

    func buildDemoContext() async {
        guard !isLoading else { return }
        let prompts = [
            "Запомни профиль вымышленного пользователя: его зовут Алекс, он разрабатывает мобильные приложения на Swift и SwiftUI, предпочитает многослойную архитектуру Domain, Data и Presentation. Ответь только: принято.",
            "Запомни детали учебного проекта Алекса: backend написан на Go, запускается в Docker на тестовом Ubuntu-сервере и обращается к облачной языковой модели. Ответь только: принято.",
            "Алекс любит короткие и точные объяснения с конкретными примерами и проверяемыми результатами. Ответь только: принято.",
            "Его цель — на практике разобраться в AI-агентах, сохранении истории, токенах и компрессии контекста. Ответь только: принято.",
            "При сравнении решений Алексу важны реальные метрики и независимый фидбек, а не субъективное впечатление. Ответь только: принято.",
            "Интерфейс проекта должен понятно показывать результат в мобильном приложении и браузере. Ответь только: принято.",
            "Каждый учебный этап должен быть зафиксирован отдельным Git-тегом, чтобы проверяющий мог открыть точную версию кода. Ответь только: принято.",
            "Документация должна кратко объяснять архитектуру, API, способ проверки и полученный результат. Ответь только: принято.",
            "Кратко подтверди, что контекст Алекса сохранён, не перечисляя все факты.",
        ]
        isLoading = true
        errorMessage = nil
        defer { isLoading = false; progressText = "" }
        do {
            for (index, prompt) in prompts.enumerated() {
                progressText = "Создаю диалог: \(index + 1)/\(prompts.count)"
                let result = try await useCase.send(message: prompt, conversationID: conversationID)
                messages.append(result.0.userMessage)
                messages.append(result.0.reply)
                state = result.1
            }
            let loaded = try await useCase.load(conversationID: conversationID)
            messages = loaded.0.messages
            state = loaded.1
        } catch { show(error) }
    }

    func compare() async {
        let question = comparisonQuestion.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !question.isEmpty, canCompare else { return }
        isLoading = true
        progressText = "Два ответа + независимая рецензия…"
        errorMessage = nil
        defer { isLoading = false; progressText = "" }
        do { comparison = try await useCase.compare(question: question, conversationID: conversationID) }
        catch { show(error) }
    }

    func clear() async {
        guard !isLoading else { return }
        isLoading = true
        defer { isLoading = false }
        do {
            try await useCase.clear(conversationID: conversationID)
            messages = []; comparison = nil; state = nil
            await refresh()
        } catch { show(error) }
    }

    private func send(_ value: String, progress: String) async {
        isLoading = true
        progressText = progress
        errorMessage = nil
        defer { isLoading = false; progressText = "" }
        do {
            let result = try await useCase.send(message: value, conversationID: conversationID)
            messages.append(result.0.userMessage); messages.append(result.0.reply); state = result.1
        } catch { input = value; show(error) }
    }

    private func refresh() async {
        isLoading = true
        defer { isLoading = false }
        do {
            let result = try await useCase.load(conversationID: conversationID)
            messages = result.0.messages; state = result.1
        } catch { show(error) }
    }

    private func show(_ error: Error) {
        errorMessage = (error as? LocalizedError)?.errorDescription ?? error.localizedDescription
    }
}
