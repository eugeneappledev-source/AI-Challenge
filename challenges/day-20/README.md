# День 20 — Оркестрация нескольких MCP-серверов

## Задача

Подключить несколько MCP-серверов с разными зонами ответственности и научить агента выбирать правильный server/tool, соблюдать порядок зависимостей и передавать данные между серверами.

## Реализация

Оркестратор работает с двумя независимыми Streamable HTTP endpoint:

- `startup-research-mcp` — `search_business_news`, `summarize_business_news`, `save_business_report`;
- `business-advisor-mcp` — `score_business_opportunities`, `build_action_plan`.

Перед выполнением агент отдельно запрашивает `tools/list` у каждого сервера и передаёт модели фактически доступные схемы. Модель строит маршрут, а backend проверяет server, tool и допустимый порядок.

Рабочий маршрут:

1. Research MCP загружает актуальные источники;
2. Research MCP превращает их в brief;
3. brief и профиль пользователя передаются Advisor MCP;
4. Advisor ранжирует гипотезы и строит ограниченный план проверки;
5. готовый research report сохраняется через Research MCP.

Профиль содержит бюджет, часы в неделю, навыки и уровень риска. Поэтому оценка показывает не абстрактно «лучшую идею», а её fit относительно ресурсов пользователя. Все советы остаются гипотезами для проверки и не обещают заработок.

## Как записать видео

1. Открыть [День 20](https://176-53-173-246.sslip.io/#day-20).
2. Ввести задачу или подставить сценарий, затем изменить бюджет/часы/навыки.
3. Нажать **«Запустить оркестратор»** и показать индикатор двух серверов.
4. Показать карточки `startup-research-mcp` и `business-advisor-mcp` с разными наборами tools.
5. Пройти маршрут из пяти вызовов и обратить внимание на переключение цвета сервера.
6. Показать рейтинг с Fit Score, основаниями и рисками.
7. Завершить недельным action plan, budget guard, stop conditions и ID сохранённого отчёта.

## Код

- [Multi-server orchestrator](../../backend/internal/application/business_network.go)
- [Research MCP server](../../backend/internal/infrastructure/mcpserver/research.go)
- [Advisor MCP server](../../backend/internal/infrastructure/mcpserver/advisor.go)
- [Регистрация двух endpoint](../../backend/cmd/api/main.go)
- [React MCP Network Lab](../../web/src/MCPNetworkLab.tsx)
- [Интеграционный тест двух серверов](../../backend/internal/application/business_network_test.go)
