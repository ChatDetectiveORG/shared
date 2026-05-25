package telegram

import (
	"fmt"
	"math"
	"slices"
	"strconv"
	"strings"

	"github.com/ChatDetectiveORG/shared/errors"

	"github.com/ChatDetectiveORG/shared/utils"
	"github.com/go-pg/pg/v10/orm"
	tele "gopkg.in/telebot.v4"

	"github.com/gomodule/redigo/redis"
)

type TextFormat struct {
	Type string
	URL string
}

func (self *TextFormat) WithCustomEmojiID(id string) *TextFormat {
	self.URL = "tg://emoji?id=" + id
	return self
}

func (self *TextFormat) WithUserMention(id int64) *TextFormat {
	self.URL = "tg://user?id=" + strconv.FormatInt(id, 10)
	return self
}

func (self *TextFormat) tagWrap() string {
	switch self.Type {
	case "bold":
		return "**" + "%s" + "**"
	case "italic":
		return "*" + "%s" + "*"
	case "underline":
		return "__" + "%s" + "__"
	case "link":
		return "![" + "%s" + "](" + self.URL + ")"
	case "blockquote":
		return "\n>%s"
	case "mono":
		return "`" + "%s" + "`"
	case "spoiler":
		return "||" + "%s" + "||"
	default:
		return "%s"
	}
}

func (self *TextFormat) ToMdV2Tag(content string, other ...TextFormat) string {
	content = utils.EscapeMarkdownV2(content)
	alreadyApplied := []string{}

	for _, format := range other {
		if slices.Contains(alreadyApplied, format.Type) {
			continue
		}

		alreadyApplied = append(alreadyApplied, format.Type)
		
		if format.Type == "blockquote" {
			content = strings.ReplaceAll(content, "\n", "\n>")
		}
		content = fmt.Sprintf(format.tagWrap(), content)
	}

	if self.Type == "blockquote" {
		content = strings.ReplaceAll(content, "\n", "\n>")
	}
	return fmt.Sprintf(self.tagWrap(), content)
}

func (self *TextFormat) ToTelebotTag(content string, offset int) tele.MessageEntity {
	contentLen := utils.TgLen(content)

	switch self.Type {
	case "bold":
		return tele.MessageEntity{
			Type: tele.EntityBold,
			Offset: offset,
			Length: contentLen,
		}
	case "italic":
		return tele.MessageEntity{
			Type: tele.EntityItalic,
			Offset: offset,
			Length: contentLen,
		}
	case "underline":
		return tele.MessageEntity{
			Type: tele.EntityUnderline,
			Offset: offset,
			Length: contentLen,
		}
	case "link":
		return tele.MessageEntity{
			Type: tele.EntityTextLink,
			Offset: offset,
			Length: contentLen,
			URL: self.URL,
		}
	case "blockquote":
		return tele.MessageEntity{
			Type: tele.EntityBlockquote,
			Offset: offset,
			Length: contentLen,
		}
	case "mono":
		return tele.MessageEntity{
			Type: tele.EntityCode,
			Offset: offset,
			Length: contentLen,
		}
	case "spoiler":
		return tele.MessageEntity{
			Type: tele.EntitySpoiler,
			Offset: offset,
			Length: contentLen,
		}
	default:
		return tele.MessageEntity{Type: ""}
	}

}

// Message builder is a helper to build comlex formatted messages
// It allows to build mdv2 messages and messages formatted via hardcoded entities
//
// ToDo: Add advanced keyboard building
type MessageBuilder struct {
	Mdv2Enabled bool

	text string
	entities []tele.MessageEntity

	keyboard [][]tele.InlineButton
	currentRow []tele.InlineButton

	builder *strings.Builder
	cursorPosition int
}

func (self *MessageBuilder) checkBuilder() {
	if self.builder == nil {
		self.builder = &strings.Builder{}
	}
}

func (self *MessageBuilder) writeString(s string, escape bool) {
	self.checkBuilder()

	if self.Mdv2Enabled && escape {
		s = utils.EscapeMarkdownV2(s)
	}

	self.builder.WriteString(s)
	self.cursorPosition += utils.TgLen(s)
}

func (self *MessageBuilder) WriteString(s string, specialFormatting ...TextFormat) *MessageBuilder {
	self.checkBuilder()

	if len(specialFormatting) == 0 {
		self.writeString(s, true)

		return self
	}

	if self.Mdv2Enabled {
		if len(specialFormatting) > 1 {
			self.writeString(specialFormatting[0].ToMdV2Tag(s, specialFormatting[1:]...), false)
		} else {
			self.writeString(specialFormatting[0].ToMdV2Tag(s), false)
		}

		return self
	}

	offset := self.cursorPosition
	self.writeString(s, true)

	for _, format := range specialFormatting {
		entity := format.ToTelebotTag(s, offset)
		if entity.Type != "" {
			self.entities = append(self.entities, entity)
		}
	}

	return self
}

func (self *MessageBuilder) AddButton(button tele.InlineButton) *MessageBuilder {
	self.currentRow = append(self.currentRow, button)
	return self
}

func (self *MessageBuilder) NextRow() *MessageBuilder {
	self.keyboard = append(self.keyboard, self.currentRow)
	self.currentRow = []tele.InlineButton{}
	return self
}

