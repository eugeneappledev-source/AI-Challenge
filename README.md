# AI Advent Challenge #9

### Практический путь от первого запроса к LLM до полноценного AI-продукта

[![Backend CI](https://github.com/eugeneappledev-source/AI-Challenge/actions/workflows/backend.yml/badge.svg)](https://github.com/eugeneappledev-source/AI-Challenge/actions/workflows/backend.yml)
[![Web CI](https://github.com/eugeneappledev-source/AI-Challenge/actions/workflows/web.yml/badge.svg)](https://github.com/eugeneappledev-source/AI-Challenge/actions/workflows/web.yml)
![Swift](https://img.shields.io/badge/Swift-6.0-F05138?logo=swift&logoColor=white)
![Go](https://img.shields.io/badge/Go-1.27-00ADD8?logo=go&logoColor=white)
![React](https://img.shields.io/badge/React-19-61DAFB?logo=react&logoColor=white)
![DeepSeek](https://img.shields.io/badge/LLM-DeepSeek-4D6BFE)

Этот репозиторий — мой практический дневник **AI Advent Challenge, поток 9**. Здесь одно приложение постепенно развивается вместе с заданиями курса: от минимальной интеграции с облачной моделью до более сложной архитектуры, инфраструктуры и пользовательских сценариев.

## Прогресс

| Неделя | День | Задание | Результат | Статус |
|:---:|:---:|---|---|:---:|
| 1 | [01](challenges/day-01/README.md) | Первый запрос к облачной LLM | SwiftUI + Go + DeepSeek + VPS | ✅ |
| 1 | [02](challenges/day-02/README.md) | Управление форматом ответа | Сравнение свободного и контролируемого ответов | ✅ |
| 1 | [03](challenges/day-03/README.md) | Разные способы рассуждения | Четыре стратегии решения и AI-арбитр | ✅ |
| 1 | [04](challenges/day-04/README.md) | Температура генерации | Три значения temperature и AI-рецензент | ✅ |
| 1 | [05](challenges/day-05/README.md) | Версии моделей | Три модели: качество, скорость, токены и стоимость | ✅ |
| 2 | [06](challenges/day-06/README.md) | Первый агент | Отдельная сущность Agent и SwiftUI-интерфейс | ✅ |
| 2 | [07](challenges/day-07/README.md) | Сохранение контекста | SQLite и восстановление истории | ✅ |
| 2 | 08 | Работа с токенами | Метрики контекста, ответа и стоимости | В работе |
| 2 | 09 | Сжатие истории | Summary + последние N сообщений | В работе |

Подробная навигация по выполненным заданиям находится в [дневнике челленджа](challenges/README.md).

## Быстрая проверка

Репозиторий построен как один развивающийся продукт, поэтому проверяющему не нужно искать отдельный проект для каждого дня:

1. открыть нужный день в таблице прогресса выше;
2. прочитать результат, условия эксперимента и способ проверки;
3. перейти по ссылкам из раздела **«Код»** к конкретным iOS- и backend-файлам;
4. при необходимости открыть тег `day-XX` — он сохраняет проект ровно в состоянии соответствующей сдачи.

Дни не перезаписывают друг друга: в iOS-приложении каждый сценарий открывается отдельным экраном из общего каталога, а обратная совместимость API отмечена в описаниях заданий.

## Текущая версия проекта

Нативное iOS-приложение открывается каталогом заданий. Первая неделя исследует базовый API-вызов, управляемый JSON, способы рассуждения, `temperature` и версии моделей. Со второй недели проект переходит к агентной архитектуре: День 6 добавляет самостоятельного агента Compass, а следующие задания развивают его память и управление контекстом. Адаптивная web-версия сохраняет сценарий Дня 2 с пищевым ассистентом.

Проект развёрнут на VPS и доступен по HTTPS без регистрации.

- **Web-приложение:** [https://176-53-173-246.sslip.io](https://176-53-173-246.sslip.io)
- **Health check:** [https://176-53-173-246.sslip.io/health](https://176-53-173-246.sslip.io/health)

## Архитектура

```mermaid
flowchart LR
    User["Пользователь"] --> App["iOS · SwiftUI"]
    User --> Web["Web · React"]
    App -->|"HTTPS · REST"| Gateway["Caddy · VPS"]
    Web -->|"HTTPS · same-origin"| Gateway
    Gateway --> API["Go backend"]
    API --> LLM["DeepSeek API"]
    LLM --> API
    API --> Gateway
    Gateway --> App
    Gateway --> Web
```

## Структура

```text
AI-Challenge/
├── ios/                         # iOS-приложение
├── web/                         # адаптивное React-приложение
├── backend/                     # Go REST API
├── deploy/                      # Docker Compose и Caddy
├── challenges/                  # дневник выполненных заданий
│   ├── day-01/                  # первый запрос к LLM
│   ├── day-02/                  # контроль формата и длины ответа
│   ├── day-03/                  # четыре стратегии рассуждения
│   ├── day-04/                  # эксперимент с temperature
│   ├── day-05/                  # сравнение версий моделей
│   ├── day-06/                  # отдельная сущность первого агента
│   └── day-07/                  # постоянная память в SQLite
└── .github/workflows/           # автоматические проверки
```

Рабочий код остаётся в стабильных каталогах `ios`, `backend` и `deploy`, а каждый новый день получает отдельную страницу в `challenges`. Благодаря этому проект может последовательно развиваться без копирования одинаковых исходников.

## Технологии

### iOS

- Swift 6 и SwiftUI;
- Observation и Swift Concurrency;
- URLSession;
- слои `Domain / Data / Presentation`;
- Swift Testing;
- iOS 17+.

### Backend и инфраструктура

- Go и `net/http`;
- DeepSeek Chat Completions API;
- Docker Compose;
- Caddy и HTTPS;
- Ubuntu VPS;
- GitHub Actions.

### Web

- React, TypeScript и Vite;
- адаптивный интерфейс для desktop и mobile;
- разбор контролируемого JSON в пользовательскую карточку;
- публичный same-origin endpoint без секретов в браузере;
- серверные ограничения частоты и дневного числа запросов.

## Задания

Каждая завершённая работа получает:

- отдельное описание в `challenges/day-XX`;
- ссылки на относящиеся к заданию части проекта;
- зафиксированный результат и способ проверки;
- Git-тег, сохраняющий состояние проекта на момент сдачи.

Зафиксированные версии заданий:

- [`day-01`](https://github.com/eugeneappledev-source/AI-Challenge/tree/day-01) — первый запрос к облачной LLM;
- [`day-02`](https://github.com/eugeneappledev-source/AI-Challenge/tree/day-02) — управление форматом, длиной и завершением ответа;
- [`day-03`](https://github.com/eugeneappledev-source/AI-Challenge/tree/day-03) — сравнение четырёх способов рассуждения;
- [`day-04`](https://github.com/eugeneappledev-source/AI-Challenge/tree/day-04) — сравнение точности, креативности и разнообразия при трёх температурах;
- [`day-05`](https://github.com/eugeneappledev-source/AI-Challenge/tree/day-05) — сравнение качества, скорости, токенов и стоимости трёх моделей.
- [`day-06`](https://github.com/eugeneappledev-source/AI-Challenge/tree/day-06) — отдельная сущность агента с инкапсулированной логикой LLM-вызова.
- [`day-07`](https://github.com/eugeneappledev-source/AI-Challenge/tree/day-07) — сохранение истории в SQLite и восстановление контекста.
