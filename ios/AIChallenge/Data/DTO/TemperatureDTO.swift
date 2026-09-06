struct RunTemperatureRequestDTO: Encodable {
    let prompt: String
    let temperature: TemperaturePreset
}

struct ReviewTemperatureRequestDTO: Encodable {
    let prompt: String
    let attempts: [TemperatureAttempt]
}
