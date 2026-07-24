package rawmessage

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"

	e "github.com/ChatDetectiveORG/shared/errors"
	"github.com/ChatDetectiveORG/shared/utils"
	tele "gopkg.in/telebot.v4"
)

const (
	MetadataFormatLegacyStruct = 0
	MetadataFormatRawAPIv1     = 1
)

var fileIDMediaKeys = []string{
	"photo", "video", "document", "audio", "voice", "video_note",
	"sticker", "animation",
}

var primaryMediaKeys = []string{
	"photo", "video", "document", "audio", "voice", "video_note",
	"sticker", "animation", "poll", "location", "contact", "venue",
	"dice", "game", "invoice", "paid_media",
}

var allContentKeys = append(append(append([]string(nil), primaryMediaKeys...), "checklist"), "rich_message")

var serviceContentKeys = []string{
	"giveaway", "giveaway_winners", "giveaway_created", "giveaway_completed",
	"story", "live_photo", "checklist_tasks_done", "checklist_tasks_added",
	"successful_payment", "pinned_message",
}

var richBlockOutputOnlyKeys = []string{
	"message_id", "date", "edit_date", "file_unique_id", "file_size",
	"chat", "from", "sender_chat", "forward_origin", "forward_from",
	"forward_from_chat", "forward_date", "via_bot",
}

var checklistTaskOutputOnlyKeys = []string{
	"completed_by_user", "completed_by_chat", "completion_date",
}

var readOnlyKeys = []string{
	"message_id", "date", "edit_date", "forward_from", "forward_from_chat",
	"forward_from_message_id", "forward_signature", "forward_sender_name",
	"forward_date", "forward_origin", "via_bot", "reply_to_message",
	"is_topic_message", "is_automatic_forward", "has_protected_content",
	"media_group_id", "author_signature", "sender_chat", "sender_boost_count",
	"business_connection_id",
}

type StoredMessage struct {
	Format  int
	Payload json.RawMessage
}

type ReplayOptions struct {
	TargetChatID     int64
	ReplyToMessageID int
	StripReadOnly    bool
	// IncludeBusinessConnection adds business_connection_id to the Telegram API call.
	// Must be false when replaying into the bot↔owner notification chat (mirror).
	// True only when sending back into the original business customer chat.
	IncludeBusinessConnection bool
	// AllowCopyMessage enables copyMessage as the first checklist replay strategy.
	AllowCopyMessage          bool
	BusinessConnectionID      string
}

func StripReadOnlyFields(m map[string]any) map[string]any {
	out := make(map[string]any, len(m))
	for k, v := range m {
		if isReadOnlyKey(k) {
			continue
		}
		out[k] = v
	}
	return out
}

func isReadOnlyKey(key string) bool {
	for _, k := range readOnlyKeys {
		if k == key {
			return true
		}
	}
	return false
}

func IsPlainText(raw json.RawMessage) bool {
	m, err := decodeMap(raw)
	if err != nil {
		return false
	}
	text, _ := m["text"].(string)
	if text == "" {
		return false
	}
	for _, key := range allContentKeys {
		if blocksPlainText(m, key) {
			return false
		}
	}
	return true
}

func blocksPlainText(m map[string]any, key string) bool {
	if isFileIDMediaKey(key) {
		_, ok := m[key]
		return ok
	}
	return hasContent(m, key)
}

func isFileIDMediaKey(key string) bool {
	for _, mediaKey := range fileIDMediaKeys {
		if mediaKey == key {
			return true
		}
	}
	return false
}

func mediaKeyPresent(m map[string]any, key string) bool {
	_, ok := m[key]
	return ok
}

func hasContent(m map[string]any, key string) bool {
	v, ok := m[key]
	if !ok || v == nil {
		return false
	}
	switch val := v.(type) {
	case string:
		return val != ""
	case []any:
		return len(val) > 0
	case map[string]any:
		return len(val) > 0
	default:
		return true
	}
}

