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

	prev, err := db.GetUserProState(params.UserID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Set user tier: %s: %s\n", params.UserID, err)

		if errors.Is(err, storage.ErrUserNotFound) {
			return operations.NewPostUserUserIDTierNotFound()
		}

		return operations.NewPostUserUserIDTierInternalServerError()
	}

	if err := db.SetUserProTerm(params.UserID, tier, paidUntil); err != nil {
		fmt.Fprintf(os.Stderr, "Set user tier: %s: %s\n", params.UserID, err)

		return operations.NewPostUserUserIDTierInternalServerError()
	}

	reviveProUser(db, params.UserID, prev, paidUntil)

	return operations.NewPostUserUserIDTierOK().WithPayload(proTermPayload(tier, paidUntil))
}

// ExtendUser - extend the paid period of a paid key by the requested months,
// counting from the current period end.
func ExtendUser(db *storage.BrigadeStorage, params operations.PostUserUserIDExtendParams, principal interface{}) middleware.Responder {
	if !db.IsPRO() {
		return operations.NewPostUserUserIDExtendForbidden()
	}

	prev, err := db.GetUserProState(params.UserID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Extend user: %s: %s\n", params.UserID, err)

		if errors.Is(err, storage.ErrUserNotFound) {
			return operations.NewPostUserUserIDExtendNotFound()
		}

		return operations.NewPostUserUserIDExtendInternalServerError()
	}

	if !storage.IsValidProTier(prev.Tier) {
		return operations.NewPostUserUserIDExtendBadRequest()
	}

	base := prev.PaidUntil
	if base.IsZero() || base.Before(time.Now().UTC()) {
		base = time.Now().UTC()
	}

	paidUntil := base.AddDate(0, int(swag.Int64Value(params.Body.Months)), 0)

	if err := db.SetUserProTerm(params.UserID, prev.Tier, paidUntil); err != nil {
		fmt.Fprintf(os.Stderr, "Extend user: %s: %s\n", params.UserID, err)

		return operations.NewPostUserUserIDExtendInternalServerError()
	}

	reviveProUser(db, params.UserID, prev, paidUntil)

	return operations.NewPostUserUserIDExtendOK().WithPayload(proTermPayload(prev.Tier, paidUntil))
}

// reviveProUser - if the key was blocked because its paid period had expired
// and the new period reaches into the future, unblock it. Manual and billing
// blocks are left alone.
func reviveProUser(db *storage.BrigadeStorage, id string, prev storage.UserProState, paidUntil time.Time) {
	if !prev.Blocked || prev.BlockReason != storage.ProBlockExpired || !paidUntil.After(time.Now().UTC()) {
		return
	}

	if err := db.UnblockUserPro(id); err != nil {
		fmt.Fprintf(os.Stderr, "Revive pro user: %s: %s\n", id, err)
	}
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

// ——— Billing (local stub invoices) ———

// GetProBilling - billing state of the PRO brigade.
func GetProBilling(db *storage.BrigadeStorage, params operations.GetProBillingParams, principal interface{}) middleware.Responder {
	if !db.IsPRO() {
		return operations.NewGetProBillingForbidden()
	}

	info, err := db.GetProBilling()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Get pro billing: %s\n", err)

		return operations.NewGetProBillingInternalServerError()
	}

	return operations.NewGetProBillingOK().WithPayload(proBillingPayload(info))
}

// GetProInvoices - the invoice history of the PRO brigade.
func GetProInvoices(db *storage.BrigadeStorage, params operations.GetProInvoicesParams, principal interface{}) middleware.Responder {
	if !db.IsPRO() {
		return operations.NewGetProInvoicesForbidden()
	}

	info, err := db.GetProBilling()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Get pro invoices: %s\n", err)

		return operations.NewGetProInvoicesInternalServerError()
	}

	payload := make([]*models.ProInvoice, 0, len(info.Invoices))
	for i := len(info.Invoices) - 1; i >= 0; i-- {
		payload = append(payload, proInvoicePayload(info.Invoices[i]))
	}

	return operations.NewGetProInvoicesOK().WithPayload(payload)
}

// PayProInvoice - stub payment of the current invoice: marks it paid and
// brings the keys blocked over billing back to life.
func PayProInvoice(db *storage.BrigadeStorage, params operations.PostProInvoicesCurrentPayParams, principal interface{}) middleware.Responder {
	if !db.IsPRO() {
		return operations.NewPostProInvoicesCurrentPayForbidden()
	}

	toUnblock, err := db.PayProInvoice(time.Now().UTC())
	if err != nil {
		fmt.Fprintf(os.Stderr, "Pay pro invoice: %s\n", err)

		return operations.NewPostProInvoicesCurrentPayInternalServerError()
	}

	for _, id := range toUnblock {
		if err := db.UnblockUserPro(id); err != nil {
			fmt.Fprintf(os.Stderr, "Pay pro invoice: unblock %s: %s\n", id, err)
		}
	}

	info, err := db.GetProBilling()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Pay pro invoice: %s\n", err)

		return operations.NewPostProInvoicesCurrentPayInternalServerError()
	}

	return operations.NewPostProInvoicesCurrentPayOK().WithPayload(proBillingPayload(info))
}

func proBillingPayload(info storage.ProBillingInfo) *models.ProBilling {
	state := info.State
	if state == storage.ProBillingPaid {
		state = "paid"
	}

	payload := &models.ProBilling{
		State: swag.String(state),
	}

	if info.Current != nil {
		payload.InvoiceID = info.Current.ID
		payload.TotalCents = info.Current.TotalCents
		payload.IssuedAt = (*strfmt.DateTime)(&info.Current.CreatedAt)
		payload.DueAt = (*strfmt.DateTime)(&info.Current.DueAt)
		suspendAt := info.Current.CreatedAt.AddDate(0, 0, storage.ProInvoiceGraceDays)
		payload.SuspendAt = (*strfmt.DateTime)(&suspendAt)
	}

	return payload
}

func proInvoicePayload(invoice storage.ProInvoice) *models.ProInvoice {
	payload := &models.ProInvoice{
		ID:         swag.String(invoice.ID),
		Status:     swag.String(invoice.Status),
		KeysCount:  swag.Int64(int64(invoice.KeysCount)),
		TotalCents: swag.Int64(invoice.TotalCents),
	}

	createdAt := invoice.CreatedAt
	dueAt := invoice.DueAt
	payload.CreatedAt = (*strfmt.DateTime)(&createdAt)
	payload.DueAt = (*strfmt.DateTime)(&dueAt)

	if !invoice.PaidAt.IsZero() {
		paidAt := invoice.PaidAt
		payload.PaidAt = (*strfmt.DateTime)(&paidAt)
	}

	return payload
}
