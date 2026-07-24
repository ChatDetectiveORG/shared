# Chain tests (mock integration)

Chain tests проверяют **цепочку** внутри микросервиса: вход (как от api-gateway / RabbitMQ) → обработка с подставной БД → выход (как в message-sender).

## Пакеты

| Пакет | Назначение |
|-------|------------|
| `shared/testutil/chaintest` | Захват исходящих `telegram.OutgoingRequest`, фикстуры `tele.Update` / AMQP delivery, запуск endpoint |
| `shared/testutil/pgfixture` | Подключение к тестовой Postgres, сидирование `telegramusers` / `messages` |

## Запуск локально

**Авто-подключение к dev-стенду:** если `CHAINTEST_DATABASE_URL` не задан, тесты пробуют
`postgres://chatdetective:chatdetective@127.0.0.1:5432/chatdetective?sslmode=disable`
(как в `values-local.yaml`). Нужен port-forward:

```bash
kubectl port-forward svc/chatdetective-dev-postgresql 5432:5432
```

Явный URL (опционально):

```bash
export CHAINTEST_DATABASE_URL='postgres://chatdetective:chatdetective@127.0.0.1:5432/chatdetective?sslmode=disable'
export MASTER_KEY='01234567890123456789012345678901'  # 32 bytes

cd shared && go test ./testutil/...
cd business-events-edited-handler && go test ./src/application/endpoints/editedMessage -run Chain
```

Если ни env, ни local postgres недоступны — chain-тесты **пропускаются** (`t.Skip`).

## Шаблон теста

```go
db := pgfixture.Open(t)
pgfixture.Reset(t, db)
postgresql.SetDBForTest(db)
t.Cleanup(postgresql.ResetDBForTest)

owner := pgfixture.SeedBotUser(t, db, pgfixture.BotUserSpec{...})
pgfixture.SeedBusinessMessage(t, db, owner, pgfixture.BusinessMessageSpec{...})

update := chaintest.EditedBusinessMessageUpdate(...)
capture := chaintest.NewOutgoingCapture(t, 8)
chaintest.RunEndpoint(t, myEndpoint, update, capture, "mirror-id")

requests := capture.Collect(500 * time.Millisecond)
chaintest.AssertAnyTextSubstr(t, requests, "expected", "fragments")
```

## Хуки в сервисах

- `handlers.Endpoint.RunForTest` — синхронный прогон endpoint с injected jobs/waiters
- `handlers.PublishEnvelope` — экспортированный тип исходящего job
- `<service>/infrastructure/postgresql.SetDBForTest` — подмена глобального `GetDB()` в тестах

При добавлении нового микросервиса скопируйте `SetDBForTest` / `ResetDBForTest` по образцу `business-events-edited-handler`.
