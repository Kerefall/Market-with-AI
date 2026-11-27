# ShopAI backend

Backend платформы интернет-магазинов с ИИ-помощником. Go, chi, PostgreSQL, JWT.

## Запуск

    docker compose up --build

API: http://localhost:8080, документация Swagger: http://localhost:8080/docs

## Проверка

    bash scripts/smoke.sh

## Настройки окружения (.env)

| Переменная | Назначение |
|---|---|
| `JWT_SECRET` | Секрет подписи токенов, не короче 16 символов (генерация: `openssl rand -hex 32`) |
| `LLM_API_URL`, `LLM_API_KEY`, `LLM_MODEL` | ИИ-провайдер (OpenAI-совместимый API). Без них работает демо-режим |
| `CLOUDPAYMENTS_PUBLIC_ID`, `CLOUDPAYMENTS_API_SECRET` | Оплата через CloudPayments. Без них работает тестовая оплата |
| `CORS_ORIGIN` | Адрес фронтенда (по умолчанию `*`) |

Файл `.env` не коммитьте: в нём ключи.

## Частые проблемы

- **Порт 8080 занят** (`address already in use`, ответ «502 nginx»): остановите программу, которая его держит, или смените левый порт в `docker-compose.yml`, например `"8081:8080"`.
- **`/api/ai/*` отвечает 502**: смотрите `docker compose logs api`, туда пишется причина ошибки провайдера.
