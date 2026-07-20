// Package messageBuilder helps assemble formatted Telegram messages with optional
// MarkdownV2 or entity-based styling, inline keyboards, media attachments and
// mirror-aware static assets.
//
// Intended for dot-import in handlers:
//
//	import . "github.com/ChatDetectiveORG/shared/messageBuilder"
//
//	mb := MessageBuilder{Mdv2Enabled: true}
//	mb.WriteSlice([]Text{
//		T("Hello"),
//		B(T("world")),
//	})
//	msg := mb.Build(chatID)
package messageBuilder