func ExtractTextFields(raw json.RawMessage) (text string, entities []tele.MessageEntity, ok bool) {
	m, err := decodeMap(raw)
	if err != nil {
		return "", nil, false
	}
	textVal, _ := m["text"].(string)
	if textVal == "" {
		return "", nil, false
	}
	rawEntities, _ := m["entities"].([]any)
	for _, item := range rawEntities {
		entityMap, mapOK := item.(map[string]any)
		if !mapOK {
			continue
		}
		entity, entityOK := mapToEntity(entityMap)
		if entityOK {
			entities = append(entities, entity)
		}
	}
	return textVal, entities, true
}

func BuildSendPayload(raw json.RawMessage, opts ReplayOptions) (method string, payload map[string]any, err error) {
	m, err := decodeMap(raw)
	if err != nil {
		return "", nil, err
	}

	method, payload, err = methodAndPayloadFromMessage(m, opts)
	if err != nil {
		return "", nil, err
	}
	payload["chat_id"] = opts.TargetChatID
	if opts.ReplyToMessageID != 0 {
		payload["reply_to_message_id"] = opts.ReplyToMessageID
	}
	applyBusinessConnection(m, payload, opts)
	return method, payload, nil
}

func BuildAlbumPayload(raws []json.RawMessage, opts ReplayOptions) (method string, payload map[string]any, err error) {
	if len(raws) == 0 {
		return "", nil, fmt.Errorf("rawmessage: empty album")
	}

	type indexed struct {
		id      int
		payload map[string]any
	}
	items := make([]indexed, 0, len(raws))
	for _, raw := range raws {
		m, decErr := decodeMap(raw)
		if decErr != nil {
			return "", nil, decErr
		}
		msgID := messageIDFromMap(m)
		media, mediaErr := inputMediaFromMap(m)
		if mediaErr != nil {
			return "", nil, mediaErr
		}
		items = append(items, indexed{id: msgID, payload: media})
	}

	sort.Slice(items, func(i, j int) bool {
		return items[i].id < items[j].id
	})

	media := make([]map[string]any, 0, len(items))
	for _, item := range items {
		media = append(media, item.payload)
	}

	payload = map[string]any{
		"chat_id": opts.TargetChatID,
		"media":   media,
	}
	if opts.ReplyToMessageID != 0 {
		payload["reply_to_message_id"] = opts.ReplyToMessageID
	}
	applyBusinessConnection(nil, payload, opts)
	return "sendMediaGroup", payload, nil
}

func LoadStoredMessage(format int, encrypted []byte, key []byte) (StoredMessage, *tele.Message, error) {
	decrypted, errInfo := utils.Decrypt(encrypted, key)
	if e.IsNonNil(errInfo) {
		return StoredMessage{}, nil, fmt.Errorf("decrypt metadata: %s", errInfo.Message)
	}

	if format == MetadataFormatRawAPIv1 {
		return StoredMessage{Format: format, Payload: json.RawMessage(decrypted)}, nil, nil
	}

	var legacy tele.Message
	if uerr := json.Unmarshal(decrypted, &legacy); uerr != nil {
		return StoredMessage{}, nil, uerr
	}
	return StoredMessage{Format: MetadataFormatLegacyStruct, Payload: json.RawMessage(decrypted)}, &legacy, nil
}

func MarshalBusinessMessage(msg *tele.Message) ([]byte, error) {
	if msg == nil {
		return nil, fmt.Errorf("rawmessage: nil message")
	}
	return json.Marshal(msg)
}

