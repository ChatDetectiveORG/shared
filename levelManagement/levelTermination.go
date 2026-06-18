package levelmanagement

import (
	"context"
	"log"
	"strconv"
	"sync"
	"time"

	"github.com/ChatDetectiveORG/shared/constants"
	e "github.com/ChatDetectiveORG/shared/errors"
	postgresmodels "github.com/ChatDetectiveORG/shared/postgresModels"
	"github.com/ChatDetectiveORG/shared/telegram"

	amqp "github.com/rabbitmq/amqp091-go"
	tele "gopkg.in/telebot.v4"

	"github.com/ChatDetectiveORG/event-loop/src/infrastructure/config"
	postgresql "github.com/ChatDetectiveORG/event-loop/src/infrastructure/postgresql"
	h "github.com/ChatDetectiveORG/shared/handlers"
	"github.com/go-pg/pg/v10"
	"github.com/go-pg/pg/v10/orm"
)

func StartLevelTerminationLoop(ctx context.Context, interval time.Duration, config *config.Config) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	db := postgresql.GetDB()

	for {
		select {
		case <-ctx.Done():
			log.Printf("Level termination manager: stopping")
			return
		case <-ticker.C:
			err := notifyExpiration(db, config, expirationNotification{
				untilExpirationTime:    1 * time.Hour * 24,
				delta:                  interval / 2,
				untilExpirationTimeStr: "1 день",
			})
			if e.IsNonNil(err) {
				log.Println(err)
			}

			err = notifyExpiration(db, config, expirationNotification{
				untilExpirationTime:    3 * time.Hour * 24,
				delta:                  interval / 2,
				untilExpirationTimeStr: "3 дня",
			})
			if e.IsNonNil(err) {
				log.Println(err)
			}

			err = deleteExpired(db, config)
			if e.IsNonNil(err) {
				log.Println(err)
			}
		}
	}
}

type expirationNotification struct {
	untilExpirationTime time.Duration
	delta               time.Duration

	untilExpirationTimeStr string
}

func notifyExpiration(db *pg.DB, cfg *config.Config, notification expirationNotification) *e.ErrorInfo {
	taskCfg := taskCfg{
		UserIdSelectQuery: db.Model((*postgresmodels.UserLevels)(nil)).
			Column("user_levels.linked_user_id").
			Where("user_levels.until_timestamp >= ?", time.Now().Add(notification.untilExpirationTime).Add(-notification.delta)).
			Where("user_levels.until_timestamp <= ?", time.Now().Add(notification.untilExpirationTime).Add(notification.delta)),
		UserLevelSelectQuery: db.Model((*postgresmodels.UserLevels)(nil)).
			Where("user_levels.until_timestamp >= ?", time.Now().Add(notification.untilExpirationTime).Add(-notification.delta)).
			Where("user_levels.until_timestamp <= ?", time.Now().Add(notification.untilExpirationTime).Add(notification.delta)),
		UntilExpirationTimeStr: notification.untilExpirationTimeStr,
		Delete:                 false,

		Cfg: cfg,
	}

	return task(db, taskCfg)
}

func deleteExpired(db *pg.DB, cfg *config.Config) *e.ErrorInfo {
	taskCfg := taskCfg{
		UserIdSelectQuery: db.Model((*postgresmodels.UserLevels)(nil)).
			Column("user_levels.linked_user_id").
			Where("user_levels.until_timestamp <= ?", time.Now()),
		UserLevelSelectQuery: db.Model((*postgresmodels.UserLevels)(nil)).
			Where("user_levels.until_timestamp <= ?", time.Now()).
			Relation("LinkedUser"),
		Delete: true,

		Cfg: cfg,
	}

	return task(db, taskCfg)
}

type taskCfg struct {
	UserIdSelectQuery      *orm.Query
	UserLevelSelectQuery   *orm.Query
	UntilExpirationTimeStr string
	Delete                 bool

	Cfg *config.Config
}

