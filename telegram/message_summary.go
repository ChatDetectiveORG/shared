package telegram

import (
	"strings"
	"time"
	"unicode/utf16"

	tele "gopkg.in/telebot.v4"
)

const (
	summaryLinkURL    = "https://www.google.com/"
	maxSummaryTextLen = 3900
)

func HideSendOptsIntoMessage(msg *tele.Message, sendOptions *tele.SendOptions) *tele.Message {
	msg.ReplyTo = sendOptions.ReplyTo
	msg.ReplyMarkup = sendOptions.ReplyMarkup
	msg.PreviewOptions.Disabled = sendOptions.DisableWebPagePreview
	
	msg.Entities = sendOptions.Entities
	msg.Protected = sendOptions.Protected
	msg.HasMediaSpoiler = sendOptions.HasSpoiler
	msg.EffectID = sendOptions.EffectID

	return msg
}

type summaryTextBuilder struct {
	sb       strings.Builder
	entities tele.Entities
	utf16Len int
}

type replySummary struct {
	hasReply    bool
	quoted      bool
	source      string
	kind        string
	quoteText   string
	previewText string
}

type forwardSummary struct {
	user       string
	chat       *tele.Chat
	hidden     bool
	senderName string
	date       string
	signature  string
	automatic  bool
}

// BuildMessageSummary собирает человекочитаемую сводку по сообщению
// и возвращает текст вместе с send options для ссылок в Telegram entities.
func BuildMessageSummary(msg *tele.Message) (string, *tele.SendOptions) {
	opts := &tele.SendOptions{
		DisableWebPagePreview: true,
	}
	if msg == nil {
		return "", opts
	}

	var b summaryTextBuilder

	appendStorySummary(&b, msg)
	appendReplySummary(&b, msg)
	appendForwardSummary(&b, msg)
	appendViaBotSummary(&b, msg)
	appendStateSummary(&b, msg)

	if len(b.entities) > 0 {
		opts.Entities = b.entities
	}

	return strings.TrimSpace(b.String()), opts
}

func appendStorySummary(b *summaryTextBuilder, msg *tele.Message) {
	story := msg.ReplyToStory
	if story == nil {
		story = msg.Story
	}
	if story == nil {
		return
	}

	b.appendLine(func(b *summaryTextBuilder) {
		b.write("Это ответ на ")
		b.writeLink("историю", summaryLinkURL)
		if owner := storyOwnerPhrase(story); owner != "" {
			b.write(" ")
			b.write(owner)
		}
		b.write(".")
	})
}

func appendReplySummary(b *summaryTextBuilder, msg *tele.Message) {
	summary := buildReplySummary(msg)
	if !summary.hasReply {
		return
	}

	b.appendLine(func(b *summaryTextBuilder) {
		b.write("Это ответ ")
		if summary.quoted {
			b.writeLink("цитатой", summaryLinkURL)
			b.write(" на ")
		} else {
			b.write("на ")
		}
		b.writeLink("сообщение", summaryLinkURL)
		if summary.kind != "" {
			b.write(" ")
			b.write(summary.kind)
		}
		if summary.source != "" {
			b.write(" ")
			b.write(summary.source)
		}
		b.write(".")
	})

	if summary.quoteText != "" {
		appendQuotedLine(b, "Цитата: ", summary.quoteText)
		return
	}

	appendQuotedLine(b, "Текст исходного сообщения: ", summary.previewText)
}

func appendForwardSummary(b *summaryTextBuilder, msg *tele.Message) {
	summary := buildForwardSummary(msg)
	if !summary.hasData() {
		return
	}

	b.appendLine(func(b *summaryTextBuilder) {
		b.write("Сообщение переслано")
		if source := forwardSourcePhrase(summary); source != "" {
			b.write(source)
		}

		var details []string
		if summary.date != "" {
			details = append(details, summary.date)
		}
		if summary.signature != "" {
			details = append(details, "подпись автора: "+summary.signature)
		}
		if summary.automatic {
			details = append(details, "это автоматическая пересылка")
		}
		if len(details) > 0 {
			b.write(", ")
			b.write(strings.Join(details, ", "))
		}

		b.write(".")
	})
}

