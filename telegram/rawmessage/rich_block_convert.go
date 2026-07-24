package rawmessage

// convertRichMessageOutputToInput transforms Message RichMessage JSON (received
// from Bot API) into InputRichMessage suitable for sendRichMessage.
func convertRichMessageOutputToInput(rich map[string]any) map[string]any {
	out := make(map[string]any, len(rich))
	for k, v := range rich {
		switch k {
		case "blocks":
			blocks, ok := v.([]any)
			if !ok {
				continue
			}
			out[k] = convertRichBlocksToInput(blocks)
		case "media":
			if items, ok := v.([]any); ok {
				out[k] = convertRichMessageMediaList(items)
			}
		default:
			out[k] = v
		}
	}
	return out
}

func convertRichMessageMediaList(items []any) []any {
	out := make([]any, 0, len(items))
	for _, item := range items {
		itemMap, ok := item.(map[string]any)
		if !ok {
			continue
		}
		converted := convertRichMessageMediaItem(itemMap)
		if len(converted) > 0 {
			out = append(out, converted)
		}
	}
	return out
}

func convertRichMessageMediaItem(item map[string]any) map[string]any {
	out := make(map[string]any, 2)
	copyIfPresent(item, out, "id")
	if mediaRaw, exists := item["media"]; exists {
		if mediaObj, ok := mediaRaw.(map[string]any); ok {
			if converted := convertInputMediaFromOutput(mediaObj); converted != nil {
				out["media"] = converted
			}
		}
	}
	return out
}

func convertRichBlocksToInput(blocks []any) []any {
	out := make([]any, 0, len(blocks))
	for _, block := range blocks {
		blockMap, ok := block.(map[string]any)
		if !ok {
			continue
		}
		converted := convertRichBlockToInput(blockMap)
		if len(converted) > 0 {
			out = append(out, converted)
		}
	}
	return out
}

func convertRichBlockToInput(block map[string]any) map[string]any {
	blockType, _ := block["type"].(string)
	if blockType == "" {
		return nil
	}

	switch blockType {
	case "photo":
		return convertRichMediaBlock(block, "photo", "photo")
	case "video":
		return convertRichMediaBlock(block, "video", "video")
	case "animation":
		return convertRichMediaBlock(block, "animation", "animation")
	case "audio":
		return convertRichMediaBlock(block, "audio", "audio")
	case "voice_note":
		return convertRichMediaBlock(block, "voice_note", "voice_note")
	case "list":
		return convertRichListBlock(block)
	case "blockquote", "collage", "slideshow", "details":
		return convertRichNestedBlocks(block)
	case "table":
		return convertRichTableBlock(block)
	case "map":
		return convertRichMapBlock(block)
	default:
		return convertRichTextBlock(block)
	}
}

func convertRichMediaBlock(block map[string]any, blockType, mediaKey string) map[string]any {
	out := map[string]any{"type": blockType}
	media := inputMediaFromRichBlockMedia(blockType, mediaKey, block[mediaKey], block["has_spoiler"])
	if media == nil {
		return nil
	}
	out[mediaKey] = media
	copyRichBlockCaption(block, out)
	return out
}

func inputMediaFromRichBlockMedia(blockType, mediaKey string, raw any, hasSpoiler any) map[string]any {
	fileID := richBlockFileID(mediaKey, raw)
	if fileID == "" {
		return nil
	}
	media := map[string]any{
		"type":  blockType,
		"media": fileID,
	}
	if hasSpoiler == true {
		media["has_spoiler"] = true
	}
	return media
}

func richBlockFileID(mediaKey string, raw any) string {
	switch mediaKey {
	case "photo":
		return largestPhotoFileID(raw)
	default:
		return fileIDFromMediaObject(raw)
	}
}

func convertRichListBlock(block map[string]any) map[string]any {
	out := map[string]any{"type": "list"}
	if items, ok := block["items"].([]any); ok {
		out["items"] = convertRichListItemsToInput(items)
	}
	return out
}

