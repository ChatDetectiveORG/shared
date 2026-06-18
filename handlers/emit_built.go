package handlers

import (
	"context"
	"time"

	e "github.com/ChatDetectiveORG/shared/errors"
	models "github.com/ChatDetectiveORG/shared/postgresModels"
	"github.com/ChatDetectiveORG/shared/telegram"
	"github.com/go-pg/pg/v10/orm"
	tele "gopkg.in/telebot.v4"
)

// EmitBuilt resolves mirror file assets, builds the message and sends it.
// For mirror bots it always waits for the send result so stale file_id values can be re-uploaded
// and cached for the mirror.
func (hch *HandlerChainHashe) EmitBuilt(
	ctx context.Context,
	db orm.DB,
	routingKey string,
	chatID int64,
	builder *telegram.MessageBuilder,
) *e.ErrorInfo {
	if builder == nil {
		return e.NewError("builder is nil", "EmitBuilt").WithSeverity(e.Warning)
	}

	mirrorAssets := builder.ConsumeMirrorFiles()
	if err := resolveMirrorFiles(builder, db, hch.MirrorID(), mirrorAssets); e.IsNonNil(err) {
		return err
	}

	msg := builder.Build(chatID)
	if msg == nil {
		return e.NewError("built message is nil", "EmitBuilt").WithSeverity(e.Warning)
	}

	if hch.MirrorID() == "" {
		return hch.Emit(routingKey, msg)
	}

	sent, waitErr := hch.EmitWait(ctx, routingKey, msg)
	if e.IsNonNil(waitErr) {
		return waitErr
	}

	return upsertMirrorFileCache(db, hch.MirrorID(), mirrorAssets, sent)
}

func resolveMirrorFiles(builder *telegram.MessageBuilder, db orm.DB, mirrorID string, assets []telegram.MirrorFileAsset) *e.ErrorInfo {
	for _, asset := range assets {
		fileID, err := resolveMirrorFileID(db, mirrorID, asset)
		if e.IsNonNil(err) {
			return err
		}
		builder.AddFile(fileID, asset.FallbackPath, asset.MimeType)
	}
	return e.Nil()
}

func resolveMirrorFileID(db orm.DB, mirrorID string, asset telegram.MirrorFileAsset) (string, *e.ErrorInfo) {
	if mirrorID == "" {
		return asset.PrimaryFileID, e.Nil()
	}
	if db == nil {
		return "", e.Nil()
	}

	parsedMirrorID, err := models.ParseMirrorID(mirrorID)
	if e.IsNonNil(err) {
		return "", err
	}

	cachedFileID, err := models.FindMirrorFileID(db, parsedMirrorID, asset.MirrorFileKey)
	if e.IsNonNil(err) {
		return "", err
	}

	return cachedFileID, e.Nil()
}

func upsertMirrorFileCache(db orm.DB, mirrorID string, assets []telegram.MirrorFileAsset, sent *tele.Message) *e.ErrorInfo {
	if db == nil || mirrorID == "" || len(assets) == 0 || sent == nil {
		return e.Nil()
	}

	parsedMirrorID, err := models.ParseMirrorID(mirrorID)
	if e.IsNonNil(err) {
		return err
	}

	fileID := telegram.ExtractSentFileID(sent)
	if fileID == "" {
		return e.Nil()
	}

	// Current flow supports one mirror asset per message.
	asset := assets[0]
	if asset.MirrorFileKey == "" {
		return e.Nil()
	}

	return models.UpsertMirrorFileID(db, parsedMirrorID, asset.MirrorFileKey, fileID, time.Now())
}
