package notification

import (
	"context"
	"encoding/json"

	e "github.com/ChatDetectiveORG/shared/errors"
	h "github.com/ChatDetectiveORG/shared/handlers"
	"github.com/ChatDetectiveORG/shared/telegram"
	"github.com/ChatDetectiveORG/shared/telegram/rawmessage"
	"github.com/ChatDetectiveORG/shared/utils"
	tele "gopkg.in/telebot.v4"
)

type Actor struct {
	Name string
	ID   int64
}

type EditDispatchInput struct {
	ReceiverID       int64
	Actor            Actor
	OldRaw           json.RawMessage
	NewRaw           json.RawMessage
	OldMessage       *tele.Message
	NewMessage       *tele.Message
	MediaGroupIDHash string
	GroupRaws        []json.RawMessage
	EditedIndex      int
	HighlightDiff    bool
	RoutingKey       string
}

type DeleteDispatchInput struct {
	ReceiverID       int64
	Actor            Actor
	Raw              json.RawMessage
	Message          *tele.Message
	MediaGroupIDHash string
	GroupRaws        []json.RawMessage
	RoutingKey       string
}

func DispatchEdit(ctx context.Context, hashe *h.HandlerChainHashe, input EditDispatchInput) *e.ErrorInfo {
	switch ClassifyEdit(input.OldRaw, input.NewRaw, input.MediaGroupIDHash) {
	case StrategyMediaGroup:
		return dispatchEditMediaGroup(ctx, hashe, input)
	case StrategyTextCombined:
		return dispatchEditCombinedText(ctx, hashe, input)
	default:
		return dispatchEditReplay(ctx, hashe, input)
	}
}

func DispatchDelete(ctx context.Context, hashe *h.HandlerChainHashe, input DeleteDispatchInput) *e.ErrorInfo {
	switch ClassifyDelete(input.Raw, input.MediaGroupIDHash) {
	case StrategyMediaGroup:
		return dispatchDeleteMediaGroup(ctx, hashe, input)
	case StrategyTextCombined:
		return dispatchDeleteCombinedText(ctx, hashe, input)
	default:
		return dispatchDeleteReplay(ctx, hashe, input)
	}
}

func dispatchEditMediaGroup(ctx context.Context, hashe *h.HandlerChainHashe, input EditDispatchInput) *e.ErrorInfo {
	sentOld, err := emitRawAlbumWait(ctx, hashe, input.RoutingKey, input.ReceiverID, input.GroupRaws, 0)
	if e.IsNonNil(err) {
		return err
	}
	oldSummary, oldOpts := buildNoticeSummary(input.OldMessage, input.OldRaw)
	if err := emitNoticeReply(ctx, hashe, input.RoutingKey, input.ReceiverID, replyTargetID(sentOld), editMediaGroupOldVersionPrefix(input.Actor), oldSummary, oldOpts, input.Actor); e.IsNonNil(err) {
		return err
	}

	newGroup := append([]json.RawMessage(nil), input.GroupRaws...)
	if input.EditedIndex >= 0 && input.EditedIndex < len(newGroup) {
		newGroup[input.EditedIndex] = input.NewRaw
	}
	sentNew, err := emitRawAlbumWait(ctx, hashe, input.RoutingKey, input.ReceiverID, newGroup, 0)
	if e.IsNonNil(err) {
		return err
	}
	newSummary, newOpts := buildNoticeSummary(input.NewMessage, input.NewRaw)
	return emitNoticeReply(ctx, hashe, input.RoutingKey, input.ReceiverID, replyTargetID(sentNew), editMediaGroupNewVersionPrefix(input.Actor), newSummary, newOpts, input.Actor)
}

