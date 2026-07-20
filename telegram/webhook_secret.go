package telegram

import "os"

// WebhookSecretEnv is the environment variable holding the shared secret that Telegram
// echoes back in the X-Telegram-Bot-Api-Secret-Token header of every webhook request.
const WebhookSecretEnv = "TELEGRAM_WEBHOOK_SECRET"

// WebhookSecretHeader is the header Telegram sets when secret_token was passed to setWebhook.
const WebhookSecretHeader = "X-Telegram-Bot-Api-Secret-Token"

// WebhookSecret returns the configured webhook secret token ("" when not configured).
// All SetWebhook call sites (main bot and mirrors) must register with this secret so the
// api-gateway can authenticate every incoming webhook request.
func WebhookSecret() string {
	return os.Getenv(WebhookSecretEnv)
}
