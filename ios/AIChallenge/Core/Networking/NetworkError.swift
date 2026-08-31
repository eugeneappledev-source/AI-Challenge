import Foundation

enum NetworkError: LocalizedError, Sendable {
    case invalidResponse
    case httpStatus(code: Int, message: String)
    case decoding

    var errorDescription: String? {
        switch self {
        case .invalidResponse:
            "Сервер вернул некорректный ответ."
        case let .httpStatus(_, message):
            message
        case .decoding:
            "Не удалось прочитать ответ сервера."
        }
    }
}
