package handlers

import (
	"strings"

	tele "gopkg.in/telebot.v4"
)

// UpdateFilter ограничивает обработку endpoint-ом только подходящими апдейтами.
type UpdateFilter interface {
	Filter(update tele.Update) bool
}

type commandFilter struct {
	commands []string
}

func (c *commandFilter) Filter(update tele.Update) bool {
	for _, command := range c.commands {
		if update.Message == nil {
			continue
		}
		if update.Message.Text == "" {
			continue
		}
		text := strings.TrimSpace(update.Message.Text)
		prefix := "/" + command
		if !strings.HasPrefix(text, prefix) {
			continue
		}
		rest := strings.TrimPrefix(text, prefix)
		// Telegram commands may be: "/cmd", "/cmd@bot", "/cmd arg1 arg2"
		if rest == "" || strings.HasPrefix(rest, " ") || strings.HasPrefix(rest, "@") {
			return true
		}
	}
	return false
}

func Command(command []string) UpdateFilter {
	return &commandFilter{
		commands: command,
	}
}

type textCommand struct {
	matchString string
}

func (t *textCommand) Filter(update tele.Update) bool {
	if update.Message == nil {
		return false
	}
	if update.Message.Text == "" {
		return false
	}
	return strings.Contains(update.Message.Text, t.matchString)
}

func TextCommand(matchString string) UpdateFilter {
	return &textCommand{
		matchString: matchString,
	}
}

type callbackQueryJSON struct {
	matchCallbackDataArg string
	matchCallbackDataKey string
}

func (c *callbackQueryJSON) Filter(update tele.Update) bool {
	if update.Callback == nil {
		return false
	}
	if update.Callback.Data == "" {
		return false
	}
	return strings.Contains(update.Callback.Data, c.matchCallbackDataArg) && strings.Contains(update.Callback.Data, c.matchCallbackDataKey)
}

func CallbackQueryJSON(matchCallbackDataArg string, matchCallbackDataKey string) UpdateFilter {
	return &callbackQueryJSON{
		matchCallbackDataArg: matchCallbackDataArg,
		matchCallbackDataKey: matchCallbackDataKey,
	}
}

type busEventType string

const (
	BusEventTypeNew    busEventType = "new"
	BusEventTypeEdited busEventType = "edited"
	BusEventTypeDeleted busEventType = "deleted"
)

type businessEvent struct {
	types busEventType
}

func (b *businessEvent) Filter(update tele.Update) bool {
	switch b.types {
	case BusEventTypeNew:
		return update.BusinessMessage != nil
	case BusEventTypeEdited:
		return update.EditedBusinessMessage != nil
	case BusEventTypeDeleted:
		return update.DeletedBusinessMessages != nil
	default:
		return false
	}
}

func BusinessEvent(acceptedTypes busEventType) UpdateFilter {
	return &businessEvent{
		types: acceptedTypes,
	}
}

type filterChain struct {
	filters  []UpdateFilter
	operator string
}

func (f *filterChain) Filter(update tele.Update) bool {
	for _, filter := range f.filters {
		if !filter.Filter(update) {
			if f.operator == "and" {
				return false
			}

			continue
		}

		if f.operator == "or" {
			return true
		}
	}

	return f.operator == "and"
}

func And(filters ...UpdateFilter) UpdateFilter {
	return &filterChain{
		filters:  filters,
		operator: "and",
	}
}

func Or(filters ...UpdateFilter) UpdateFilter {
	return &filterChain{
		filters:  filters,
		operator: "or",
	}
}
