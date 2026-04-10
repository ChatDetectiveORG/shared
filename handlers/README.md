# Пакет `handlers`

Короткая схема: **Router** получает `tele.Update` и прогоняет его через **Endpoint**-ы (фильтр → цепочка хендлеров). Исходящие сообщения в Telegram — через RabbitMQ: в exchange публикуется **JSON `tele.Message`**.

## Основные типы

| Компонент | Файл | Роль |
|-----------|------|------|
| **Router** | `dispatcher.go` | Список endpoint-ов, `Dispatch(update)`, опционально шардирование (`InitSharding` + `HandleUpdate` с заголовком `session_id`). |
| **Endpoint** | `endpoint.go` | Имя, `HandlerChain`, `UpdateFilter`. Создаётся через `Init(name, chain, filter)`. |
| **HandlerChain** | `handlerChain.go` | Последовательность шагов с таймаутом; внутри прогона доступен **HandlerChainHashe**. |
| **HandlerChainHashe** | `handlerHashe.go` | Контекст одного прогона: `Add`/`Get`/`Trunc`, `Emit`, `EmitWait`, `RunID()`. |
| **UpdateFilter** | `filters.go` | `Command`, `TextCommand`, `CallbackQueryJSON`, `BusinessEvent`, `And` / `Or`. |
| **Исходящая почта** | `outgoing.go` | Логика `Router.StartOutgoing`: очередь результатов, публикация, consumer. |

## Поток данных

1. Сервис unmarshaling-ит `tele.Update` и вызывает **`router.Dispatch(update)`**.
2. Для каждого endpoint, если фильтр пропускает апдейт, запускается **`HandlerChain.Run`**.
3. Хендлеры (`chainHandlerFunc`) получают `(update, hashe *HandlerChainHashe)` и возвращают `e.ErrorInfo`.

## Исходящие сообщения (AMQP)

- **Включение:** после готовности канала RabbitMQ один раз вызвать **`router.StartOutgoing(wg, podID, shardID)`** (тот же `podID` / шард, что и для биндинга очередей апдейтов, если они связаны).
- **Тело сообщения:** `json.Marshal(tele.Message)`.
- **Routing key:** первый аргумент `Emit` / `EmitWait` (например, идентификатор бота для маршрутизации у получателя).
- **CorrelationId** и заголовок `correlation_id` выставляются пакетом для связки с ответом.

### `Emit` vs `EmitWait`

- **`Emit(routingKey, msg)`** — только публикация, без ожидания ответа от message-sender.
- **`EmitWait(ctx, routingKey, msg)`** — ждёт JSON **`SendResult`** (см. `handlerResponse.go`) с тем же `correlation_id`, что ушёл в AMQP.

Когда обратная связь **не нужна**: ответ пользователю на его сообщение — заполняй **`ReplyTo`** в `tele.Message` из входящего апдейта; ID только что отправленного ботом сообщения не требуется.

Когда нужен **`EmitWait`**: сценарии, где Telegram должен вернуть `message_id` отправленного бота сообщения (ответ на своё сообщение, правка и т.д.).

## Шардирование внутри Router (опционально)

- **`Router.ReplicaCount` > 0** и **`InitSharding(wg, ctx)`** — создаются каналы `endpoint-00` … и воркеры, читающие из них.
- **`HandleUpdate(delivery)`** — читает `session_id` из headers, unmarshaling-ит `tele.Update`, кладёт в канал шарда по хешу `session_id`.

Если шардирование не используется, достаточно вызывать только **`Dispatch`** (как в типичном per-session горутине сервиса).

## Настройка exchange (Router)

Поля (пустая строка → значение по умолчанию в коде):

- **`OutgoingExchange`** — куда публикуется исходящее `tele.Message` (по умолчанию `chatdetective.output.send`).
- **`SendResultExchange`** — откуда очередь результатов получает ответы (по умолчанию `chatdetective.send.result`).

## Ошибки

Серьёзные сбои цепочки уходят в **`Router.ErrorChannel`**, если он задан. Статус **`e.Ingnored`** при прогоне цепочки не считается фатальной ошибкой для `Run`.
