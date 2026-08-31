import Foundation

struct AppConfiguration: Sendable {
    let baseURL: URL
    let accessToken: String

    static func live(bundle: Bundle = .main) -> AppConfiguration {
        guard
            let rawURL = bundle.object(forInfoDictionaryKey: "API_BASE_URL") as? String,
            let baseURL = URL(string: rawURL),
            baseURL.scheme != nil
        else {
            preconditionFailure("API_BASE_URL is missing or invalid")
        }

        let accessToken = bundle.object(forInfoDictionaryKey: "APP_ACCESS_TOKEN") as? String ?? ""
        return AppConfiguration(baseURL: baseURL, accessToken: accessToken)
    }
}