func methodAndPayloadFromMessage(m map[string]any, opts ReplayOptions) (string, map[string]any, error) {
	mediaTypes := []struct {
		key            string
		method         string
		captionCapable bool
	}{
		{key: "photo", method: "sendPhoto", captionCapable: true},
		{key: "video", method: "sendVideo", captionCapable: true},
		{key: "document", method: "sendDocument", captionCapable: true},
		{key: "audio", method: "sendAudio", captionCapable: true},
		{key: "voice", method: "sendVoice", captionCapable: true},
		{key: "video_note", method: "sendVideoNote"},
		{key: "animation", method: "sendAnimation", captionCapable: true},
		{key: "sticker", method: "sendSticker"},
	}
	for _, mediaType := range mediaTypes {
		if !mediaKeyPresent(m, mediaType.key) {
			continue
		}
		if !hasContent(m, mediaType.key) {
			return "", nil, fmt.Errorf("rawmessage: %s is missing file_id", mediaType.key)
		}
		fileID, err := requiredFileID(mediaType.key, m[mediaType.key])
		if err != nil {
			return "", nil, err
		}
		payload := map[string]any{mediaType.key: fileID}
		if mediaType.captionCapable {
			copySingleCaptionFields(m, payload)
		}
		copyReplayEffectFields(m, payload)
		applyBusinessConnection(m, payload, opts)
		return mediaType.method, payload, nil
	}

	exoticAdapters := []struct {
		key                        string
		method                     string
		requiresBusinessConnection bool
		build                      func(map[string]any) (map[string]any, error)
	}{
		{key: "poll", method: "sendPoll", requiresBusinessConnection: true, build: buildSendPollPayload},
		{key: "checklist", method: "sendChecklist", requiresBusinessConnection: true, build: buildSendChecklistPayload},
		{key: "rich_message", method: "sendRichMessage", requiresBusinessConnection: true, build: buildSendRichMessagePayload},
		{key: "location", method: "sendLocation", build: buildSendLocationPayload},
		{key: "venue", method: "sendVenue", build: buildSendVenuePayload},
		{key: "contact", method: "sendContact", build: buildSendContactPayload},
		{key: "dice", method: "sendDice", build: buildSendDicePayload},
		{key: "game", method: "sendGame", build: buildSendGamePayload},
		{key: "invoice", method: "sendInvoice", build: buildSendInvoicePayload},
		{key: "paid_media", method: "sendPaidMedia", build: buildSendPaidMediaPayload},
	}
	for _, adapter := range exoticAdapters {
		if !hasContent(m, adapter.key) {
			continue
		}
		if adapter.key == "checklist" && !opts.IncludeBusinessConnection {
			method, payload, err := buildChecklistNotificationPayload(m, opts)
			if err != nil {
				return "", nil, err
			}
			copyReplayEffectFields(m, payload)
			if _, hasChat := payload["chat_id"]; !hasChat {
				payload["chat_id"] = opts.TargetChatID
			}
			if opts.ReplyToMessageID != 0 {
				payload["reply_to_message_id"] = opts.ReplyToMessageID
			}
			return method, payload, nil
		}
		payload, err := adapter.build(m)
		if err != nil {
			return "", nil, err
		}
		if err := requireBusinessConnection(m, opts, adapter.requiresBusinessConnection, adapter.method); err != nil {
			return "", nil, err
		}
		copyReplayEffectFields(m, payload)
		applyBusinessConnection(m, payload, opts)
		return adapter.method, payload, nil
	}

	if serviceKey, ok := hasServiceOnlyContent(m); ok {
		return "", nil, fmt.Errorf("rawmessage: service or unreplayable message content: %s", serviceKey)
	}

	text, _ := m["text"].(string)
	if text == "" {
		return "", nil, fmt.Errorf("rawmessage: unsupported or empty message content")
	}
	payload := map[string]any{"text": text}
	copyIfPresent(m, payload, "link_preview_options")
	if entities, exists := m["entities"]; exists {
		payload["entities"] = entities
	} else {
		copyIfPresent(m, payload, "parse_mode")
	}
	copyReplayEffectFields(m, payload)
	applyBusinessConnection(m, payload, opts)
	return "sendMessage", payload, nil
}

func resolveBusinessConnectionID(m map[string]any, opts ReplayOptions) string {
	if m != nil {
		if connID, _ := m["business_connection_id"].(string); connID != "" {
			return connID
		}
	}
	return opts.BusinessConnectionID
}