func dispatchEditCombinedText(ctx context.Context, hashe *h.HandlerChainHashe, input EditDispatchInput) *e.ErrorInfo {
	oldText, oldEntities, _ := rawmessage.ExtractTextFields(input.OldRaw)
	newText, newEntities, _ := rawmessage.ExtractTextFields(input.NewRaw)

	prefix := editMessageOldVersionPrefixMultiline(input.Actor)
	postfix := editMessageNewVersionPostfix()
	prefixLen := utils.TgLen(prefix)
	postfixLen := utils.TgLen(postfix)
	oldTextLen := utils.TgLen(oldText)
	newTextLen := utils.TgLen(newText)

	entities := rawmessage.OffsetTeleEntities(oldEntities, prefixLen)
	entities = append(entities, rawmessage.OffsetTeleEntities(newEntities, prefixLen+oldTextLen+postfixLen)...)
	if input.HighlightDiff {
		oldDiff, newDiff := computeDiffBoldEntities(oldText, newText)
		entities = append(entities, rawmessage.OffsetTeleEntities(oldDiff, prefixLen)...)
		entities = append(entities, rawmessage.OffsetTeleEntities(newDiff, prefixLen+oldTextLen+postfixLen)...)
	}
	entities = append(entities,
		actorLinkEntity(input.Actor),
		tele.MessageEntity{Type: tele.EntityEBlockquote, Offset: prefixLen, Length: oldTextLen},
		tele.MessageEntity{Type: tele.EntityEBlockquote, Offset: prefixLen + oldTextLen + postfixLen, Length: newTextLen},
	)

	combined := &tele.Message{
		Chat:     &tele.Chat{ID: input.ReceiverID},
		Text:     prefix + oldText + postfix + newText,
		Entities: entities,
	}
	sent, err := hashe.EmitWait(ctx, input.RoutingKey, combined)
	if e.IsNonNil(err) {
		return err
	}

	summary, summaryOpts := buildNoticeSummary(input.NewMessage, input.NewRaw)
	if summary == "" {
		return e.Nil()
	}
	return hashe.Emit(input.RoutingKey, withSummaryReply(input.ReceiverID, sent, summary, summaryOpts))
}

func dispatchEditReplay(ctx context.Context, hashe *h.HandlerChainHashe, input EditDispatchInput) *e.ErrorInfo {
	sentOld, err := emitRawReplayWait(ctx, hashe, input.RoutingKey, input.ReceiverID, input.OldRaw, 0, false)
	if e.IsNonNil(err) {
		return err
	}
	if err := emitNoticeReply(ctx, hashe, input.RoutingKey, input.ReceiverID, sentOld, editMessageOldVersionPrefix(input.Actor), "", nil, input.Actor); e.IsNonNil(err) {
		return err
	}

	sentNew, err := emitRawReplayWait(ctx, hashe, input.RoutingKey, input.ReceiverID, input.NewRaw, 0, true)
	if e.IsNonNil(err) {
		return err
	}
	summary, summaryOpts := buildNoticeSummary(input.NewMessage, input.NewRaw)
	return emitNoticeReply(ctx, hashe, input.RoutingKey, input.ReceiverID, sentNew, editMessageNewVersionPrefix(input.Actor), summary, summaryOpts, input.Actor)
}

func dispatchDeleteMediaGroup(ctx context.Context, hashe *h.HandlerChainHashe, input DeleteDispatchInput) *e.ErrorInfo {
	sent, err := emitRawAlbumWait(ctx, hashe, input.RoutingKey, input.ReceiverID, input.GroupRaws, 0)
	if e.IsNonNil(err) {
		return err
	}
	summary, summaryOpts := buildNoticeSummary(input.Message, input.Raw)
	return emitNoticeReply(ctx, hashe, input.RoutingKey, input.ReceiverID, replyTargetID(sent), deleteMediaGroupPrefix(input.Actor), summary, summaryOpts, input.Actor)
}

func dispatchDeleteCombinedText(ctx context.Context, hashe *h.HandlerChainHashe, input DeleteDispatchInput) *e.ErrorInfo {
	text, entities, _ := rawmessage.ExtractTextFields(input.Raw)
	prefix := deleteMessagePrefix(input.Actor)
	prefixLen := utils.TgLen(prefix)
	textLen := utils.TgLen(text)

	entities = rawmessage.OffsetTeleEntities(entities, prefixLen)
	entities = append(entities,
		actorLinkEntity(input.Actor),
		tele.MessageEntity{Type: tele.EntityEBlockquote, Offset: prefixLen, Length: textLen},
	)

	msg := &tele.Message{
		Chat:     &tele.Chat{ID: input.ReceiverID},
		Text:     prefix + text,
		Entities: entities,
	}
	_, err := hashe.EmitWait(ctx, input.RoutingKey, msg)
	return err
}

func dispatchDeleteReplay(ctx context.Context, hashe *h.HandlerChainHashe, input DeleteDispatchInput) *e.ErrorInfo {
	sent, err := emitRawReplayWait(ctx, hashe, input.RoutingKey, input.ReceiverID, input.Raw, 0, true)
	if e.IsNonNil(err) {
		return err
	}
	summary, summaryOpts := buildNoticeSummary(input.Message, input.Raw)
	return emitNoticeReply(ctx, hashe, input.RoutingKey, input.ReceiverID, sent, deleteMessagePrefix(input.Actor), summary, summaryOpts, input.Actor)
}