func convertRichListItemsToInput(items []any) []any {
	out := make([]any, 0, len(items))
	for _, item := range items {
		itemMap, ok := item.(map[string]any)
		if !ok {
			continue
		}
		inputItem := map[string]any{}
		if blocks, ok := itemMap["blocks"].([]any); ok {
			inputItem["blocks"] = convertRichBlocksToInput(blocks)
		}
		copyIfPresentKeys(itemMap, inputItem, "has_checkbox", "is_checked", "value", "type")
		if len(inputItem) > 0 {
			out = append(out, inputItem)
		}
	}
	return out
}

func convertRichNestedBlocks(block map[string]any) map[string]any {
	blockType, _ := block["type"].(string)
	out := map[string]any{"type": blockType}
	if nested, ok := block["blocks"].([]any); ok {
		out["blocks"] = convertRichBlocksToInput(nested)
	}
	copyIfPresentKeys(block, out, "credit", "summary", "is_open")
	copyRichBlockCaption(block, out)
	return out
}

func convertRichTableBlock(block map[string]any) map[string]any {
	out := map[string]any{"type": "table"}
	if cells, ok := block["cells"].([]any); ok {
		out["cells"] = convertRichTableCells(cells)
	}
	copyIfPresentKeys(block, out, "is_bordered", "is_striped", "caption")
	return out
}

func convertRichTableCells(rows []any) []any {
	out := make([]any, 0, len(rows))
	for _, row := range rows {
		rowItems, ok := row.([]any)
		if !ok {
			continue
		}
		convertedRow := make([]any, 0, len(rowItems))
		for _, cell := range rowItems {
			cellMap, ok := cell.(map[string]any)
			if !ok {
				continue
			}
			convertedCell := map[string]any{}
			copyIfPresentKeys(cellMap, convertedCell, "text", "is_header", "colspan", "rowspan", "align", "valign")
			if len(convertedCell) > 0 {
				convertedRow = append(convertedRow, convertedCell)
			}
		}
		out = append(out, convertedRow)
	}
	return out
}

func convertRichMapBlock(block map[string]any) map[string]any {
	out := map[string]any{"type": "map"}
	copyIfPresentKeys(block, out, "location", "zoom", "width", "height")
	copyRichBlockCaption(block, out)
	return out
}

func convertRichTextBlock(block map[string]any) map[string]any {
	out := make(map[string]any, len(block))
	for k, v := range block {
		if isRichBlockOutputOnlyKey(k) {
			continue
		}
		out[k] = v
	}
	return out
}

func convertInputMediaFromOutput(media map[string]any) map[string]any {
	mediaType, _ := media["type"].(string)
	if mediaType == "" {
		if _, ok := media["photo"]; ok {
			mediaType = "photo"
		} else if _, ok := media["video"]; ok {
			mediaType = "video"
		}
	}
	switch mediaType {
	case "photo":
		fileID := richBlockFileID("photo", media["photo"])
		if fileID == "" {
			fileID, _ = media["media"].(string)
		}
		if fileID == "" {
			return nil
		}
		return map[string]any{"type": "photo", "media": fileID}
	case "video", "animation", "audio", "voice_note":
		key := mediaType
		fileID := richBlockFileID(key, media[key])
		if fileID == "" {
			fileID, _ = media["media"].(string)
		}
		if fileID == "" {
			return nil
		}
		return map[string]any{"type": mediaType, "media": fileID}
	default:
		if fileID, _ := media["media"].(string); fileID != "" {
			if t, _ := media["type"].(string); t != "" {
				return map[string]any{"type": t, "media": fileID}
			}
		}
		return stripRichBlockMap(media)
	}
}

func copyRichBlockCaption(block, out map[string]any) {
	if caption, exists := block["caption"]; exists {
		out["caption"] = caption
	}
	copyIfPresent(block, out, "caption_entities")
}

func copyIfPresentKeys(src, dst map[string]any, keys ...string) {
	for _, key := range keys {
		copyIfPresent(src, dst, key)
	}
}
