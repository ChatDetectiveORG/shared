package notification

import (
	"fmt"

	"github.com/ChatDetectiveORG/shared/utils"
	tele "gopkg.in/telebot.v4"
)

const actorNamePrefix = "Пользователь "

func actorLinkOffset() int {
	return utils.TgLen(actorNamePrefix)
}

func actorLinkEntity(actor Actor) tele.MessageEntity {
	return tele.MessageEntity{
		Type:   tele.EntityTextLink,
		Offset: actorLinkOffset(),
		Length: utils.TgLen(actor.Name),
		URL:    fmt.Sprintf("tg://user?id=%d", actor.ID),
	}
}

func editMessageOldVersionPrefix(actor Actor) string {
	return fmt.Sprintf("%s%s изменил сообщение!\nСтарая версия:", actorNamePrefix, actor.Name)
}

func editMessageOldVersionPrefixMultiline(actor Actor) string {
	return fmt.Sprintf("%s%s изменил сообщение!\nСтарая версия:\n", actorNamePrefix, actor.Name)
}

func editMessageNewVersionPrefix(actor Actor) string {
	return fmt.Sprintf("%s%s изменил сообщение!\nНовая версия:", actorNamePrefix, actor.Name)
}

func editMessageNewVersionPostfix() string {
	return "Новая версия:\n"
}

func editMediaGroupOldVersionPrefix(actor Actor) string {
	return fmt.Sprintf("%s%s изменил медиагруппу!\nСтарая версия:\n", actorNamePrefix, actor.Name)
}

func editMediaGroupNewVersionPrefix(actor Actor) string {
	return fmt.Sprintf("%s%s изменил медиагруппу!\nНовая версия:\n", actorNamePrefix, actor.Name)
}

func deleteMessagePrefix(actor Actor) string {
	return fmt.Sprintf("%s%s удалил сообщение!\n", actorNamePrefix, actor.Name)
}

func deleteMediaGroupPrefix(actor Actor) string {
	return fmt.Sprintf("%s%s удалил медиагруппу!\n", actorNamePrefix, actor.Name)
}
