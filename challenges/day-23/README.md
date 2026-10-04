# День 23 — Улучшение retrieval

## Задача

Улучшить качество поиска: добавить reranker или фильтрацию по score, управляемые `threshold` и `top-K`, сравнить retrieval до и после улучшения. Дополнительно проверить query rewriting.

## Реализация

Экран выполняет один вопрос двумя способами:

- **Baseline:** исходный вопрос → embedding → cosine similarity → top-K;
- **Improved:** LLM query rewrite → расширенный пул кандидатов → hybrid reranker → threshold → top-K.

Reranker объясним и воспроизводим:

```text
score = 0.72 × vector similarity
      + 0.23 × lexical overlap
      + 0.05 × metadata match
```

Пользователь сам меняет `top-K` от 1 до 10 и `threshold` от 0 до 0.50. UI показывает переписанный запрос, две итоговые выдачи, количество отброшенных фрагментов и аудит всех кандидатов до фильтра.

Модель используется только для query rewrite. Ранжирование и порог выполняются на backend, поэтому итог можно проверить по числам.

## Как записать видео

1. Открыть [День 23](https://176-53-173-246.sslip.io/#day-23).
2. Оставить подготовленный вопрос и выбрать `top-K = 5`, `threshold = 0.12`.
3. Нажать **«Сравнить retrieval»**.
4. Показать исходный и переписанный поисковый запрос.
5. Сравнить списки **Baseline** и **Improved**.
6. Поднять threshold и повторить — число прошедших фрагментов уменьшится.
7. Раскрыть аудит кандидатов и показать rerank score.

## Код

- [Query rewrite, reranker и filter](../../backend/internal/application/rag_service.go)
- [Retrieval models](../../backend/internal/domain/rag.go)
- [API endpoint](../../backend/internal/transport/http/handler.go)
- [Interactive retrieval lab](../../web/src/KnowledgeLab.tsx)
- [Reranking test](../../backend/internal/application/rag_service_test.go)