func task(db *pg.DB, taskCfg taskCfg) *e.ErrorInfo {
	var userIDs [][]byte
	var errors []*e.ErrorInfo
	err := e.Wrap(taskCfg.UserIdSelectQuery.Select(&userIDs))
	if e.IsNonNil(err) {
		return err.PushStack()
	}

	conn, eRaw := amqp.DialConfig(taskCfg.Cfg.RabbitMQConfig.URL(), amqp.Config{Heartbeat: 10 * time.Second, Locale: "en_US", Dial: amqp.DefaultDial(10 * time.Second)})
	if e.IsNonNil(eRaw) {
		return e.FromError(eRaw, "failed to dial RabbitMQ")
	}
	defer conn.Close()

	ch, eRaw := conn.Channel()
	if e.IsNonNil(eRaw) {
		return e.FromError(eRaw, "failed to get channel")
	}
	defer ch.Close()

	pub, err := h.NewOutgoingPublisher(h.OutgoingConfig{
		Channel: ch,
		PodID:   taskCfg.Cfg.RuntimeConfig.PodID,
	})
	wg := sync.WaitGroup{}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	pub.Start(&wg, ctx)

	hashe := pub.NewHashe()

	for _, userID := range userIDs {
		var expireLevels []postgresmodels.UserLevels
		err := e.Wrap(
			taskCfg.UserLevelSelectQuery.Where("linked_user_id = ?", userID).Select(&expireLevels),
		)
		if e.IsNonNil(err) {
			return err.PushStack()
		}

		var expireAmount int
		var levelsGranted int

		for _, level := range expireLevels {
			expireAmount += level.Level

			if taskCfg.Delete {
				_, eRaw := db.Model(&level).WherePK().Delete()
				if e.IsNonNil(eRaw) {
					errors = append(errors, e.FromError(eRaw, "failed to delete user level"))
				}

				if level.IsReferralBonus {
					recountErr := func() *e.ErrorInfo {
						tx, eraw := db.Begin()
						if eraw != nil {
							return e.FromError(eraw, "failed to begin level recount transaction").WithSeverity(e.Critical)
						}
						defer tx.Rollback()

						user := level.LinkedUser
						if user == nil {
							user = &postgresmodels.Telegramuser{}
							eRaw := tx.Model(user).Where("id = ?", userID).Select()
							if e.IsNonNil(eRaw) {
								return e.FromError(eRaw, "failed to get linked user via level record. Maybe user deleted data?")
							}
						}

						untrackedRalations, err := GetUntrackedRelations(tx, user.ID)
						if e.IsNonNil(err) {
							return err.PushStack()
						}

						granted, err := RecountLevels(
							tx,
							untrackedRalations,
							user.ID,
						)
						if e.IsNonNil(err) {
							return err.PushStack()
						}
						levelsGranted += granted

						if eRaw := tx.Commit(); e.IsNonNil(eRaw) {
							return e.FromError(eRaw, "failed to commit level recount transaction")
						}
						return e.Nil()
					}()
					if e.IsNonNil(recountErr) {
						errors = append(errors, recountErr.PushStack())
					}
				}
			}
		}

		messageBuilder := telegram.MessageBuilder{}
		messageBuilder.WriteString(
			"⛔️", telegram.TextFormat{Type: telegram.Link}.WithCustomEmojiID("5463358164705489689"),
		)
		var doNotSendMessage bool
		if taskCfg.Delete {
			if levelsGranted == 0 {
				messageBuilder.WriteString(
					"Твой уровень был снижен на ",
				).WriteString(
					strconv.Itoa(expireAmount), telegram.TextFormat{Type: telegram.Bold},
				).AddButton(
					tele.InlineButton{Text: "Продлить", Data: "-"},
				)
			} else {
				delta := expireAmount - levelsGranted
				switch {
				case delta > 0:
					messageBuilder.WriteString(
						"Твой уровень был снижен на ",
					).WriteString(
						strconv.Itoa(delta), telegram.TextFormat{Type: telegram.Bold},
					).AddButton(
						tele.InlineButton{Text: "Продлить", Data: "-"},
					)
				case delta < 0:
					messageBuilder.WriteString(
						"После пересчёта приглашённых пользователей, твой уровень повышен на ",
					).WriteString(
						strconv.Itoa(-delta), telegram.TextFormat{Type: telegram.Bold},
					)
				default:
					doNotSendMessage = true
				}
			}
		} else {
			messageBuilder.WriteString(
				"Через "+taskCfg.UntilExpirationTimeStr+" твой уровень будет снижен на ",
			).WriteString(
				strconv.Itoa(expireAmount), telegram.TextFormat{Type: telegram.Bold},
			).AddButton(
				tele.InlineButton{Text: "Продлить", Data: "-"},
			)
		}

		// TODO: Add button to invoice level

		if !doNotSendMessage {
			tgID, tgErr := resolveTelegramUserID(db, userID, expireLevels)
			if e.IsNonNil(tgErr) {
				errors = append(errors, tgErr.PushStack())
				continue
			}

			hashe.Emit(constants.OutgoingRoutingKey, messageBuilder.Build(tgID))
		}
	}

	cancel()

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(10 * time.Second):
	}

	if len(errors) > 0 {
		return errors[0].PushStack()
	}

	return e.Nil()
}

func resolveTelegramUserID(db *pg.DB, userID []byte, levels []postgresmodels.UserLevels) (int64, *e.ErrorInfo) {
	for _, level := range levels {
		if level.LinkedUser != nil {
			return level.LinkedUser.GetTgId()
		}
	}

	user := &postgresmodels.Telegramuser{}
	if eRaw := db.Model(user).Where("id = ?", userID).Select(); e.IsNonNil(eRaw) {
		return 0, e.FromError(eRaw, "failed to get linked user for notification")
	}

	return user.GetTgId()
}
