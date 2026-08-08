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
	TypeLeadsWelcome     = "LEADS_WELCOME"
	TypeLeadsNotifSales  = "LEADS_NOTIF_SALES"
	TypeInvoiceSent      = "INVOICE_SENT"
	TypePaymentReminder1 = "PAYMENT_REMINDER_1"
	TypePaymentReminder3 = "PAYMENT_REMINDER_3"
	// TypePaymentReminder6 completes FR-INV-05's H+1/H+3/H+6 series. The day-6
	// reminder was already being sent; without a type of its own it was logged
	// as a day-1 reminder, so the notification history said something that had
	// not happened.
	TypePaymentReminder6 = "PAYMENT_REMINDER_6"
	TypeDocRequest       = "DOC_REQUEST"
	TypeDocRejected      = "DOC_REJECTED"
	TypeReminderH30      = "REMINDER_H30"
	TypeReminderH14      = "REMINDER_H14"
	TypeReminderH7       = "REMINDER_H7"
	TypeReminderH1       = "REMINDER_H1"
	TypeDepartureConfirm = "DEPARTURE_CONFIRM"
	// TypePortalCredentials carries the temporary portal password to a newly
	// converted participant (FR-PORTAL-01). It is the only message whose content
	// nobody can reissue: portal accounts have no password-reset flow, so a
	// participant who never receives it cannot get in at all.
	TypePortalCredentials = "PORTAL_CREDENTIALS"
	// Automasi (prompt §3.1 / §2) — adapted to existing password-based portal.
	TypePortalActivated   = "PORTAL_ACTIVATED"
	TypePaymentReceived   = "PAYMENT_RECEIVED"
	TypePaymentRejected   = "PAYMENT_REJECTED"
	TypePaymentOverdue    = "PAYMENT_OVERDUE"
	TypeDocApproved       = "DOC_APPROVED"
	TypeBriefingActivated = "BRIEFING_ACTIVATED"
	TypeLeadsStale        = "LEADS_STALE"
	TypeQuotaWarning      = "QUOTA_WARNING"
)