func appendViaBotSummary(b *summaryTextBuilder, msg *tele.Message) {
	if msg.Via == nil {
		return
	}

	viaBot := formatUser(msg.Via)
	if viaBot == "" {
		viaBot = "без имени"
	}

	b.appendLine(func(b *summaryTextBuilder) {
		b.write("Сообщение отправлено через бота ")
		b.write(viaBot)
		b.write(".")
	})
}

func appendStateSummary(b *summaryTextBuilder, msg *tele.Message) {
	var sentences []string
	if msg.Protected {
		sentences = append(sentences, "Сообщение защищено от пересылки.")
	}
	if msg.FromOffline {
		sentences = append(sentences, "Сообщение было отправлено офлайн.")
	}
	if len(sentences) == 0 {
		return
	}

	b.appendLine(func(b *summaryTextBuilder) {
		b.write(strings.Join(sentences, " "))
	})
}

func buildReplySummary(msg *tele.Message) replySummary {
	quoteText := ""
	if msg.Quote != nil {
		quoteText = normalizeInlineText(msg.Quote.Text)
	}

	if msg.ExternalReply != nil {
		return buildExternalReplySummary(msg.ExternalReply, quoteText)
	}
	if msg.ReplyTo != nil {
		return buildInternalReplySummary(msg.ReplyTo, quoteText)
	}
	if quoteText != "" {
		return replySummary{
			hasReply:  true,
			quoted:    true,
			quoteText: quoteText,
		}
	}

	return replySummary{}
}

func buildInternalReplySummary(replyTo *tele.Message, quoteText string) replySummary {
	return replySummary{
		hasReply:    true,
		quoted:      quoteText != "",
		source:      replyTargetPhrase(formatUser(replyTo.Sender), replyTo.Chat),
		kind:        messageKindQualifier(replyTo),
		quoteText:   quoteText,
		previewText: messageText(replyTo),
	}
}

func buildExternalReplySummary(ext *tele.ExternalReply, quoteText string) replySummary {
	user := ""
	chat := ext.Chat
	if ext.Origin != nil {
		user = originUserDisplay(ext.Origin)
		if chat == nil {
			chat = originChat(ext.Origin)
		}
	}

	return replySummary{
		hasReply:  true,
		quoted:    quoteText != "",
		source:    replyTargetPhrase(user, chat),
		kind:      externalReplyKindQualifier(ext),
		quoteText: quoteText,
	}
}

func buildForwardSummary(msg *tele.Message) forwardSummary {
	summary := forwardSummary{
		user:       formatUser(msg.OriginalSender),
		chat:       msg.OriginalChat,
		senderName: normalizeInlineText(msg.OriginalSenderName),
		date:       formatForwardedDateRussian(int64(msg.OriginalUnixtime)),
		signature:  normalizeInlineText(msg.OriginalSignature),
		automatic:  msg.AutomaticForward,
	}

	if msg.Origin == nil {
		return summary
	}

	if summary.user == "" {
		summary.user = originUserDisplay(msg.Origin)
	}
	if summary.chat == nil {
		summary.chat = originChat(msg.Origin)
	}
	if summary.signature == "" {
		summary.signature = normalizeInlineText(msg.Origin.Signature)
	}
	if summary.date == "" {
		summary.date = formatForwardedDateRussian(msg.Origin.DateUnixtime)
	}
	if normalizeOriginType(msg.Origin.Type) == "hidden_user" {
		summary.hidden = true
	}

	return summary
}

func (s forwardSummary) hasData() bool {
	return s.user != "" ||
		s.chat != nil ||
		s.hidden ||
		s.senderName != "" ||
		s.date != "" ||
		s.signature != "" ||
		s.automatic
}

func (b *summaryTextBuilder) appendLine(render func(*summaryTextBuilder)) {
	if b.sb.Len() > 0 {
		b.write("\n")
	}
	render(b)
}

func (b *summaryTextBuilder) write(text string) {
	if text == "" {
		return
	}
	b.sb.WriteString(text)
	b.utf16Len += telegramTextLen(text)
}

func (b *summaryTextBuilder) writeLink(text, url string) {
	if text == "" {
		return
	}
	b.entities = append(b.entities, tele.MessageEntity{
		Type:   tele.EntityTextLink,
		Offset: b.utf16Len,
		Length: telegramTextLen(text),
		URL:    url,
	})
	b.write(text)
}

func (b *summaryTextBuilder) String() string {
	return b.sb.String()
}

