# День 08 — Работа с токенами

[← К списку заданий](../README.md) · [Главная страница проекта](../../README.md)

- **Дата:** 9 сентября 2026
- **Статус:** выполнено ✅
- **Зафиксированная версия:** [`day-08`](https://github.com/eugeneappledev-source/AI-Challenge/tree/day-08)

## Задание

Добавить агенту подсчёт токенов текущего запроса, всей истории и ответа; сравнить короткий, длинный и переполненный диалог; показать рост расхода и поведение при превышении лимита модели.

## Результат

В iOS появился отдельный **Token Lab**. После каждого реального сообщения он показывает:

- `prompt_tokens` последнего вызова — весь фактически отправленный контекст;
- `completion_tokens` последнего ответа;
- накопленный расход токенов по всем вызовам диалога;
- расчётную стоимость по раздельным input/cache/output usage;
- локальную оценку размера сохранённой истории и последнего user message;
- короткий, длинный и переполненный сценарии рядом.

## Что считается точно, а что оценивается

| Значение | Источник | Точность |
|---|---|---|
| Prompt и completion после вызова | `usage` ответа DeepSeek | Точное значение провайдера |
| Накопленный расход | Сумма сохранённых `usage` в SQLite | Точное для завершённых вызовов |
| Стоимость | Usage × опубликованный тариф | Расчётная |
| Токены текста до отправки | Локальная эвристика | Приблизительное значение |

Такое разделение сделано намеренно: без официального токенизатора нельзя выдавать локальный подсчёт за точный. Источником истины после запроса остаётся `usage` провайдера.

Для стоимости используется консервативный peak-тариф `deepseek-flash`: cache hit `$0.006`, cache miss `$0.30`, output `$1.20` за миллион токенов. [Актуальная таблица DeepSeek](https://api-docs.deepseek.com/quick_start/pricing/).

## Переполнение

Контекст `deepseek-flash` ограничен 1 000 000 токенов. Экран показывает демонстрационный payload размером `limit + 1` и результат `context_length_exceeded`. Агент останавливает такой сценарий на preflight-проверке до платного вызова: пользователь видит, что ломается, но тест не расходует миллион токенов.

## API

```http
GET /v1/agent/tokens?conversationId=ios-day08-...
Authorization: Bearer <access-token>
```

Ключевые поля ответа:

```json
{
  "currentMessageEstimatedTokens": 18,
  "lastContextPromptTokens": 143,
  "lastResponseTokens": 52,
  "cumulativeTotalTokens": 417,
  "estimatedCostUSD": 0.000104,
  "contextWindowTokens": 1000000,
  "scenarios": ["short", "long", "overflow"]
}
```

## Как проверить

1. Открыть **День 8 · Работа с токенами**.
2. Отправить короткий запрос и сохранить метрики.
3. Отправить несколько развёрнутых запросов — `prompt_tokens` и cumulative расход растут вместе с историей.
4. Сравнить карточки Short / Long / Overflow и показать безопасную блокировку переполнения.

## Код

- [Расчёт метрик в Agent](../../backend/internal/application/agent.go)
- [Хранение usage в SQLite](../../backend/internal/infrastructure/sqlite/conversation_store.go)
- [Token Lab на SwiftUI](../../ios/AIChallenge/Features/TokenLab/Views/TokenLabScreen.swift)
- [Use case](../../ios/AIChallenge/Domain/UseCases/InspectAgentTokensUseCase.swift)
- [Тест метрик](../../backend/internal/application/agent_test.go)