func applyBusinessConnection(m, payload map[string]any, opts ReplayOptions) {
	if !opts.IncludeBusinessConnection || payload == nil {
		return
	}
	if connID := resolveBusinessConnectionID(m, opts); connID != "" {
		payload["business_connection_id"] = connID
	}
}

func requireBusinessConnection(m map[string]any, opts ReplayOptions, required bool, method string) error {
	if !required || !opts.IncludeBusinessConnection {
		return nil
	}
	if resolveBusinessConnectionID(m, opts) == "" {
		return fmt.Errorf("rawmessage: business_connection_id is required for %s", method)
	}
	return nil
}

// EnrichRawBusinessConnection returns raw with business_connection_id set when missing.
func EnrichRawBusinessConnection(raw json.RawMessage, businessConnectionID string) json.RawMessage {
	if businessConnectionID == "" || len(raw) == 0 {
		return raw
	}
	m, err := decodeMap(raw)
	if err != nil {
		return raw
	}
	if connID, _ := m["business_connection_id"].(string); connID != "" {
		return raw
	}
	m["business_connection_id"] = businessConnectionID
	enriched, err := json.Marshal(m)
	if err != nil {
		return raw
	}
	return enriched
}

func copyReplayEffectFields(m, payload map[string]any) {
	if effectID, exists := m["effect_id"]; exists {
		payload["message_effect_id"] = effectID
	} else {
		copyIfPresent(m, payload, "message_effect_id")
	}
}

func hasServiceOnlyContent(m map[string]any) (string, bool) {
	for _, key := range serviceContentKeys {
		if hasContent(m, key) {
			return key, true
		}
	}
	return "", false
}

func requiredFileID(mediaType string, raw any) (string, error) {
	var fileID string
	if mediaType == "photo" {
		fileID = largestPhotoFileID(raw)
	} else {
		fileID = fileIDFromMediaObject(raw)
	}
	if fileID == "" {
		return "", fmt.Errorf("rawmessage: %s is missing file_id", mediaType)
	}
	return fileID, nil
}

func copySingleCaptionFields(src, dst map[string]any) {
	if spoiler, exists := src["has_media_spoiler"]; exists {
		dst["has_spoiler"] = spoiler
	} else {
		copyIfPresent(src, dst, "has_spoiler")
	}

	caption, _ := src["caption"].(string)
	if caption == "" {
		return
	}
	dst["caption"] = caption
	copyIfPresent(src, dst, "show_caption_above_media")
	if entities, exists := src["caption_entities"]; exists {
		dst["caption_entities"] = entities
		return
	}
	copyIfPresent(src, dst, "parse_mode")
}

func copyIfPresent(src, dst map[string]any, key string) {
	if value, exists := src[key]; exists {
		dst[key] = value
	}
}

func inputMediaFromMap(m map[string]any) (map[string]any, error) {
	switch {
	case hasContent(m, "photo"):
		out := map[string]any{"type": "photo"}
		if photoID := largestPhotoFileID(m["photo"]); photoID != "" {
			out["media"] = photoID
		}
		copyCaptionFields(m, out)
		return out, nil
	case hasContent(m, "video"):
		out := map[string]any{"type": "video", "media": fileIDFromMediaObject(m["video"])}
		copyCaptionFields(m, out)
		return out, nil
	case hasContent(m, "document"):
		out := map[string]any{"type": "document", "media": fileIDFromMediaObject(m["document"])}
		copyCaptionFields(m, out)
		return out, nil
	case hasContent(m, "audio"):
		out := map[string]any{"type": "audio", "media": fileIDFromMediaObject(m["audio"])}
		copyCaptionFields(m, out)
		return out, nil
	case hasContent(m, "animation"):
		out := map[string]any{"type": "animation", "media": fileIDFromMediaObject(m["animation"])}
		copyCaptionFields(m, out)
		return out, nil
	default:
		return nil, fmt.Errorf("rawmessage: unsupported album item")
	}
}

