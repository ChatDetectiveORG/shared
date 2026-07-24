package pgfixture

import (
	"encoding/json"
	"fmt"
	"strconv"
	"testing"
	"time"

	models "github.com/ChatDetectiveORG/shared/postgresModels"
	"github.com/ChatDetectiveORG/shared/telegram/rawmessage"
	u "github.com/ChatDetectiveORG/shared/utils"
	"github.com/go-pg/pg/v10"
	tele "gopkg.in/telebot.v4"
)

// BotUserSpec describes a business account owner stored in telegramusers.
type BotUserSpec struct {
	TelegramID           int64
	BusinessConnectionID string
}

// BotUserFixture is a seeded owner with the decrypted data key for assertions.
type BotUserFixture struct {
	User   *models.Telegramuser
	DataKey []byte
}

// SeedBotUser inserts a connected business owner with encrypted profile fields.
func SeedBotUser(t *testing.T, db *pg.DB, spec BotUserSpec) BotUserFixture {
	t.Helper()
	EnsureCryptoEnv(t)

	dataKey := []byte("01234567890123456789012345678901")
	masterKey, err := u.GetMasterkey()
	if !err.IsNil() {
		t.Fatalf("pgfixture: master key: %s", err.JSON())
	}

	encryptedKey, encErr := u.Encrypt(dataKey, masterKey)
	if !encErr.IsNil() {
		t.Fatalf("pgfixture: encrypt data key: %s", encErr.JSON())
	}

	idPlain := []byte(strconv.FormatInt(spec.TelegramID, 10))
	encryptedID, encErr := u.Encrypt(idPlain, dataKey)
	if !encErr.IsNil() {
		t.Fatalf("pgfixture: encrypt user id: %s", encErr.JSON())
	}

	idHash, hashErr := u.ToSecureHash(spec.TelegramID)
	if !hashErr.IsNil() {
		t.Fatalf("pgfixture: hash user id: %s", hashErr.JSON())
	}

	businessHash, hashErr := u.ToSecureHash(spec.BusinessConnectionID)
	if !hashErr.IsNil() {
		t.Fatalf("pgfixture: hash business connection: %s", hashErr.JSON())
	}

	referralCode, refErr := u.GenerateReferralCode(8)
	if refErr != nil {
		t.Fatalf("pgfixture: referral code: %v", refErr)
	}

	user := &models.Telegramuser{
		ID:                       encryptedID,
		IDHash:                   idHash,
		BusinessConnectionIDHash: businessHash,
		IsConnected:              true,
		DataEncryptionKey:        encryptedKey,
		ReferralCode:             referralCode,
		CreatedAt:                time.Now(),
		UpdatedAt:                time.Now(),
	}
	if _, insertErr := db.Model(user).Insert(); insertErr != nil {
		t.Fatalf("pgfixture: insert bot user: %v", insertErr)
	}

	settings := &models.UserSettings{LinkedUserID: encryptedID}
	if _, insertErr := db.Model(settings).Insert(); insertErr != nil {
		t.Fatalf("pgfixture: insert user settings: %v", insertErr)
	}

	return BotUserFixture{User: user, DataKey: append([]byte(nil), dataKey...)}
}

// BusinessMessageSpec describes a stored business chat message row.
type BusinessMessageSpec struct {
	BusinessConnectionID string
	CustomerChatID       int64
	MessageID            int
	Text                 string
	RawJSON              json.RawMessage
}

// SeedBusinessMessage stores encrypted raw-api metadata for a business message.
func SeedBusinessMessage(t *testing.T, db *pg.DB, owner BotUserFixture, spec BusinessMessageSpec) *models.Message {
	t.Helper()

	chatHash, err := u.ToSecureHash(spec.CustomerChatID)
	if !err.IsNil() {
		t.Fatalf("pgfixture: hash chat id: %s", err.JSON())
	}
	businessHash, err := u.ToSecureHash(spec.BusinessConnectionID)
	if !err.IsNil() {
		t.Fatalf("pgfixture: hash business connection: %s", err.JSON())
	}

	var rawPayload []byte
	switch {
	case len(spec.RawJSON) > 0:
		rawPayload = append([]byte(nil), spec.RawJSON...)
	default:
		marshaled, marshalErr := rawmessage.MarshalBusinessMessage(&tele.Message{
			ID:                   spec.MessageID,
			Text:                 spec.Text,
			BusinessConnectionID: spec.BusinessConnectionID,
			Chat:                 &tele.Chat{ID: spec.CustomerChatID},
		})
		if marshalErr != nil {
			t.Fatalf("pgfixture: marshal message metadata: %v", marshalErr)
		}
		rawPayload = marshaled
	}

	encryptedMetadata, encErr := u.Encrypt(rawPayload, owner.DataKey)
	if !encErr.IsNil() {
		t.Fatalf("pgfixture: encrypt metadata: %s", encErr.JSON())
	}

	message := &models.Message{
		ChatIDHash:               chatHash,
		MessageID:                spec.MessageID,
		BusinessConnectionIDHash: businessHash,
		Metadata:                 encryptedMetadata,
		MetadataFormat:           rawmessage.MetadataFormatRawAPIv1,
		CreatedAt:                time.Now(),
		UpdatedAt:                time.Now(),
	}
	if _, insertErr := db.Model(message).Insert(); insertErr != nil {
		t.Fatalf("pgfixture: insert message: %v", insertErr)
	}
	return message
}

// LoadMessageMetadata decrypts stored metadata to tele.Message for assertions.
func LoadMessageMetadata(t *testing.T, owner BotUserFixture, message *models.Message) *tele.Message {
	t.Helper()
	stored, legacy, loadErr := rawmessage.LoadStoredMessage(int(message.MetadataFormat), message.Metadata, owner.DataKey)
	if loadErr != nil {
		t.Fatalf("pgfixture: load metadata: %v", loadErr)
	}
	if legacy != nil {
		return legacy
	}
	parsed := &tele.Message{}
	if err := json.Unmarshal(stored.Payload, parsed); err != nil {
		t.Fatalf("pgfixture: unmarshal metadata: %v", err)
	}
	return parsed
}

// FormatSkipHint documents how to run chain tests locally.
func FormatSkipHint() string {
	return fmt.Sprintf("Set %s or port-forward dev postgres (%s) — see shared/testutil/README.md",
		envDatabaseURL, DefaultLocalDatabaseURL)
}
