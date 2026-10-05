# link-checker

[![CI](https://github.com/denagava/link-checker/actions/workflows/ci.yml/badge.svg)](https://github.com/denagava/link-checker/actions/workflows/ci.yml)
![Go](https://img.shields.io/badge/Go-1.23-00ADD8?logo=go)

HTTP-сервис на Go, который параллельно проверяет доступность списка ссылок, сохраняет результаты и формирует по ним PDF-отчёт.

## Возможности

- **Параллельная проверка** с ограничением одновременных запросов (worker-пул на семафоре, по умолчанию 20).
- **Таймауты** на каждый запрос (10 с) и на весь пакет (30 с); при отмене запроса клиентом проверки прерываются.
- **Сохранение результатов** в JSON-файл с атомарной записью (temp + rename): после перезапуска данные и нумерация продолжаются.
- **PDF-отчёт** по одному или нескольким наборам ссылок, с сортировкой и цветовой индикацией статуса.
- **Graceful shutdown**: по `Ctrl+C` / `SIGTERM` сервер перестаёт принимать запросы и дожидается завершения начатых проверок.
- **Валидация входа**: лимит 100 ссылок на запрос и 1 МБ на тело.

## Запуск

```bash
go run . -addr :8080 -data links_data.json
```

Или в Docker:

```bash
docker build -t link-checker .
docker run --rm -p 8080:8080 -v "$PWD/data:/data" link-checker
```

## API

### `POST /check` — проверить ссылки

```bash
curl -X POST http://localhost:8080/check \
  -H "Content-Type: application/json" \
  -d '{"links": ["google.com", "https://github.com", "invalid-test-domain-12345.com"]}'
```

```json
{
  "links": {
    "google.com": "available",
    "https://github.com": "available",
    "invalid-test-domain-12345.com": "not available"
  },
  "links_num": 1
}
```

Ссылка считается доступной, если сервер ответил кодом меньше 400. Домены без схемы проверяются по `http://`. `links_num` — ID сохранённого набора для отчёта.

### `POST /report` — получить PDF-отчёт

```bash
curl -X POST http://localhost:8080/report \
  -H "Content-Type: application/json" \
  -d '{"links_list": [1, 2]}' \
  --output report.pdf
```

### Ошибки

| Код | Когда |
|---|---|
| `400` | невалидный JSON, пустой список, больше 100 ссылок |
| `404` | ни одного набора с указанными ID |
| `405` | метод отличается от POST |

Тело ошибки: `{"error": "описание"}`.

## Тесты

```bash
go test -race ./...
```

Проверяются: нормализация URL, статусы доступности на локальных тестовых серверах (200 / 404 / 500 / недоступный хост), **соблюдение лимита параллельности**, отмена по контексту, сохранение и восстановление данных после перезапуска, полный сценарий `/check` → `/report` с проверкой, что ответ — PDF, и валидация запросов.

## Структура

```
main.go      запуск, флаги, graceful shutdown
server.go    HTTP-хендлеры и валидация
checker.go   параллельная проверка ссылок
storage.go   хранение наборов с записью на диск
report.go    генерация PDF
```

## Стек

Go · net/http · [gofpdf](https://github.com/jung-kurt/gofpdf) · Docker · GitHub Actions