func copyCaptionFields(src, dst map[string]any) {
	if caption, ok := src["caption"].(string); ok && caption != "" {
		dst["caption"] = caption
	}
	if entities, ok := src["caption_entities"]; ok {
		dst["caption_entities"] = entities
	}
}

func largestPhotoFileID(raw any) string {
	items, ok := raw.([]any)
	if !ok || len(items) == 0 {
		return ""
	}
	bestID := ""
	bestSize := -1
	for _, item := range items {
		obj, mapOK := item.(map[string]any)
		if !mapOK {
			continue
		}
		fileID, _ := obj["file_id"].(string)
		size := intFromAny(obj["file_size"])
		if fileID != "" && size >= bestSize {
			bestSize = size
			bestID = fileID
		}
	}
	if bestID == "" {
		if obj, mapOK := items[len(items)-1].(map[string]any); mapOK {
			bestID, _ = obj["file_id"].(string)
		}
	}
	return bestID
}

func fileIDFromMediaObject(raw any) string {
	obj, ok := raw.(map[string]any)
	if !ok {
		return ""
	}
	fileID, _ := obj["file_id"].(string)
	return fileID
}

func messageIDFromMap(m map[string]any) int {
	return intFromAny(m["message_id"])
}

func intFromAny(v any) int {
	switch n := v.(type) {
	case float64:
		return int(n)
	case int:
		return n
	case int64:
		return int(n)
	case json.Number:
		i, _ := n.Int64()
		return int(i)
	case string:
		i, _ := strconv.Atoi(n)
		return i
	default:
		return 0
	}
}

func decodeMap(raw json.RawMessage) (map[string]any, error) {
	if len(raw) == 0 {
		return nil, fmt.Errorf("rawmessage: empty payload")
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		return nil, err
	}
	return m, nil
}

func mapToEntity(m map[string]any) (tele.MessageEntity, bool) {
	entityType, _ := m["type"].(string)
	if entityType == "" {
		return tele.MessageEntity{}, false
	}
	entity := tele.MessageEntity{
		Type:   tele.EntityType(entityType),
		Offset: intFromAny(m["offset"]),
		Length: intFromAny(m["length"]),
	}
	if url, ok := m["url"].(string); ok {
		entity.URL = url
	}
	if lang, ok := m["language"].(string); ok {
		entity.Language = lang
	}
	if emojiID, ok := m["custom_emoji_id"].(string); ok {
		entity.CustomEmojiID = emojiID
	}
	return entity, true
}

func buildSendPollPayload(m map[string]any) (map[string]any, error) {
	pollObj, ok := m["poll"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("rawmessage: poll is missing or invalid")
	}
	question, _ := pollObj["question"].(string)
	if question == "" {
		return nil, fmt.Errorf("rawmessage: poll question is required")
	}

	payload := map[string]any{
		"question":  question,
		"is_closed": true,
	}
	if _, exists := pollObj["type"]; !exists {
		payload["type"] = "regular"
	}
	copyIfPresent(pollObj, payload, "type")
	copyIfPresent(pollObj, payload, "is_anonymous")
	copyIfPresent(pollObj, payload, "allows_multiple_answers")
	copyIfPresent(pollObj, payload, "allows_revoting")
	copyIfPresent(pollObj, payload, "shuffle_options")
	copyIfPresent(pollObj, payload, "allow_adding_options")
	copyIfPresent(pollObj, payload, "hide_results_until_closes")
	copyIfPresent(pollObj, payload, "correct_option_ids")
	copyIfPresent(pollObj, payload, "explanation")
	copyIfPresent(pollObj, payload, "explanation_entities")
	copyIfPresent(pollObj, payload, "explanation_parse_mode")
	copyIfPresent(pollObj, payload, "description")
	copyIfPresent(pollObj, payload, "description_entities")
	copyIfPresent(pollObj, payload, "description_parse_mode")
	copyIfPresent(pollObj, payload, "question_entities")
	copyIfPresent(pollObj, payload, "question_parse_mode")
	copyIfPresent(pollObj, payload, "open_period")
	copyIfPresent(pollObj, payload, "close_date")

	options, err := inputPollOptionsFromMap(pollObj["options"])
	if err != nil {
		return nil, err
	}
	payload["options"] = options
	return payload, nil
}

