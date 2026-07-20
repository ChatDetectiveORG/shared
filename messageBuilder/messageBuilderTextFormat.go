package messageBuilder

import (
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/ChatDetectiveORG/shared/utils"
	tele "gopkg.in/telebot.v4"
)

type Format string

const (
	FormatBold          Format = "bold"
	FormatItalic        Format = "italic"
	FormatUnderline     Format = "underline"
	FormatLink          Format = "link"
	FormatBlockquote    Format = "blockquote"
	FormatMono          Format = "mono"
	FormatSpoiler       Format = "spoiler"
	FormatStrikethrough Format = "strikethrough"
)

type TextFormat struct {
	Type Format
	URL  string

	isCustomEmoji bool
}

func (self TextFormat) WithCustomEmojiID(id string) TextFormat {
	self.URL = "tg://emoji?id=" + id
	self.isCustomEmoji = true
	return self
}

func (self TextFormat) WithUserMention(id int64) TextFormat {
	self.URL = "tg://user?id=" + strconv.FormatInt(id, 10)
	return self
}

func (self *TextFormat) tagWrap() string {
	switch self.Type {
	case FormatBold:
		return "*" + "%s" + "*"
	case FormatItalic:
		return "_" + "%s" + "_"
	case FormatUnderline:
		return "__" + "%s" + "__"
	case FormatStrikethrough:
		return "~" + "%s" + "~"
	case FormatLink:
		res := "[" + "%s" + "](" + self.URL + ")"
		if self.isCustomEmoji {
			res = "!" + res
		}
		return res
	case FormatBlockquote:
		return "\n>%s"
	case FormatMono:
		return "`" + "%s" + "`"
	case FormatSpoiler:
		return "||" + "%s" + "||"
	case "mention":
		return "%s"
	default:
		return "%s"
	}
}

func (self *TextFormat) toMdV2Tag(content string, other ...TextFormat) string {
	content = utils.EscapeMarkdownV2(content)

	typeMap := make(map[Format]TextFormat)
	order := []Format{}
	for _, f := range other {
		typeMap[f.Type] = f
		order = append(order, f.Type)
	}
	typeMap[self.Type] = *self
	order = append(order, self.Type)

	uniqueTypes := []TextFormat{}
	seen := map[Format]struct{}{}
	for i := len(order) - 1; i >= 0; i-- {
		typ := order[i]
		if _, ok := seen[typ]; !ok {
			uniqueTypes = append([]TextFormat{typeMap[typ]}, uniqueTypes...)
			seen[typ] = struct{}{}
		}
	}

	formatPriority := map[Format]int{
		FormatMono:          7,
		FormatBlockquote:    6,
		FormatBold:          5,
		FormatItalic:        4,
		FormatUnderline:     3,
		FormatSpoiler:       2,
		FormatStrikethrough: 1,
		FormatLink:          0,
	}
	slices.SortFunc(uniqueTypes, func(a, b TextFormat) int {
		ap, aok := formatPriority[a.Type]
		bp, bok := formatPriority[b.Type]
		if !aok {
			ap = 100
		}
		if !bok {
			bp = 100
		}
		return ap - bp
	})

	for _, format := range uniqueTypes {
		if format.Type == FormatBlockquote {
			content = strings.ReplaceAll(content, "\n", "\n>")
		}
		content = fmt.Sprintf(format.tagWrap(), content)
	}

	return content
}

func (self *TextFormat) toTelebotTag(content string, offset int) tele.MessageEntity {
	contentLen := utils.TgLen(content)

	switch self.Type {
	case FormatBold:
		return tele.MessageEntity{
			Type:   tele.EntityBold,
			Offset: offset,
			Length: contentLen,
		}
	case FormatItalic:
		return tele.MessageEntity{
			Type:   tele.EntityItalic,
			Offset: offset,
			Length: contentLen,
		}
	case FormatUnderline:
		return tele.MessageEntity{
			Type:   tele.EntityUnderline,
			Offset: offset,
			Length: contentLen,
		}
	case FormatLink:
		return tele.MessageEntity{
			Type:   tele.EntityTextLink,
			Offset: offset,
			Length: contentLen,
			URL:    self.URL,
		}
	case FormatStrikethrough:
		return tele.MessageEntity{
			Type:   tele.EntityStrikethrough,
			Offset: offset,
			Length: contentLen,
		}
	case FormatBlockquote:
		return tele.MessageEntity{
			Type:   tele.EntityBlockquote,
			Offset: offset,
			Length: contentLen,
		}
	case FormatMono:
		return tele.MessageEntity{
			Type:   tele.EntityCode,
			Offset: offset,
			Length: contentLen,
		}
	case FormatSpoiler:
		return tele.MessageEntity{
			Type:   tele.EntitySpoiler,
			Offset: offset,
			Length: contentLen,
		}
	default:
		return tele.MessageEntity{Type: ""}
	}

}
