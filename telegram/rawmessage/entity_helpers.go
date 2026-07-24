package rawmessage

import tele "gopkg.in/telebot.v4"

func rawEntitiesSlice(raw any) []any {
	items, ok := raw.([]any)
	if !ok || len(items) == 0 {
		return nil
	}
	return items
}

func offsetRawEntities(raw any, delta int) any {
	items := rawEntitiesSlice(raw)
	if len(items) == 0 || delta == 0 {
		return raw
	}
	out := make([]any, len(items))
	for i, item := range items {
		m, ok := item.(map[string]any)
		if !ok {
			out[i] = item
			continue
		}
		shifted := make(map[string]any, len(m))
		for k, v := range m {
			shifted[k] = v
		}
		shifted["offset"] = intFromAny(m["offset"]) + delta
		out[i] = shifted
	}
	return out
}

func appendOffsetRawEntities(dst []any, raw any, delta int) []any {
	items := rawEntitiesSlice(raw)
	if len(items) == 0 {
		return dst
	}
	offsetted, ok := offsetRawEntities(items, delta).([]any)
	if !ok {
		return dst
	}
	return append(dst, offsetted...)
}

// OffsetTeleEntities shifts entity offsets; used by notification dispatch.
func OffsetTeleEntities(entities []tele.MessageEntity, delta int) []tele.MessageEntity {
	if delta == 0 || len(entities) == 0 {
		return append([]tele.MessageEntity(nil), entities...)
	}
	out := make([]tele.MessageEntity, len(entities))
	for i, entity := range entities {
		entity.Offset += delta
		out[i] = entity
	}
	return out
}

func TeleEntitiesFromRaw(raw any) []tele.MessageEntity {
	items := rawEntitiesSlice(raw)
	if len(items) == 0 {
		return nil
	}
	out := make([]tele.MessageEntity, 0, len(items))
	for _, item := range items {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		entity, ok := mapToEntity(m)
		if ok {
			out = append(out, entity)
		}
	}
	return out
}

func TeleEntitiesToRaw(entities []tele.MessageEntity) []any {
	if len(entities) == 0 {
		return nil
	}
	out := make([]any, len(entities))
	for i, entity := range entities {
		m := map[string]any{
			"type":   string(entity.Type),
			"offset": entity.Offset,
			"length": entity.Length,
		}
		if entity.URL != "" {
			m["url"] = entity.URL
		}
		if entity.Language != "" {
			m["language"] = entity.Language
		}
		if entity.CustomEmojiID != "" {
			m["custom_emoji_id"] = entity.CustomEmojiID
		}
		out[i] = m
	}
	return out
}