type CreateGenericKeyboardParams struct {
	ChatID int64
	PageUnique string

	ButtonsPerPage int
	ButtonsPerRow int
	ArrowForwardText string
	ArrowBackText string
	ShowNavigation bool
	MergeButtons [][]tele.InlineButton
}

func (self *CreateGenericKeyboardParams) FillDefaults() *CreateGenericKeyboardParams {
	if self.ButtonsPerPage == 0 {
		self.ButtonsPerPage = defaultCreateGenericKeyboardParams.ButtonsPerPage
	}
	if self.ButtonsPerRow == 0 {
		self.ButtonsPerRow = defaultCreateGenericKeyboardParams.ButtonsPerRow
	}
	if self.ArrowForwardText == "" {
		self.ArrowForwardText = defaultCreateGenericKeyboardParams.ArrowForwardText
	}
	if self.ArrowBackText == "" {
		self.ArrowBackText = defaultCreateGenericKeyboardParams.ArrowBackText
	}

	return self
}

var defaultCreateGenericKeyboardParams = CreateGenericKeyboardParams{
	ButtonsPerPage: 8,
	ButtonsPerRow: 2,
	ArrowForwardText: ">->>",
	ArrowBackText: "<<-<",
}

// Returns: page, error
func (self *MessageBuilder) updateRedis(redisConn redis.Conn, chatID int64, pageUnique string, pageDelta int) (int, *errors.ErrorInfo) {
	key := fmt.Sprintf("keyboard:%s:%d", pageUnique, chatID)
	p, eRaw := redis.Int(redisConn.Do("HGET", key, "page"))
	if eRaw != nil {
		p = 0
	}

	p += pageDelta

	if _, eRaw = redisConn.Do("HSET", key, "page", p); eRaw != nil {
		return p, errors.FromError(eRaw, "failed to update redis").WithSeverity(errors.Notice)
	}
	if _, eRaw = redisConn.Do("EXPIRE", key, 600); eRaw != nil {
		return p, errors.FromError(eRaw, "failed to update redis").WithSeverity(errors.Notice)
	}

	return p, errors.Nil()
}

type Buttonable interface {
	ToTelegramButton() tele.InlineButton
}

// parseKeyboardPageDelta reads pageDelta from callback data (action?...&pageDelta=N).
// Empty callbackData means the first open (delta 0).
func parseKeyboardPageDelta(callbackData string) (int, bool) {
	if callbackData == "" {
		return 0, true
	}

	callbackDataParams := utils.ParseCallbackData(callbackData)
	deltaStr, ok := callbackDataParams["pageDelta"]
	if !ok {
		deltaStr = "0"
	}

	delta, err := strconv.Atoi(deltaStr)
	if err != nil {
		return 0, false
	}

	return delta, true
}

func CreateGenericKeyboard[T any](
	builder *MessageBuilder,
	query *orm.Query,
	redisConn redis.Conn,
	callbackData string,
	params CreateGenericKeyboardParams,
) {
	if params.ChatID == 0 || params.PageUnique == "" {
		return
	}
	
	delta, ok := parseKeyboardPageDelta(callbackData)
	if !ok {
		return
	}

	page, err := builder.updateRedis(redisConn, params.ChatID, params.PageUnique, delta)
	if err != nil {
		return
	}

	params.FillDefaults()

	count, eRaw := query.Count()
	if eRaw != nil {
		return
	}

	if count <= 0 {
		return
	}

	maxPage := int(math.Ceil(float64(count)/float64(params.ButtonsPerPage))) - 1
	
	if page > maxPage {
		page, err = builder.updateRedis(redisConn, params.ChatID, params.PageUnique, page * -1)
		if err != nil {
			return
		}
	}
	if page < 0 {
		page, err = builder.updateRedis(redisConn, params.ChatID, params.PageUnique, maxPage)
		if err != nil {
			return
		}
	}

	var buttons []T

	eRaw = query.Model(&buttons).Limit(params.ButtonsPerPage).Offset(page * params.ButtonsPerPage).Select()
	if eRaw != nil {
		return
	}

	for _, b := range buttons {
		buttonable, ok := any(b).(Buttonable)
		if !ok {
			continue
		}

		button := buttonable.ToTelegramButton()
		builder.AddButton(button)

		if len(builder.currentRow) >= params.ButtonsPerRow {
			builder.NextRow()
		}
	}

	if len(builder.currentRow) > 0 {
		builder.NextRow()
	}

	if len(params.MergeButtons) != 0 {
		for _, row := range params.MergeButtons {
			for _, button := range row {
				builder.AddButton(button)
			}

			builder.NextRow()
		}
	}

	if maxPage > 0 && params.ShowNavigation {
		builder.AddButton(tele.InlineButton{Text: params.ArrowBackText, Data: utils.DumpCallbackData(params.PageUnique, map[string]any{"pageDelta": -1})})
		builder.AddButton(tele.InlineButton{Text: params.ArrowForwardText, Data: utils.DumpCallbackData(params.PageUnique, map[string]any{"pageDelta": 1})})
		builder.NextRow()
	}
}

func (self *MessageBuilder) Build() *tele.Message {
	msg := &tele.Message{
		Text: self.builder.String(),
		ReplyMarkup: &tele.ReplyMarkup{
			InlineKeyboard: self.keyboard,
		},
	}

	if !self.Mdv2Enabled {
		msg.Entities = self.entities
	}

	return msg
}