func appendQuotedLine(b *summaryTextBuilder, prefix, rawText string) {
	text := normalizeInlineText(rawText)
	if text == "" {
		return
	}

	available := maxSummaryTextLen - b.utf16Len
	if b.sb.Len() > 0 {
		available -= telegramTextLen("\n")
	}
	minLen := telegramTextLen(prefix + "«x».")
	if available < minLen {
		return
	}

	maxTextLen := available - telegramTextLen(prefix) - telegramTextLen("«».")
	text = truncateTelegramText(text, maxTextLen)
	if text == "" {
		return
	}

	b.appendLine(func(b *summaryTextBuilder) {
		b.write(prefix)
		b.write("«")
		b.write(text)
		b.write("».")
	})
}

func normalizeOriginType(t string) string {
	t = strings.TrimSpace(t)
	t = strings.TrimPrefix(t, "message_origin_")
	return t
}

func formatForwardedDateRussian(unix int64) string {
	if unix == 0 {
		return ""
	}
	t := time.Unix(unix, 0)
	loc := time.FixedZone("GMT+3", 3*60*60)
	t = t.In(loc)
	return "дата отправки оригинала (GMT+3): " + t.Format("02.01.2006 15:04")
}

func originUserDisplay(o *tele.MessageOrigin) string {
	if o == nil {
		return ""
	}
	if o.Sender != nil {
		return formatUser(o.Sender)
	}
	if o.SenderUsername != "" {
		return "@" + strings.TrimPrefix(o.SenderUsername, "@")
	}
	return ""
}

func originChat(o *tele.MessageOrigin) *tele.Chat {
	if o == nil {
		return nil
	}

	switch normalizeOriginType(o.Type) {
	case "channel":
		if o.Chat != nil {
			return o.Chat
		}
	case "chat":
		if o.SenderChat != nil {
			return o.SenderChat
		}
	}

	if o.Chat != nil {
		return o.Chat
	}
	return o.SenderChat
}

func storyOwnerPhrase(story *tele.Story) string {
	if story == nil || story.Poster == nil {
		return ""
	}
	return ownerPhrase(story.Poster)
}

func replyTargetPhrase(user string, chat *tele.Chat) string {
	var parts []string
	if user != "" {
		parts = append(parts, "пользователя "+user)
	}
	if chatPart := replyChatPhrase(chat, user != ""); chatPart != "" {
		parts = append(parts, chatPart)
	}
	return strings.Join(parts, " ")
}

func replyChatPhrase(chat *tele.Chat, hasUser bool) string {
	if chat == nil {
		return ""
	}
	display := formatChat(chat)
	if display == "" {
		return ""
	}

	switch chat.Type {
	case tele.ChatPrivate:
		if hasUser {
			return ""
		}
		return "пользователя " + display
	case tele.ChatChannel, tele.ChatChannelPrivate:
		return "из канала " + display
	default:
		return "из чата " + display
	}
}

func forwardSourcePhrase(summary forwardSummary) string {
	switch {
	case summary.user != "" && summary.chat != nil:
		if chatPart := forwardChatPhrase(summary.chat, true); chatPart != "" {
			return " от пользователя " + summary.user + " " + chatPart
		}
		return " от пользователя " + summary.user
	case summary.user != "":
		return " от пользователя " + summary.user
	case summary.chat != nil:
		if chatPart := forwardChatPhrase(summary.chat, false); chatPart != "" {
			return " " + chatPart
		}
	case summary.hidden && summary.senderName != "":
		return " от пользователя со скрытым профилем по имени " + summary.senderName
	case summary.hidden:
		return " от пользователя со скрытым профилем"
	case summary.senderName != "":
		return " от отправителя " + summary.senderName
	}

	return ""
}

func forwardChatPhrase(chat *tele.Chat, hasUser bool) string {
	if chat == nil {
		return ""
	}
	display := formatChat(chat)
	if display == "" {
		return ""
	}

	switch chat.Type {
	case tele.ChatPrivate:
		if hasUser {
			return ""
		}
		return "от пользователя " + display
	case tele.ChatChannel, tele.ChatChannelPrivate:
		return "из канала " + display
	default:
		return "из чата " + display
	}
}

func ownerPhrase(chat *tele.Chat) string {
	if chat == nil {
		return ""
	}
	display := formatChat(chat)
	if display == "" {
		return ""
	}

	switch chat.Type {
	case tele.ChatPrivate:
		return "пользователя " + display
	case tele.ChatChannel, tele.ChatChannelPrivate:
		return "из канала " + display
	default:
		return "из чата " + display
	}
}

