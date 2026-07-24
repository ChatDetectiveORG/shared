package chaintest

import (
	"encoding/json"
	"strconv"

	amqp "github.com/rabbitmq/amqp091-go"
	tele "gopkg.in/telebot.v4"
)

// UpdateDelivery builds an AMQP delivery as api-gateway publishes to business-events queues.
func UpdateDelivery(update tele.Update, sessionID, mirrorID string) amqp.Delivery {
	body, err := json.Marshal(update)
	if err != nil {
		panic("chaintest: marshal update: " + err.Error())
	}
	return amqp.Delivery{
		Body: body,
		Headers: amqp.Table{
			"session_id": sessionID,
			"mirror_id":  mirrorID,
		},
	}
}

// EditedBusinessMessageUpdate builds a minimal edited-business-message update.
func EditedBusinessMessageUpdate(messageID int, businessConnectionID string, customerID int64, customerName, text string) tele.Update {
	return tele.Update{
		ID: 1,
		EditedBusinessMessage: &tele.Message{
			ID:                   messageID,
			BusinessConnectionID: businessConnectionID,
			Text:                 text,
			Chat: &tele.Chat{
				ID:        customerID,
				FirstName: customerName,
				Type:      tele.ChatPrivate,
			},
			Sender: &tele.User{
				ID:        customerID,
				FirstName: customerName,
			},
		},
	}
}

// EditedBusinessMessagePollUpdate builds an edited poll business message update.
func EditedBusinessMessagePollUpdate(messageID int, businessConnectionID string, customerID int64, customerName string, pollJSON json.RawMessage) tele.Update {
	return editedBusinessMessageFromRaw(messageID, businessConnectionID, customerID, customerName, pollJSON, "poll")
}

// EditedBusinessMessageChecklistUpdate builds an edited checklist business message update.
func EditedBusinessMessageChecklistUpdate(messageID int, businessConnectionID string, customerID int64, customerName string, checklistJSON json.RawMessage) tele.Update {
	return editedBusinessMessageFromRaw(messageID, businessConnectionID, customerID, customerName, checklistJSON, "checklist")
}

// EditedBusinessMessageRichUpdate builds an edited rich_message business message update.
func EditedBusinessMessageRichUpdate(messageID int, businessConnectionID string, customerID int64, customerName string, richJSON json.RawMessage) tele.Update {
	return editedBusinessMessageFromRaw(messageID, businessConnectionID, customerID, customerName, richJSON, "rich_message")
}

func editedBusinessMessageFromRaw(messageID int, businessConnectionID string, customerID int64, customerName string, contentJSON json.RawMessage, contentKey string) tele.Update {
	raw := []byte(`{"message_id":` + itoa(messageID) +
		`,"business_connection_id":"` + businessConnectionID + `"` +
		`,"chat":{"id":` + itoa64(customerID) + `,"first_name":"` + customerName + `","type":"private"}` +
		`,"from":{"id":` + itoa64(customerID) + `,"first_name":"` + customerName + `"}` +
		`,"` + contentKey + `":` + string(contentJSON) + `}`)
	var msg tele.Message
	if err := json.Unmarshal(raw, &msg); err != nil {
		panic("chaintest: marshal edited business message: " + err.Error())
	}
	return tele.Update{ID: 1, EditedBusinessMessage: &msg}
}

func itoa(v int) string {
	return strconv.Itoa(v)
}

func itoa64(v int64) string {
	return strconv.FormatInt(v, 10)
}
