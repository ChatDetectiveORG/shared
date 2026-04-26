package postgresmodels

import (
	"time"

	e "github.com/ChatDetectiveORG/shared/errors"
	"github.com/go-pg/pg/v10"
	"github.com/go-pg/pg/v10/orm"
)

type MirrorFile struct {
	ID int `pg:"id,pk"`

	MirrorID int     `pg:"mirror_id,unique:mirror_file"`
	Mirror   *Mirror `pg:"rel:has-one,fk:mirror_id"`

	FileKey string `pg:"file_key,unique:mirror_file"`
	FileID  string `pg:"file_id"`

	CreatedAt time.Time `pg:"created_at,default:now()"`
	UpdatedAt time.Time `pg:"updated_at,default:now()"`
}

func FindMirrorFileID(db orm.DB, mirrorID int, fileKey string) (string, *e.ErrorInfo) {
	if mirrorID == 0 || fileKey == "" {
		return "", e.Nil()
	}

	file := &MirrorFile{}
	err := db.Model(file).
		Where("mirror_id = ?", mirrorID).
		Where("file_key = ?", fileKey).
		Select()
	if err == pg.ErrNoRows {
		return "", e.Nil()
	}
	if err != nil {
		return "", e.FromError(err, "failed to get mirror file").WithSeverity(e.Notice)
	}
	return file.FileID, e.Nil()
}

func UpsertMirrorFileID(db orm.DB, mirrorID int, fileKey string, fileID string, now time.Time) *e.ErrorInfo {
	if mirrorID == 0 || fileKey == "" || fileID == "" {
		return e.Nil()
	}

	file := &MirrorFile{
		MirrorID:  mirrorID,
		FileKey:   fileKey,
		FileID:    fileID,
		CreatedAt: now,
		UpdatedAt: now,
	}
	_, err := db.Model(file).
		OnConflict("(mirror_id, file_key) DO UPDATE").
		Set("file_id = EXCLUDED.file_id").
		Set("updated_at = EXCLUDED.updated_at").
		Insert()
	if err != nil {
		return e.FromError(err, "failed to upsert mirror file").WithSeverity(e.Notice)
	}
	return e.Nil()
}
