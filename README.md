# ПЗ №5 — PostgreSQL и database/sql

Сортов Семён Александрович, ЭФМО-01-26. Дата: 25.10.2026.

Программа подключается к PostgreSQL, создаёт задачи параметризованным INSERT и читает их через SELECT. Дополнительно реализованы ListDone, FindByID и массовая вставка CreateMany в транзакции. Измерение настроек пула находится в BenchmarkPool.

Требования: Go 1.25+, Docker Desktop. Проверено с Go 1.27.2 на Windows 11 amd64 и PostgreSQL 15. Пароль `postgres` относится к локальной учебной базе.

## Запуск из PowerShell

```powershell
docker compose up -d
# Дождаться готовности: accepting connections
docker compose exec postgres pg_isready -U postgres -d todo
# БД todo создаётся при первом старте контейнера.
Get-Content -Raw -Encoding UTF8 schema.sql | docker compose exec -T postgres psql -U postgres -d todo -v ON_ERROR_STOP=1
$env:DATABASE_URL='postgres://postgres:postgres@localhost:5435/todo?sslmode=disable'
go run .
docker compose exec postgres psql -U postgres -d todo -c 'SELECT * FROM tasks ORDER BY id;'
```

В PowerShell для корректной передачи кириллицы в psql перед выполнением команды с конвейером можно задать:

```powershell
$OutputEncoding = [System.Text.UTF8Encoding]::new()
```

Один запуск Go добавляет пять задач: три по одной и две в транзакции. При повторном запуске записи добавляются ещё раз; номера ID могут отличаться. schema.sql также вставляет одну демонстрационную строку.

## Проверки

```powershell
go test -v ./...
$env:GOMAXPROCS='8'
go test -run '^$' -bench BenchmarkPool -benchtime=800x -count=1
go vet ./...
go build -o bin/app.exe .
```

Интеграционный тест использует временную таблицу в отдельном соединении: проверяет выборки, кавычки в параметрах, отмену context и полный откат пакета при ошибке второй строки. Постоянная таблица не изменяется. Для теста и измерения нужна DATABASE_URL; без неё они пропускаются.

Пул: 10 открытых, 5 простаивающих соединений, срок соединения 30 минут. В замере на Ryzen 7 9800X3D при 8 параллельных исполнителях варианты 1/1, 5/2 и 10/5 дали примерно 0,587, 0,144 и 0,097 мс на операцию соответственно. У 10/5 ожиданий свободного соединения не было. Это результат локального короткого SELECT, а не универсальный показатель для любой нагрузки. Срок 30 минут выбран для периодического обновления соединений; этот короткий замер его не сравнивает.

## Типовые проблемы

- Connection refused: запустить Docker Desktop и дождаться готовности PostgreSQL.
- Ошибка пароля: проверить DATABASE_URL и локальные данные postgres/postgres.
- Таблица tasks отсутствует: выполнить schema.sql именно в БД todo.
- Нет DATABASE_URL: задать переменную в текущем окне PowerShell. .env.example служит образцом и автоматически не загружается.