func inputPollOptionsFromMap(raw any) ([]map[string]any, error) {
	items, ok := raw.([]any)
	if !ok || len(items) == 0 {
		return nil, fmt.Errorf("rawmessage: poll options are required")
	}
	options := make([]map[string]any, 0, len(items))
	for _, item := range items {
		optMap, mapOK := item.(map[string]any)
		if !mapOK {
			continue
		}
		text, _ := optMap["text"].(string)
		if text == "" {
			continue
		}
		inputOpt := map[string]any{"text": text}
		if entities, exists := optMap["text_entities"]; exists {
			inputOpt["text_entities"] = entities
		} else {
			copyIfPresent(optMap, inputOpt, "text_parse_mode")
		}
		options = append(options, inputOpt)
	}
	if len(options) == 0 {
		return nil, fmt.Errorf("rawmessage: poll options are required")
	}
	return options, nil
}

func buildSendChecklistPayload(m map[string]any) (map[string]any, error) {
	checklistObj, ok := m["checklist"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("rawmessage: checklist is missing or invalid")
	}

	tasks, err := inputChecklistTasksFromMap(checklistObj["tasks"])
	if err != nil {
		return nil, err
	}

	inputChecklist := map[string]any{"tasks": tasks}
	copyIfPresent(checklistObj, inputChecklist, "title")
	copyIfPresent(checklistObj, inputChecklist, "title_entities")
	copyIfPresent(checklistObj, inputChecklist, "others_can_add_tasks")
	copyIfPresent(checklistObj, inputChecklist, "others_can_mark_tasks_as_done")

	return map[string]any{"checklist": inputChecklist}, nil
}

func buildSendRichMessagePayload(m map[string]any) (map[string]any, error) {
	richObj, ok := m["rich_message"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("rawmessage: rich_message is missing or invalid")
	}

	inputRich := convertRichMessageOutputToInput(richObj)
	if !hasRichMessageInput(inputRich) {
		return nil, fmt.Errorf("rawmessage: rich_message blocks, html, or markdown are required")
	}

	return map[string]any{"rich_message": inputRich}, nil
}

func hasRichMessageInput(rich map[string]any) bool {
	if html, _ := rich["html"].(string); html != "" {
		return true
	}
	if md, _ := rich["markdown"].(string); md != "" {
		return true
	}
	blocks, ok := rich["blocks"].([]any)
	return ok && len(blocks) > 0
}

func stripRichBlockMap(m map[string]any) map[string]any {
	out := make(map[string]any, len(m))
	for k, v := range m {
		if isRichBlockOutputOnlyKey(k) || isChecklistTaskOutputOnlyKey(k) {
			continue
		}
		switch val := v.(type) {
		case map[string]any:
			out[k] = stripRichBlockMap(val)
		case []any:
			out[k] = stripRichBlockSlice(val)
		default:
			out[k] = v
		}
	}
	return out
}

func stripRichBlockSlice(items []any) []any {
	out := make([]any, 0, len(items))
	for _, item := range items {
		switch val := item.(type) {
		case map[string]any:
			out = append(out, stripRichBlockMap(val))
		case []any:
			out = append(out, stripRichBlockSlice(val))
		default:
			out = append(out, val)
		}
	}
	return out
}

func isRichBlockOutputOnlyKey(key string) bool {
	for _, k := range richBlockOutputOnlyKeys {
		if k == key {
			return true
		}
	}
	return false
}

func isChecklistTaskOutputOnlyKey(key string) bool {
	for _, k := range checklistTaskOutputOnlyKeys {
		if k == key {
			return true
		}
	}
	return false
}

