package notification

import "time"

// WANotification is a log of a WhatsApp message sent by the system.
type WANotification struct {
	ID             string      `json:"id"`
	RecipientPhone string      `json:"recipient_phone"`
	RecipientName  string      `json:"recipient_name"`
	MessageType    string      `json:"message_type"`
	MessageContent string      `json:"message_content"`
	ReferenceID    *string     `json:"reference_id,omitempty"`
	ReferenceType  *string     `json:"reference_type,omitempty"`
	Status         string      `json:"status"`
	FonnteResponse interface{} `json:"fonnte_response,omitempty"`
	SentAt         *time.Time  `json:"sent_at,omitempty"`
	CreatedAt      time.Time   `json:"created_at"`
}

// Message types (template codes)
const (
	TypeLeadsWelcome      = "LEADS_WELCOME"
	TypeLeadsNotifSales   = "LEADS_NOTIF_SALES"
	TypeInvoiceSent       = "INVOICE_SENT"
	TypePaymentReminder1  = "PAYMENT_REMINDER_1"
	TypePaymentReminder3  = "PAYMENT_REMINDER_3"
	TypeDocRequest        = "DOC_REQUEST"
	TypeDocRejected       = "DOC_REJECTED"
	TypeReminderH30       = "REMINDER_H30"
	TypeReminderH14       = "REMINDER_H14"
	TypeReminderH7        = "REMINDER_H7"
	TypeReminderH1        = "REMINDER_H1"
	TypeDepartureConfirm  = "DEPARTURE_CONFIRM"
)
