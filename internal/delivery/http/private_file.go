package httpdelivery

// Private files (§19.2) — participant documents and payment proofs — live in
// buckets nothing can read without a signed URL. That URL is a bearer
// capability: for an hour, whoever holds it reads the object with no further
// authentication. Minting one is therefore the access decision itself, and this
// file is where the decision gets the facts it needs.
//
// The signer takes a domain identifier ("the document with this id") and looks
// up where it is stored. It used to take the bucket and path from the query
// string, so the caller chose the file and the server only checked that the
// caller was signed in — a participant could name another participant's path
// and be handed a URL to their passport.

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	invoicesvc "github.com/irfan-ghzl/pintour-travel/internal/application/invoice"
	"github.com/irfan-ghzl/pintour-travel/internal/domain/document"
)

// The kinds of private file the signer serves — the only two private objects
// stored per participant (§16.2).
const (
	fileKindDocument     = "document"
	fileKindPaymentProof = "payment_proof"
)

// Buckets behind those kinds. They are constants here, and not parameters, for
// the same reason the path is: nothing a caller sends may pick a bucket.
const (
	bucketParticipantDocuments = "participant-documents"
	bucketPaymentProofs        = "payment-proofs"
)

// signedURLTTL is how long a minted URL stays valid, per §19.2.
const signedURLTTL = 3600 // 1 hour, in seconds

// Sentinels, never shown to a client: privateFileError substitutes the
// Indonesian message each one gets answered with.
var (
	// errMissingFileRef means the request named no resource at all.
	errMissingFileRef = errors.New("no private file referenced")
	// errUnknownFileKind means the request named a category that does not exist.
	// That is a malformed request rather than a missing file, so it is reported
	// as one — no file's existence is disclosed by saying the kind is unknown.
	errUnknownFileKind = errors.New("unknown private file kind")
)

// privateFile is a stored object together with the participant it belongs to:
// everything an access decision needs, all of it resolved from the database and
// none of it taken from the request.
type privateFile struct {
	Bucket        string
	Path          string
	ParticipantID string
}

// privateFileResolver maps a (kind, id) pair onto the object behind it.
//
// Documents come straight from their repository; payment proofs go through the
// invoice service, because a proof carries no owner of its own and only that
// service holds both sides of the proof→invoice→participant chain.
type privateFileResolver struct {
	documents document.Repository
	invoices  *invoicesvc.Service
}

// resolve returns the stored object for kind and id. A row that exists but
// records no file path is reported as missing: there is nothing to sign, and
// signing the bucket root instead would hand out a URL to someone else's
// directory listing.
func (r privateFileResolver) resolve(ctx context.Context, kind, id string) (privateFile, error) {
	var file privateFile

	switch kind {
	case fileKindDocument:
		d, err := r.documents.GetByID(ctx, id)
		if err != nil {
			return privateFile{}, err
		}
		file = privateFile{
			Bucket:        bucketParticipantDocuments,
			Path:          d.FilePath,
			ParticipantID: d.ParticipantID,
		}
	case fileKindPaymentProof:
		proof, ownerID, err := r.invoices.GetProofWithOwner(ctx, id)
		if err != nil {
			return privateFile{}, err
		}
		file = privateFile{
			Bucket:        bucketPaymentProofs,
			Path:          proof.FilePath,
			ParticipantID: ownerID,
		}
	default:
		return privateFile{}, errUnknownFileKind
	}

	if file.Path == "" {
		return privateFile{}, sql.ErrNoRows
	}
	return file, nil
}

// isAbsoluteURL reports whether a stored file path is already a reachable
// address rather than an object inside a bucket. Deployments without storage
// keys record one of these instead of uploading, so both the ownership check and
// the signer have to recognise the shape — and must agree on it.
func isAbsoluteURL(path string) bool {
	return strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://")
}

// pathBelongsToParticipant reports whether objectPath is one participantID could
// have produced: StorageService.Upload files every object under the owner's id,
// so that prefix is what "my file" means in the bucket.
//
// This guards the other end of the same decision. Ownership is checked against
// the stored row, and a participant writes their own rows — both a document and
// a payment proof carry a file path the client supplies. Without this check the
// ownership check is satisfiable in two steps: record another participant's
// object as your own document, then ask for a URL to your own document.
func pathBelongsToParticipant(objectPath, participantID string) bool {
	if objectPath == "" || participantID == "" {
		return false
	}

	// The manual-URL fallback (deployment without storage keys) records an
	// absolute URL instead of a bucket path, so those are let through. Note what
	// that does and does not cover: such a URL reaches no private object — one in
	// a private bucket is unreadable without a signature — but it IS opened
	// verbatim when someone clicks through to the file, so a participant can
	// still point a reviewer at a site of their choosing. Narrowing that is a
	// change to the fallback flow itself, raised separately.
	if isAbsoluteURL(objectPath) {
		return true
	}

	// Anything percent-escaped is refused rather than decoded. Upload builds
	// every path from an id, a timestamp, and a sanitised filename, so a real one
	// never contains "%" — while "own-id/..%2fother-id/passport.jpg" carries the
	// right prefix, survives a literal-slash split intact, and still reaches
	// storage as a traversal.
	if strings.Contains(objectPath, "%") {
		return false
	}
	// A path that climbs out of its prefix is refused outright rather than
	// resolved: "own-id/../other-id/passport.jpg" carries the right prefix, and
	// the segments are collapsed later when the path is put into a request URL.
	for _, segment := range strings.Split(strings.ReplaceAll(objectPath, `\`, "/"), "/") {
		if segment == ".." {
			return false
		}
	}
	return strings.HasPrefix(objectPath, participantID+"/")
}
