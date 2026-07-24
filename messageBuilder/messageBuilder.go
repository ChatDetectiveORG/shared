package messageBuilder

import (
	tele "gopkg.in/telebot.v4"

	"github.com/go-pg/pg/v10/orm"
	"github.com/gomodule/redigo/redis"
)

type MessageBuilder = tele.MessageBuilder

type CreateGenericKeyboardParams = tele.CreateGenericKeyboardParams

type TelegramButtonConversionArgs = tele.TelegramButtonConversionArgs

type Buttonable = tele.Buttonable

type MediaGroup = tele.MediaGroup

type MirrorFileAsset = tele.MirrorFileAsset

type Format = tele.Format

type TextFormat = tele.TextFormat

type Args = tele.Args

const (
	FormatBold          = tele.FormatBold
	FormatItalic        = tele.FormatItalic
	FormatUnderline     = tele.FormatUnderline
	FormatLink          = tele.FormatLink
	FormatBlockquote    = tele.FormatBlockquote
	FormatMono          = tele.FormatMono
	FormatSpoiler       = tele.FormatSpoiler
	FormatStrikethrough = tele.FormatStrikethrough
)

var (
	T             = tele.T
	B             = tele.B
	I             = tele.I
	U             = tele.U
	Strikethrough = tele.Strikethrough
	Mono          = tele.Mono
	Spoiler       = tele.Spoiler
	Link          = tele.Link
	UserMention   = tele.UserMention
	E             = tele.E
)

func CreateGenericKeyboard[T Buttonable](
	builder *MessageBuilder,
	query *orm.Query,
	redisConn redis.Conn,
	postgresDb orm.DB,
	callbackData string,
	params CreateGenericKeyboardParams,
) {
	tele.CreateGenericKeyboard[T](builder, query, redisConn, postgresDb, callbackData, params)
}
