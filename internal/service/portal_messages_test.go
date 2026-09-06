package service

// Tiket 03 — the portal messages have to promise only what the portal can
// actually do at the moment they arrive.
//
// Asserted on the rendered text, the way the credential message has been tested
// since it was written: the content IS the feature here, and a message whose
// wording is wrong still sends successfully and still logs as delivered.

import (
	"strings"
	"testing"
)

// mentions fails the test unless the message contains every phrase, matched
// case-insensitively so a sentence can be recapitalised without breaking this.
func mentions(t *testing.T, label, msg string, phrases ...string) {
	t.Helper()
	lower := strings.ToLower(msg)
	for _, want := range phrases {
		if !strings.Contains(lower, strings.ToLower(want)) {
			t.Errorf("%s tidak menyebut %q:\n%s", label, want, msg)
		}
	}
}

// The message a participant receives the minute they are converted. It used to
// invite them to "gunakan portal untuk mengunggah dokumen" while the portal was
// still shut — on the test data it arrived seventeen minutes early. Now the
// invitation is true, and it has to name the thing that matters most: paying.
func TestPortalCredentialsMessage_NamesWhatIsOpenNow(t *testing.T) {
	msg := portalCredentialsMessage("Budi", "628111000001", "Xk7mQp2r",
		"https://pintour.test/portal")

	mentions(t, "pesan kredensial", msg,
		// still the only copy of the credential that ever leaves the system
		"Xk7mQp2r", "628111000001", "https://pintour.test/portal", "Budi",
		// what they can do the moment they log in
		"invoice", "bayar", "bukti transfer", "dokumen",
		// and what follows, so a locked menu does not read as a broken one
		"setelah pembayaran dikonfirmasi", "itinerary", "briefing")
}

// The confirmation message still goes out, but what it announces has changed:
// the portal was already open, so what payment adds is the travel content.
// Telling a participant their portal is "now active" would leave them wondering
// what they had been using for the past three weeks.
func TestPortalActivatedMessage_AnnouncesTheContentNotTheAccess(t *testing.T) {
	msg := portalActivatedMessage("Budi", "628111000001", "https://pintour.test/portal")

	mentions(t, "pesan portal aktif", msg,
		"Budi", "628111000001", "https://pintour.test/portal",
		"pembayaran", "itinerary", "briefing", "tour leader")
}

// The invoice message hands over a link the reader can follow now. It used to
// end with "(tersedia di portal setelah pembayaran dikonfirmasi)", which was the
// honest thing to say while the portal was shut and is the wrong thing to say
// now that the PDF is downloadable from the moment the invoice is issued.
func TestInvoiceMessage_LinksToAPDFThatCanBeFetchedNow(t *testing.T) {
	msg := invoiceMessage("Budi", "INV-202608-0001", "25.000.000", "20 Agu 2026",
		"https://pintour.test/portal/invoices")

	mentions(t, "pesan invoice", msg,
		"INV-202608-0001", "25.000.000", "20 Agu 2026",
		"https://pintour.test/portal/invoices")
	if strings.Contains(strings.ToLower(msg), "setelah pembayaran") {
		t.Errorf("pesan invoice masih menunda PDF sampai pembayaran:\n%s", msg)
	}
}
