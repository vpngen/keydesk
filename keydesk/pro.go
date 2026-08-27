package keydesk

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/go-openapi/runtime/middleware"
	"github.com/go-openapi/strfmt"
	"github.com/go-openapi/swag"
	"github.com/vpngen/keydesk/gen/models"
	"github.com/vpngen/keydesk/gen/restapi/operations"
	"github.com/vpngen/keydesk/keydesk/storage"
)

// PRO per-key operations. All of them are allowed only for PRO brigades, so
// free and VIP brigades are not affected by this API surface at all.

// UpdateUserPro - partial update of the brigadier's key annotations
// (label / note / sold-for).
func UpdateUserPro(db *storage.BrigadeStorage, params operations.PatchUserUserIDProParams, principal interface{}) middleware.Responder {
	if !db.IsPRO() {
		return operations.NewPatchUserUserIDProForbidden()
	}

	meta := storage.UserProMeta{
		Label:   params.Body.Label,
		Note:    params.Body.Note,
		SoldFor: params.Body.SoldForCents,
	}

	if err := db.UpdateUserProMeta(params.UserID, meta); err != nil {
		fmt.Fprintf(os.Stderr, "Update user pro meta: %s: %s\n", params.UserID, err)

		if errors.Is(err, storage.ErrUserNotFound) {
			return operations.NewPatchUserUserIDProNotFound()
		}

		return operations.NewPatchUserUserIDProInternalServerError()
	}

	return operations.NewPatchUserUserIDProOK()
}

// SetUserTier - change the tier of a key. A paid tier starts a new paid
// period of the requested months from now; "free" resets the key back to a
// regular free key.
func SetUserTier(db *storage.BrigadeStorage, params operations.PostUserUserIDTierParams, principal interface{}) middleware.Responder {
	if !db.IsPRO() {
		return operations.NewPostUserUserIDTierForbidden()
	}

	tier := swag.StringValue(params.Body.Tier)
	months := int(params.Body.Months)

	var paidUntil time.Time

	switch {
	case tier == storage.TierFree || tier == "free":
		tier = storage.TierFree
	case storage.IsValidProTier(tier):
		if months == 0 {
			months = 1
		}

		paidUntil = time.Now().UTC().AddDate(0, months, 0)
	default:
		return operations.NewPostUserUserIDTierBadRequest()
	}

	if err := db.SetUserProTerm(params.UserID, tier, paidUntil); err != nil {
		fmt.Fprintf(os.Stderr, "Set user tier: %s: %s\n", params.UserID, err)

		if errors.Is(err, storage.ErrUserNotFound) {
			return operations.NewPostUserUserIDTierNotFound()
		}

		return operations.NewPostUserUserIDTierInternalServerError()
	}

	return operations.NewPostUserUserIDTierOK().WithPayload(proTermPayload(tier, paidUntil))
}

// ExtendUser - extend the paid period of a paid key by the requested months,
// counting from the current period end.
func ExtendUser(db *storage.BrigadeStorage, params operations.PostUserUserIDExtendParams, principal interface{}) middleware.Responder {
	if !db.IsPRO() {
		return operations.NewPostUserUserIDExtendForbidden()
	}

	tier, paidUntil, err := db.GetUserProTerm(params.UserID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Extend user: %s: %s\n", params.UserID, err)

		if errors.Is(err, storage.ErrUserNotFound) {
			return operations.NewPostUserUserIDExtendNotFound()
		}

		return operations.NewPostUserUserIDExtendInternalServerError()
	}

	if !storage.IsValidProTier(tier) {
		return operations.NewPostUserUserIDExtendBadRequest()
	}

	base := paidUntil
	if base.IsZero() {
		base = time.Now().UTC()
	}

	paidUntil = base.AddDate(0, int(swag.Int64Value(params.Body.Months)), 0)

	if err := db.SetUserProTerm(params.UserID, tier, paidUntil); err != nil {
		fmt.Fprintf(os.Stderr, "Extend user: %s: %s\n", params.UserID, err)

		return operations.NewPostUserUserIDExtendInternalServerError()
	}

	return operations.NewPostUserUserIDExtendOK().WithPayload(proTermPayload(tier, paidUntil))
}

func proTermPayload(tier string, paidUntil time.Time) *models.UserProTerm {
	payload := &models.UserProTerm{
		Tier: swag.String(proTierAPIValue(tier)),
	}

	if !paidUntil.IsZero() {
		payload.PaidUntil = (*strfmt.DateTime)(&paidUntil)
	}

	return payload
}

// proTierAPIValue - the storage keeps "" for free keys, the API talks "free".
func proTierAPIValue(tier string) string {
	if tier == storage.TierFree {
		return "free"
	}

	return tier
}