func messageKindQualifier(msg *tele.Message) string {
	if msg == nil {
		return ""
	}

	switch {
	case msg.Photo != nil:
		return "с фотографией"
	case msg.Video != nil:
		return "с видео"
	case msg.Document != nil:
		return "с документом"
	case msg.Audio != nil:
		return "с аудио"
	case msg.Voice != nil:
		return "с голосовым сообщением"
	case msg.VideoNote != nil:
		return "с видеообщением"
	case msg.Sticker != nil:
		return "со стикером"
	case msg.Animation != nil:
		return "с GIF-анимацией"
	case msg.Location != nil:
		return "с геолокацией"
	case msg.Contact != nil:
		return "с контактом"
	case msg.Venue != nil:
		return "с местом"
	case msg.Poll != nil:
		return "с опросом"
	case msg.Dice != nil:
		return "с кубиком"
	case msg.Invoice != nil:
		return "со счётом"
	case msg.Game != nil:
		return "с игрой"
	case len(msg.PaidMedia.PaidMedia) > 0:
		return "с платным контентом"
	default:
		return ""
	}
}

func externalReplyKindQualifier(ext *tele.ExternalReply) string {
	if ext == nil {
		return ""
	}

	switch {
	case len(ext.Photo) > 0:
		return "с фотографией"
	case ext.Video != nil:
		return "с видео"
	case ext.Document != nil:
		return "с документом"
	case ext.Audio != nil:
		return "с аудио"
	case ext.Voice != nil:
		return "с голосовым сообщением"
	case ext.Note != nil:
		return "с видеообщением"
	case ext.Sticker != nil:
		return "со стикером"
	case ext.Animation != nil:
		return "с GIF-анимацией"
	case ext.Location != nil:
		return "с геолокацией"
	case ext.Contact != nil:
		return "с контактом"
	case ext.Venue != nil:
		return "с местом"
	case ext.Poll != nil:
		return "с опросом"
	case ext.Dice != nil:
		return "с кубиком"
	case ext.Invoice != nil:
		return "со счётом"
	case ext.Game != nil:
		return "с игрой"
	case ext.Story != nil:
		return "с историей"
	case len(ext.PaidMedia.PaidMedia) > 0:
		return "с платным контентом"
	default:
		return ""
	}
}

func formatChat(c *tele.Chat) string {
	if c == nil {
		return ""
	}
	if c.Username != "" {
		return "@" + strings.TrimPrefix(c.Username, "@")
	}
	if name := strings.TrimSpace(c.FirstName + " " + c.LastName); name != "" {
		return name
	}
	return strings.TrimSpace(c.Title)
}

func messageText(msg *tele.Message) string {
	if msg == nil {
		return ""
	}
	if text := normalizeInlineText(msg.Text); text != "" {
		return text
	}
	return normalizeInlineText(msg.Caption)
}

func normalizeInlineText(text string) string {
	return strings.Join(strings.Fields(text), " ")
}

func truncateTelegramText(text string, maxUnits int) string {
	text = normalizeInlineText(text)
	if text == "" || maxUnits <= 0 {
		return ""
	}
	if telegramTextLen(text) <= maxUnits {
		return text
	}

	ellipsis := "..."
	if maxUnits <= telegramTextLen(ellipsis) {
		return trimToUTF16(text, maxUnits)
	}

	limit := maxUnits - telegramTextLen(ellipsis)
	var out strings.Builder
	used := 0

	for _, r := range text {
		runeLen := telegramRuneLen(r)
		if used+runeLen > limit {
			break
		}
		out.WriteRune(r)
		used += runeLen
	}

	return strings.TrimSpace(out.String()) + ellipsis
}

func trimToUTF16(text string, maxUnits int) string {
	if maxUnits <= 0 {
		return ""
	}

	var out strings.Builder
	used := 0
	for _, r := range text {
		runeLen := telegramRuneLen(r)
		if used+runeLen > maxUnits {
			break
		}
		out.WriteRune(r)
		used += runeLen
	}
	return strings.TrimSpace(out.String())
}

func telegramTextLen(text string) int {
	return len(utf16.Encode([]rune(text)))
}

func telegramRuneLen(r rune) int {
	if r > 0xFFFF {
		return 2
	}
	return 1
}