func emitRawReplayWait(ctx context.Context, hashe *h.HandlerChainHashe, routingKey string, chatID int64, raw json.RawMessage, replyTo int, allowCopy bool) (*tele.Message, *e.ErrorInfo) {
	opts := notificationReplayOptions(chatID, replyTo, allowCopy)
	if rawmessage.HasChecklistContent(raw) {
		return emitChecklistReplayWait(ctx, hashe, routingKey, raw, opts)
	}
	method, payload, err := rawmessage.BuildSendPayload(raw, opts)
	if err != nil {
		return nil, e.FromError(err, "build raw replay payload").WithSeverity(e.Notice)
	}
	body, marshalErr := json.Marshal(payload)
	if marshalErr != nil {
		return nil, e.FromError(marshalErr, "marshal raw replay payload").WithSeverity(e.Notice)
	}
	return hashe.EmitRawWait(ctx, routingKey, method, body)
}

func emitChecklistReplayWait(ctx context.Context, hashe *h.HandlerChainHashe, routingKey string, raw json.RawMessage, opts rawmessage.ReplayOptions) (*tele.Message, *e.ErrorInfo) {
	attempts, err := rawmessage.ChecklistReplayAttempts(raw, opts)
	if err != nil {
		return nil, e.FromError(err, "build checklist replay attempts").WithSeverity(e.Notice)
	}
	var lastErr *e.ErrorInfo
	for _, attempt := range attempts {
		body, marshalErr := json.Marshal(attempt.Payload)
		if marshalErr != nil {
			return nil, e.FromError(marshalErr, "marshal checklist replay payload").WithSeverity(e.Notice)
		}
		msg, waitErr := hashe.EmitRawWait(ctx, routingKey, attempt.Method, body)
		if !e.IsNonNil(waitErr) {
			return msg, e.Nil()
		}
		lastErr = waitErr
	}
	return nil, lastErr
}

func emitRawAlbumWait(ctx context.Context, hashe *h.HandlerChainHashe, routingKey string, chatID int64, raws []json.RawMessage, replyTo int) ([]*tele.Message, *e.ErrorInfo) {
	method, payload, err := rawmessage.BuildAlbumPayload(raws, notificationReplayOptions(chatID, replyTo, false))
	if err != nil {
		return nil, e.FromError(err, "build raw album payload").WithSeverity(e.Notice)
	}
	body, marshalErr := json.Marshal(payload)
	if marshalErr != nil {
		return nil, e.FromError(marshalErr, "marshal raw album payload").WithSeverity(e.Notice)
	}
	return hashe.EmitRawAlbumWait(ctx, routingKey, method, body)
}

func notificationReplayOptions(chatID int64, replyTo int, allowCopy bool) rawmessage.ReplayOptions {
	return rawmessage.ReplayOptions{
		TargetChatID:              chatID,
		ReplyToMessageID:          replyTo,
		IncludeBusinessConnection: false,
		AllowCopyMessage:          allowCopy,
	}
}

func emitNoticeReply(ctx context.Context, hashe *h.HandlerChainHashe, routingKey string, chatID int64, replyTo *tele.Message, prefix, summary string, summaryOpts *tele.SendOptions, actor Actor) *e.ErrorInfo {
	prefixLen := utils.TgLen(prefix)
	if summaryOpts != nil {
		summaryOpts.Entities = rawmessage.OffsetTeleEntities(summaryOpts.Entities, prefixLen)
	}
	msg := &tele.Message{
		Chat:     &tele.Chat{ID: chatID},
		Text:     prefix + summary,
		ReplyTo:  replyTo,
		Entities: tele.Entities{actorLinkEntity(actor)},
	}
	msg = telegram.HideSendOptsIntoMessage(msg, summaryOpts)
	_ = ctx
	return hashe.Emit(routingKey, msg)
}

func withSummaryReply(chatID int64, replyTo *tele.Message, summary string, opts *tele.SendOptions) *tele.Message {
	msg := &tele.Message{
		Chat:    &tele.Chat{ID: chatID},
		Text:    summary,
		ReplyTo: replyTo,
	}
	return telegram.HideSendOptsIntoMessage(msg, opts)
}

func replyTargetID(messages []*tele.Message) *tele.Message {
	if len(messages) == 0 {
		return nil
	}
	return messages[0]
}
