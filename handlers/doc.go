// Package handlers — роутинг tele.Update по endpoint-ам (фильтры + цепочки) и публикация исходящих
// tele.Message в RabbitMQ exchange.
//
// Исходящие сообщения: тело AMQP = JSON tele.Message; routing key задаётся вызовом Emit(routingKey, msg).
// Обратная связь от message-sender не обязательна: если второе сообщение — ответ пользователю на его же
// апдейт, достаточно заполнить msg.ReplyTo (и Chat) из входящего update — тогда не нужен ID сообщения бота.
// EmitWait нужен, когда от Telegram требуется message_id уже отправленного ботом сообщения
// (ответ боту самому себе, редактирование и т.д.): message-sender должен вернуть JSON SendResult
// с тем же correlation_id, что в AMQP CorrelationId (или в поле correlation_id в теле).
package handlers
