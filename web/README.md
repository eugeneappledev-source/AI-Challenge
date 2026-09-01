# Web client

Адаптивная web-версия AI Challenge Day 2. Она отправляет один вопрос в двух режимах, сравнивает свободный ответ с контролируемым JSON и отображает структурированный результат как карточку рецепта.

## Локальный запуск

```bash
pnpm install
pnpm dev
```

В development-режиме Vite проксирует `/web-api` на локальный backend по адресу `http://localhost:8080`. В production запросы маршрутизирует Caddy на VPS.

## Production-сборка

```bash
pnpm build
```

Клиент не хранит ключ DeepSeek или общий токен нативного API. Авторизацию публичного web-маршрута добавляет Caddy внутри приватной Docker-сети.
