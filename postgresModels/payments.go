package postgresmodels

import "time"

const (
	PaymentStatusInvoiceSent         = "invoice_sent"
	PaymentStatusPreCheckoutReceived = "pre_checkout_received"
	PaymentStatusApproved            = "approved"
	PaymentStatusCancelled           = "cancelled"
	PaymentStatusTimedOut            = "timed_out"
)

type Payment struct {
	ID int `pg:"id,pk,autoincrement"`

	ClientID []byte        `pg:"client_id,fk:telegram_user_id"`
	Client   *Telegramuser `pg:"rel:has-one,fk:client_id"`

	CreatedAt time.Time `pg:"created_at,default:now()"`
	UpdatedAt time.Time `pg:"updated_at,default:now()"`

	IsSuccess bool `pg:"is_success,default:false"`

	ServiceType        string `pg:"service_type"`
	Payload            string `pg:"payload,unique"`
	Currency           string `pg:"currency"`
	Amount             int    `pg:"amount"`
	PaymentMethod      string `pg:"payment_method"`
	InvoiceTitle       string `pg:"invoice_title"`
	InvoiceDescription string `pg:"invoice_description"`
	PriceLabel         string `pg:"price_label"`
	PreCheckoutID      string `pg:"pre_checkout_id"`
	Status             string `pg:"status"`
	UserSideError      string `pg:"user_side_error"`
}
