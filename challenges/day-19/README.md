# День 19 — Цепочка MCP-инструментов

## Задача

Научить агента решать одну задачу несколькими инструментами: самостоятельно определить порядок вызовов, передать результат одного tool следующему и получить общий результат.

## Реализация

Один `startup-research-mcp` регистрирует три инструмента с разными контрактами:

1. `search_business_news` получает актуальные новости из внешнего API;
2. `summarize_business_news` принимает структурированный массив stories и создаёт краткий brief;
3. `save_business_report` принимает готовый brief и сохраняет отчёт в SQLite.

Перед выполнением агент запрашивает `tools/list` и отдаёт модели реальные названия, descriptions и JSON Schema. Модель возвращает план. Backend не доверяет плану вслепую: проверяет доступность инструментов и допустимый порядок, после чего последовательно выполняет `tools/call`.

Интерфейс показывает:

- план агента и его объяснение;
- вход, выход и длительность каждого этапа;
- явный handoff данных между tools;
- итоговую сводку и ID сохранённого отчёта.

## Как записать видео

1. Открыть [День 19](https://176-53-173-246.sslip.io/#day-19).
2. Ввести собственную исследовательскую задачу или подставить пример.
3. Нажать **«Запустить pipeline»** и показать индикатор `tools/list → plan → tool/call × 3`.
4. После ответа показать план `search → summarize → save` и объяснение модели.
5. По очереди показать INPUT/OUTPUT трёх этапов — особенно передачу stories и brief.
6. В конце показать сохранённый report ID и сделать вывод: инструменты не вызываются вручную из браузера, ими управляет агент.

## Код

- [Оркестратор цепочки](../../backend/internal/application/business_pipeline.go)
- [Три инструмента одного MCP-сервера](../../backend/internal/infrastructure/mcpserver/research.go)
- [SQLite persistence](../../backend/internal/infrastructure/sqlite/radar_store.go)
- [Доменные модели pipeline](../../backend/internal/domain/mcp.go)
- [React Tool Pipeline Lab](../../web/src/ToolPipelineLab.tsx)
- [Интеграционный тест цепочки](../../backend/internal/application/business_pipeline_test.go)
