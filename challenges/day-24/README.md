# День 24 — Ответы с доказательствами

## Задача

Возвращать вместе с ответом источники и цитаты из найденных документов. При низкой уверенности не выдумывать ответ, а явно сообщать о недостатке контекста. Проверить поведение на 10 вопросах.

## Реализация

Evidence-first pipeline состоит из пяти проверяемых этапов:

1. query rewrite и retrieval;
2. hybrid reranking и threshold;
3. confidence gate до вызова генерации;
4. LLM-ответ только по прошедшим фрагментам;
5. серверные citations с дословными excerpt из chunks.

Модель не генерирует цитаты и пути файлов. Backend выбирает максимум три опоры, сохраняет `source`, `section`, `chunk_id`, score и проверяет, что quote действительно является подстрокой исходного chunk. Это исключает «красивые», но выдуманные ссылки.

Если ни один фрагмент не проходит threshold, возвращается статус `insufficient_context`; вызов генерации ответа пропускается и расход токенов равен нулю.

Контрольная матрица прогоняет 10 вопросов и отдельно проверяет ожидаемый источник, минимальный score и валидность цитаты.

## Как записать видео

1. Открыть [День 24](https://176-53-173-246.sslip.io/#day-24).
2. Задать вопрос о проекте и оставить threshold `0.08`.
3. Нажать **«Получить ответ с источниками»**.
4. Показать confidence, ответ, пути и цитаты с отметкой `VERBATIM`.
5. Увеличить threshold до высокого значения и повторить — появится честный отказ.
6. Показать, что при отказе model помечена как `generation skipped`, а tokens равны нулю.
7. Нажать **«Повторить 10 проверок»** и показать матрицу PASS/CHECK.

## Код

- [Confidence gate и citations](../../backend/internal/application/rag_service.go)
- [Evidence domain models](../../backend/internal/domain/rag.go)
- [Evidence API](../../backend/internal/transport/http/handler.go)
- [Evidence UI](../../web/src/KnowledgeLab.tsx)
- [Citation/refusal tests](../../backend/internal/application/rag_service_test.go)
