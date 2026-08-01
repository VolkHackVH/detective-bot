# Detective Bot

Discord-бот для подачи и проверки улик с сохранением данных в SQLite.

## Возможности

- Подача улики через modal-форму.
- Публикация улики в отдельном канале.
- Упоминание ролей проверяющих.
- Статусы: ожидает проверки, на рассмотрении, принято и отклонено.
- Уведомления автора в личных сообщениях.
- Запуск локально или через Docker.

## Настройка

### 1. Discord

В [Discord Developer Portal](https://discord.com/developers/applications) включите:

```text
Bot → Privileged Gateway Intents → Message Content Intent
```

Боту нужны права на просмотр канала, отправку сообщений, embed и упоминание настроенных ролей.

### 2. Переменные окружения

Создайте `.env`:

```dotenv
DISCORD_TOKEN=your_discord_bot_token
DATABASE_PATH=./data/evidence.db
```

### 3. Конфигурация

Скопируйте пример:

```bash
cp config.example.json config.json
```

Укажите Discord ID строками в кавычках:

```json
{
  "discord": {
    "guild_id": "111111111111111111"
  },
  "evidence": {
    "intake_channel_id": "222222222222222222",
    "review_channel_id": "333333333333333333",
    "reviewer_role_ids": [
      "444444444444444444",
      "555555555555555555"
    ]
  }
}
```

- `intake_channel_id` — канал, в котором создаётся панель подачи улик.
- `review_channel_id` — канал проверки улик.
- `reviewer_role_ids` — роли, которые упоминаются и могут обрабатывать улики.

## Запуск в Docker

```bash
docker compose up --build -d
```

SQLite хранится в Docker volume и сохраняется после пересоздания контейнера. Команда `docker compose down -v` удалит базу вместе с volume.

## Локальный запуск

Требуется Go 1.25 или новее.

```bash
go mod download
go run ./cmd/bot
```

## Использование

1. Администратор отправляет `!улика` в канале подачи улик.
2. Бот публикует сообщение с кнопкой `Подать улику`.
3. Пользователь заполняет форму.
4. Улика сохраняется в SQLite и публикуется в канале проверки.
5. Проверяющий берёт её в работу и принимает либо отклоняет.
-----

[LICENSE](LICENSE)
