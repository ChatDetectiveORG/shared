// Package chaintest provides helpers for service chain tests:
// feed upstream inputs (Telegram update, AMQP delivery), capture downstream
// outgoing envelopes (message-sender queue), and assert end-to-end behavior.
//
// Typical flow:
//
//  1. Seed DB state with pgfixture (optional).
//  2. Build tele.Update or amqp.Delivery as api-gateway would publish.
//  3. Run endpoint/router with OutgoingCapture jobs + waiters.
//  4. Assert captured telegram.OutgoingRequest payloads and DB side effects.
package chaintest
