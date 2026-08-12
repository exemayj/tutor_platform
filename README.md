# Tutor Platform

Сервис для поиска репетиторов. Бесплатный для пользователей.

## Функциональность

- Регистрация с ролями (ученик / репетитор / админ)
- Анкеты репетиторов с предметами, ценой, городом
- Каталог с фильтрами и сортировкой
- Заявки от учеников (создание, принятие, отклонение, завершение)
- Чат между учеником и репетитором в реальном времени (WebSocket)
- Рейтинг и отзывы
- Админ-панель для модерации (блокировка репетиторов)

## Технологии

- **Backend:** Go, Chi (роутер)
- **База данных:** PostgreSQL, pgx
- **Шаблоны:** templ
- **WebSocket:** gorilla/websocket
- **Аутентификация:** JWT, bcrypt

## Установка и запуск

### Требования

- Go 1.21+
- PostgreSQL 15+
- templ CLI

### Установка

\`\`\`bash
# Клонировать репозиторий
git clone https://github.com/exemayj/tutor_platform.git
cd tutor_platform

# Установить зависимости
go mod tidy

# Установить templ
go install github.com/a-h/templ/cmd/templ@latest
\`\`\`

### Настройка базы данных

\`\`\`bash
# Создать базу
createdb tutor_platform

# Применить миграции
psql -d tutor_platform -f internal/database/migrations/000001_init.up.sql

# Добавить тестовые предметы
psql -d tutor_platform
INSERT INTO subjects (name, slug) VALUES
('Математика', 'matematika'),
('Русский язык', 'russkii-yazyk'),
('Английский язык', 'angliiskii-yazyk'),
('Физика', 'fizika'),
('Информатика', 'informatika');
\q
\`\`\`

### Настройка окружения

Создать файл `.env`:

\`\`\`
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=
DB_NAME=tutor_platform
JWT_SECRET=your-secret-key
SERVER_PORT=8080
\`\`\`

### Запуск

\`\`\`bash
templ generate
go run ./cmd/server/
\`\`\`

Открыть `http://localhost:8080/`

## Структура проекта

\`\`\`
tutor_platform/
├── cmd/server/          # точка входа
├── internal/
│   ├── config/          # конфигурация
│   ├── database/        # подключение к БД
│   ├── handlers/        # HTTP-хендлеры
│   ├── middleware/      # JWT, AdminOnly
│   └── models/          # модели данных
├── web/
│   ├── static/          # CSS, JS
│   └── templates/       # templ-шаблоны
└── .env.example
\`\`\`

## Лицензия

MIT