func inputChecklistTasksFromMap(raw any) ([]map[string]any, error) {
	items, ok := raw.([]any)
	if !ok || len(items) == 0 {
		return nil, fmt.Errorf("rawmessage: checklist tasks are required")
	}
	tasks := make([]map[string]any, 0, len(items))
	for _, item := range items {
		taskMap, mapOK := item.(map[string]any)
		if !mapOK {
			continue
		}
		text, _ := taskMap["text"].(string)
		if text == "" {
			continue
		}
		inputTask := map[string]any{"text": text}
		if id := intFromAny(taskMap["id"]); id != 0 {
			inputTask["id"] = id
		}
		if entities, exists := taskMap["text_entities"]; exists {
			inputTask["text_entities"] = entities
		}
		tasks = append(tasks, inputTask)
	}
	if len(tasks) == 0 {
		return nil, fmt.Errorf("rawmessage: checklist tasks are required")
	}
	return tasks, nil
}

func buildSendLocationPayload(m map[string]any) (map[string]any, error) {
	locationObj, ok := m["location"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("rawmessage: location is missing or invalid")
	}
	payload := map[string]any{}
	copyIfPresent(locationObj, payload, "latitude")
	copyIfPresent(locationObj, payload, "longitude")
	copyIfPresent(locationObj, payload, "live_period")
	copyIfPresent(locationObj, payload, "heading")
	copyIfPresent(locationObj, payload, "proximity_alert_radius")
	copyIfPresent(locationObj, payload, "horizontal_accuracy")
	if _, hasLat := payload["latitude"]; !hasLat {
		return nil, fmt.Errorf("rawmessage: location latitude is required")
	}
	if _, hasLng := payload["longitude"]; !hasLng {
		return nil, fmt.Errorf("rawmessage: location longitude is required")
	}
	return payload, nil
}

func buildSendVenuePayload(m map[string]any) (map[string]any, error) {
	venueObj, ok := m["venue"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("rawmessage: venue is missing or invalid")
	}
	locationObj, ok := venueObj["location"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("rawmessage: venue location is missing or invalid")
	}
	payload := map[string]any{}
	copyIfPresent(locationObj, payload, "latitude")
	copyIfPresent(locationObj, payload, "longitude")
	copyIfPresent(locationObj, payload, "horizontal_accuracy")
	copyIfPresent(venueObj, payload, "title")
	copyIfPresent(venueObj, payload, "address")
	copyIfPresent(venueObj, payload, "foursquare_id")
	copyIfPresent(venueObj, payload, "foursquare_type")
	copyIfPresent(venueObj, payload, "google_place_id")
	copyIfPresent(venueObj, payload, "google_place_type")
	if _, hasLat := payload["latitude"]; !hasLat {
		return nil, fmt.Errorf("rawmessage: venue latitude is required")
	}
	if _, hasLng := payload["longitude"]; !hasLng {
		return nil, fmt.Errorf("rawmessage: venue longitude is required")
	}
	return payload, nil
}

func buildSendContactPayload(m map[string]any) (map[string]any, error) {
	contactObj, ok := m["contact"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("rawmessage: contact is missing or invalid")
	}
	phoneNumber, _ := contactObj["phone_number"].(string)
	firstName, _ := contactObj["first_name"].(string)
	if phoneNumber == "" || firstName == "" {
		return nil, fmt.Errorf("rawmessage: contact phone_number and first_name are required")
	}
	payload := map[string]any{
		"phone_number": phoneNumber,
		"first_name":   firstName,
	}
	copyIfPresent(contactObj, payload, "last_name")
	copyIfPresent(contactObj, payload, "vcard")
	return payload, nil
}

func buildSendDicePayload(m map[string]any) (map[string]any, error) {
	diceObj, ok := m["dice"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("rawmessage: dice is missing or invalid")
	}
	emoji, _ := diceObj["emoji"].(string)
	if emoji == "" {
		return nil, fmt.Errorf("rawmessage: dice emoji is required")
	}
	return map[string]any{"emoji": emoji}, nil
}

