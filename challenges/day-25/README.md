# День 25 — Диалоговый RAG с памятью задачи

## Задача

Объединить историю диалога, RAG, источники и память текущей задачи. Агент должен продолжать длинный разговор, помнить цель и договорённости после перезапуска и быть проверен на двух сценариях по 10–15 сообщений.

## Реализация

RAG-агент хранит в SQLite два независимых слоя:

- **chat history** — user/assistant messages, citations, token usage и timestamp;
- **task memory** — цель, последние важные ограничения и словарь терминов.

На каждом ходе агент:

1. загружает сохранённую историю и task memory;
2. обновляет цель, ограничения и определения из новой реплики;
3. строит retrieval query из задачи и текущего вопроса;
4. выполняет rewrite, reranking и confidence filter;
5. передаёт LLM task memory, последние восемь сообщений и найденные chunks;
6. сохраняет обе новые реплики вместе с проверенными sources.

Session ID создаётся отдельно для браузера и хранится в `localStorage`, поэтому разные пользователи не делят один разговор. История и задача находятся на backend и восстанавливаются после reload страницы.

В интерфейсе подготовлены два длинных сценария:

- **«Релиз Knowledge Studio»** — 12 сообщений о цели, бюджете, совместимости и критериях готовности;
- **«Техническое ревью RAG»** — 11 сообщений об архитектуре, evidence gate, рисках и улучшениях.

## Как записать видео

1. Открыть [День 25](https://176-53-173-246.sslip.io/#day-25).
2. Нажать **«Очистить историю»** для чистой демонстрации.
3. Выбрать первый сценарий и отправлять подготовленные реплики кнопкой **«Следующая реплика»**.
4. После нескольких ходов показать, как заполняются goal, constraints и terms.
5. Показать sources под ответами и счётчик токенов.
6. Перезагрузить страницу: сообщения и task memory должны восстановиться.
7. Очистить историю, выбрать второй сценарий и повторить проверку независимого диалога.

## Проверки

- application-тест подтверждает накопление history/task memory и очистку;
- SQLite-тест закрывает и повторно открывает базу, затем проверяет восстановление данных;
- последние сообщения и явная память передаются модели раздельно;
- citations формируются backend из retrieved chunks.

## Код

- [Conversational RAG agent](../../backend/internal/application/rag_service.go)
- [SQLite chat/task store](../../backend/internal/infrastructure/sqlite/rag_chat_store.go)
- [Database migration](../../backend/internal/infrastructure/sqlite/conversation_store.go)
- [Chat domain models](../../backend/internal/domain/rag.go)
- [Persistent chat UI](../../web/src/KnowledgeLab.tsx)
- [Agent tests](../../backend/internal/application/rag_service_test.go)
- [SQLite reopen test](../../backend/internal/infrastructure/sqlite/rag_chat_store_test.go)
