package messageBuilder

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/ChatDetectiveORG/shared/utils"
	tele "gopkg.in/telebot.v4"
)

func (self *MessageBuilder) Write(contents ...text) *MessageBuilder {
	self.checkBuilder()

	for _, t := range contents {
		if self.Mdv2Enabled {
			s := t.ToMdV2String()
			self.builder.WriteString(s)
			self.cursorPosition += utils.TgLen(s)

			continue
		}

		self.entities = append(self.entities, t.TelebotEntities(self.cursorPosition)...)

		s := t.Text()
		self.builder.WriteString(s)
		self.cursorPosition += utils.TgLen(s)
	}

	return self
}

type text interface {
	ToMdV2String() string
	TelebotEntities(offset int) []tele.MessageEntity
	Text() string
}

type universalText struct {
	Contents string
	Upper    text

	MdV2tag       string
	TelebotEntity tele.MessageEntity
}

func (self *universalText) ToMdV2String() string {
	if self.Upper != nil {
		return self.MdV2tag + self.Upper.ToMdV2String() + self.MdV2tag

	}

	return self.MdV2tag + utils.EscapeMarkdownV2(self.Contents) + self.MdV2tag
}

func (self *universalText) TelebotEntities(offset int) []tele.MessageEntity {
	if self.TelebotEntity.Type == "" {
		return []tele.MessageEntity{}
	}

	self.TelebotEntity.Length = utils.TgLen(self.Text())
	self.TelebotEntity.Offset = offset

	if self.Upper != nil {
		entities := self.Upper.TelebotEntities(offset)

		entities = append(entities, self.TelebotEntity)

		return entities
	}

	return []tele.MessageEntity{self.TelebotEntity}
}

func (self *universalText) Text() string {
	if self.Upper != nil {
		return self.Upper.Text()
	}

	// fmt.Sprintf()

	return self.Contents
}

type Args struct {
	A         []any
	NoNewline bool
}

// Represents Plain Text
//
// Embeds fmt.Sprintf logic. Last arg reserved for newline control
func T(t string, a ...Args) text {
	postfix := "\n"

	if len(a) > 0 && a[0].NoNewline {
		postfix = ""
	}

	contents := t + postfix
	if len(a) > 0 && a[0].A != nil && len(a[0].A) > 0 {
		contents = fmt.Sprintf(strings.ReplaceAll(contents, "%", "%%"), a[0].A)
	}

	return &universalText{
		Contents:      contents,
		TelebotEntity: tele.MessageEntity{Type: ""},
		MdV2tag:       "",
	}
}

// Represents Bold Text
func B(t text) text {
	return &universalText{
		MdV2tag: "*",
		TelebotEntity: tele.MessageEntity{
			Type: tele.EntityBold,
		},
		Upper: t,
	}
}

// Represents Italic Text
func I(t text) text {
	return &universalText{
		MdV2tag: "_",
		TelebotEntity: tele.MessageEntity{
			Type: tele.EntityItalic,
		},
		Upper: t,
	}
}

// Represents Underline Text
func U(t text) text {
	return &universalText{
		MdV2tag: "__",
		TelebotEntity: tele.MessageEntity{
			Type: tele.EntityUnderline,
		},
		Upper: t,
	}
}

func Strikethrough(t text) text {
	return &universalText{
		MdV2tag: "~",
		TelebotEntity: tele.MessageEntity{
			Type: tele.EntityStrikethrough,
		},
		Upper: t,
	}
}

func Mono(t text) text {
	return &universalText{
		MdV2tag: "`",
		TelebotEntity: tele.MessageEntity{
			Type: tele.EntityCode,
		},
		Upper: t,
	}
}

func Spoiler(t text) text {
	return &universalText{
		MdV2tag: "||",
		TelebotEntity: tele.MessageEntity{
			Type: tele.EntitySpoiler,
		},
		Upper: t,
	}
}

type link struct {
	Contents      string
	IsCustomEmoji bool
	URL           string
}

func (self *link) ToMdV2String() string {

	res := "[" + utils.EscapeMarkdownV2(self.Contents) + "](" + self.URL + ")"
	if self.IsCustomEmoji {
		res = "!" + res
	}

	return res
}

func (self *link) TelebotEntities(offset int) []tele.MessageEntity {
	return []tele.MessageEntity{{
		Type:   tele.EntityTextLink,
		Offset: offset,
		Length: utils.TgLen(self.Contents),
		URL:    self.URL,
	}}
}

func (self *link) Text() string {
	return self.Contents
}

func Link(t string, url string) text {
	return &link{
		Contents: t,
		URL:      url,
	}
}

func UserMention(name string, ID int64) text {
	return &link{
		Contents: name,
		URL:      "tg://user?id=" + strconv.FormatInt(ID, 10),
	}
}

// Represents Custom Emoji
func E(emojiID string, placholder ...string) text {
	emoji := "👾"
	if len(placholder) > 0 {
		emoji = placholder[0]
	}

	return &link{
		Contents:      emoji,
		URL:           "tg://emoji?id=" + emojiID,
		IsCustomEmoji: true,
	}
}