func buildSendGamePayload(m map[string]any) (map[string]any, error) {
	gameObj, ok := m["game"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("rawmessage: game is missing or invalid")
	}
	shortName, _ := gameObj["game_short_name"].(string)
	if shortName == "" {
		shortName, _ = gameObj["title"].(string)
	}
	if shortName == "" {
		return nil, fmt.Errorf("rawmessage: game_short_name is required")
	}
	return map[string]any{"game_short_name": shortName}, nil
}

func buildSendInvoicePayload(m map[string]any) (map[string]any, error) {
	invoiceObj, ok := m["invoice"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("rawmessage: invoice is missing or invalid")
	}
	title, _ := invoiceObj["title"].(string)
	description, _ := invoiceObj["description"].(string)
	payloadStr, _ := invoiceObj["payload"].(string)
	currency, _ := invoiceObj["currency"].(string)
	if title == "" || description == "" || payloadStr == "" || currency == "" {
		return nil, fmt.Errorf("rawmessage: invoice title, description, payload and currency are required")
	}
	payload := map[string]any{
		"title":       title,
		"description": description,
		"payload":     payloadStr,
		"currency":    currency,
	}
	copyIfPresent(invoiceObj, payload, "provider_token")
	copyIfPresent(invoiceObj, payload, "provider_data")
	copyIfPresent(invoiceObj, payload, "prices")
	copyIfPresent(invoiceObj, payload, "max_tip_amount")
	copyIfPresent(invoiceObj, payload, "suggested_tip_amounts")
	copyIfPresent(invoiceObj, payload, "start_parameter")
	copyIfPresent(invoiceObj, payload, "need_name")
	copyIfPresent(invoiceObj, payload, "need_phone_number")
	copyIfPresent(invoiceObj, payload, "need_email")
	copyIfPresent(invoiceObj, payload, "need_shipping_address")
	copyIfPresent(invoiceObj, payload, "send_phone_number_to_provider")
	copyIfPresent(invoiceObj, payload, "send_email_to_provider")
	copyIfPresent(invoiceObj, payload, "is_flexible")
	copyIfPresent(invoiceObj, payload, "subscription_period")
	if totalAmount, exists := invoiceObj["total_amount"]; exists {
		payload["total_amount"] = totalAmount
	}
	return payload, nil
}

func buildSendPaidMediaPayload(m map[string]any) (map[string]any, error) {
	paidMediaObj, ok := m["paid_media"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("rawmessage: paid_media is missing or invalid")
	}
	starCount := intFromAny(paidMediaObj["star_count"])
	if starCount <= 0 {
		return nil, fmt.Errorf("rawmessage: paid_media star_count is required")
	}
	mediaItems, err := inputPaidMediaFromMap(paidMediaObj["paid_media"])
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"star_count": starCount,
		"media":      mediaItems,
	}, nil
}

func inputPaidMediaFromMap(raw any) ([]map[string]any, error) {
	items, ok := raw.([]any)
	if !ok || len(items) == 0 {
		return nil, fmt.Errorf("rawmessage: paid_media items are required")
	}
	media := make([]map[string]any, 0, len(items))
	for _, item := range items {
		itemMap, mapOK := item.(map[string]any)
		if !mapOK {
			continue
		}
		mediaType, _ := itemMap["type"].(string)
		if mediaType == "" {
			continue
		}
		inputItem := map[string]any{"type": mediaType}
		switch mediaType {
		case "photo":
			if fileID := largestPhotoFileID(itemMap["photo"]); fileID != "" {
				inputItem["media"] = fileID
			} else if fileID := fileIDFromMediaObject(itemMap["photo"]); fileID != "" {
				inputItem["media"] = fileID
			}
		case "video":
			if fileID := fileIDFromMediaObject(itemMap["video"]); fileID != "" {
				inputItem["media"] = fileID
			}
		}
		if inputItem["media"] == nil {
			return nil, fmt.Errorf("rawmessage: paid_media item is missing file_id")
		}
		media = append(media, inputItem)
	}
	if len(media) == 0 {
		return nil, fmt.Errorf("rawmessage: paid_media items are required")
	}
	return media, nil
}
