package main

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/textproto"
	"strconv"
	"strings"
	"time"

	amodels "github.com/ghotso/libredesk/internal/auth/models"
	cmodels "github.com/ghotso/libredesk/internal/conversation/models"
	smodels "github.com/ghotso/libredesk/internal/conversation/status/models"
	"github.com/ghotso/libredesk/internal/envelope"
	inboxemail "github.com/ghotso/libredesk/internal/inbox/channel/email"
	imodels "github.com/ghotso/libredesk/internal/inbox/models"
	medModels "github.com/ghotso/libredesk/internal/media/models"
	notifier "github.com/ghotso/libredesk/internal/notification"
	tmpl "github.com/ghotso/libredesk/internal/template"
	"github.com/ghotso/libredesk/internal/user/models"
	"github.com/knadh/smtppool"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
)

// portalAuth validates the session for a contact and ensures portal is enabled.
// Rejects if portal is disabled, session invalid, user is not a contact, or contact is disabled.
// Sets "user" (amodels.User) and "contact" (models.User) in request context.
func portalAuth(handler fastglue.FastRequestHandler) fastglue.FastRequestHandler {
	return func(r *fastglue.Request) error {
		app := r.Context.(*App)

		// Check if portal is enabled.
		settingsJSON, err := app.setting.GetByPrefix("app")
		if err != nil {
			app.lo.Error("error fetching app settings for portal", "error", err)
			return r.SendErrorEnvelope(http.StatusInternalServerError, app.i18n.Ts("globals.messages.errorFetching", "name", app.i18n.T("globals.terms.setting")), nil, envelope.GeneralError)
		}
		var settings map[string]interface{}
		if err := json.Unmarshal(settingsJSON, &settings); err != nil {
			app.lo.Error("error unmarshalling app settings", "error", err)
			return r.SendErrorEnvelope(http.StatusInternalServerError, app.i18n.Ts("globals.messages.errorFetching", "name", app.i18n.T("globals.terms.setting")), nil, envelope.GeneralError)
		}
		portalEnabled, _ := settings["app.portal_enabled"].(bool)
		if !portalEnabled {
			return r.SendErrorEnvelope(http.StatusForbidden, app.i18n.T("auth.invalidOrExpiredSession"), nil, envelope.PermissionError)
		}

		// Validate portal (contact) session — uses libredesk_portal_session cookie only.
		sessUser, err := app.auth.ValidatePortalSession(r)
		if err != nil || sessUser.ID <= 0 {
			app.lo.Error("error validating portal session", "error", err)
			return r.SendErrorEnvelope(http.StatusUnauthorized, app.i18n.T("auth.invalidOrExpiredSession"), nil, envelope.GeneralError)
		}

		// Must be a contact session.
		if sessUser.Type != models.UserTypeContact {
			return r.SendErrorEnvelope(http.StatusUnauthorized, app.i18n.T("auth.invalidOrExpiredSession"), nil, envelope.GeneralError)
		}

		// Load full contact user.
		contact, err := app.user.Get(sessUser.ID, "", models.UserTypeContact)
		if err != nil {
			return sendErrorEnvelope(r, err)
		}

		if !contact.Enabled {
			return r.SendErrorEnvelope(http.StatusUnauthorized, app.i18n.T("user.accountDisabled"), nil, envelope.PermissionError)
		}

		// Set context for portal handlers.
		r.RequestCtx.SetUserValue("user", amodels.User{
			ID:        contact.ID,
			Email:     contact.Email.String,
			FirstName: contact.FirstName,
			LastName:  contact.LastName,
			UserType:  models.UserTypeContact,
		})
		r.RequestCtx.SetUserValue("contact", contact)

		return handler(r)
	}
}

// handlePortalLogin handles POST /api/v1/portal/auth/login (email + password).
func handlePortalLogin(r *fastglue.Request) error {
	app := r.Context.(*App)

	// Check if portal is enabled.
	settingsJSON, err := app.setting.GetByPrefix("app")
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	var settings map[string]interface{}
	if err := json.Unmarshal(settingsJSON, &settings); err != nil {
		app.lo.Error("error unmarshalling app settings", "error", err)
		return r.SendErrorEnvelope(http.StatusInternalServerError, app.i18n.Ts("globals.messages.errorFetching", "name", app.i18n.T("globals.terms.setting")), nil, envelope.GeneralError)
	}
	portalEnabled, _ := settings["app.portal_enabled"].(bool)
	if !portalEnabled {
		return r.SendErrorEnvelope(http.StatusForbidden, app.i18n.T("auth.invalidOrExpiredSession"), nil, envelope.PermissionError)
	}

	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := r.Decode(&req, "json"); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.errorParsing", "name", "{globals.terms.request}"), nil, envelope.InputError)
	}
	if req.Email == "" || req.Password == "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("globals.messages.badRequest"), nil, envelope.InputError)
	}

	user, err := app.user.VerifyPasswordForContact(req.Email, []byte(req.Password))
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	if err := app.auth.SavePortalSession(amodels.User{
		ID:        user.ID,
		Email:     user.Email.String,
		FirstName: user.FirstName,
		LastName:  user.LastName,
		UserType:  models.UserTypeContact,
	}, r); err != nil {
		app.lo.Error("error saving portal session", "error", err)
		return sendErrorEnvelope(r, envelope.NewError(envelope.GeneralError, app.i18n.Ts("globals.messages.errorSaving", "name", "{globals.terms.session}"), nil))
	}
	if err := app.auth.SetCSRFCookie(r); err != nil {
		app.lo.Error("error setting csrf cookie", "error", err)
	}

	// Return contact info (no sensitive fields).
	out := map[string]interface{}{
		"id":         user.ID,
		"email":      user.Email.String,
		"first_name": user.FirstName,
		"last_name":  user.LastName,
	}
	return r.SendEnvelope(out)
}

// handlePortalLogout handles POST /api/v1/portal/auth/logout.
func handlePortalLogout(r *fastglue.Request) error {
	app := r.Context.(*App)
	if err := app.auth.DestroyPortalSession(r); err != nil {
		app.lo.Error("error destroying portal session", "error", err)
	}
	r.RequestCtx.Response.Header.Add("Cache-Control", "no-store, no-cache, must-revalidate")
	r.RequestCtx.Response.Header.Add("Pragma", "no-cache")
	r.RequestCtx.Response.Header.Add("Expires", "-1")
	return r.SendEnvelope(map[string]string{"ok": "true"})
}

// handlePortalMe returns the current contact (portalAuth required).
func handlePortalMe(r *fastglue.Request) error {
	app := r.Context.(*App)
	contact, ok := r.RequestCtx.UserValue("contact").(models.User)
	if !ok {
		return r.SendErrorEnvelope(http.StatusUnauthorized, app.i18n.T("auth.invalidOrExpiredSession"), nil, envelope.GeneralError)
	}
	out := map[string]interface{}{
		"id":         contact.ID,
		"email":      contact.Email.String,
		"first_name": contact.FirstName,
		"last_name":  contact.LastName,
	}
	return r.SendEnvelope(out)
}

// handlePortalSetPassword handles POST /api/v1/portal/auth/set-password (token + new password from set-password link).
func handlePortalSetPassword(r *fastglue.Request) error {
	app := r.Context.(*App)

	// Portal must be enabled.
	settingsJSON, err := app.setting.GetByPrefix("app")
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	var settings map[string]interface{}
	if err := json.Unmarshal(settingsJSON, &settings); err != nil {
		return r.SendErrorEnvelope(http.StatusInternalServerError, app.i18n.Ts("globals.messages.errorFetching", "name", app.i18n.T("globals.terms.setting")), nil, envelope.GeneralError)
	}
	portalEnabled, _ := settings["app.portal_enabled"].(bool)
	if !portalEnabled {
		return r.SendErrorEnvelope(http.StatusForbidden, app.i18n.T("auth.invalidOrExpiredSession"), nil, envelope.PermissionError)
	}

	var req struct {
		Token    string `json:"token"`
		Password string `json:"password"`
	}
	if err := r.Decode(&req, "json"); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.errorParsing", "name", "{globals.terms.request}"), nil, envelope.InputError)
	}
	if req.Token == "" || req.Password == "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("globals.messages.badRequest"), nil, envelope.InputError)
	}

	if err := app.user.ResetPassword(req.Token, req.Password); err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(map[string]string{"ok": "true"})
}

// sendPortalEmail sends an email using the portal default inbox's SMTP (same as conversation emails).
func sendPortalEmail(app *App, msg notifier.Message) error {
	settingsJSON, err := app.setting.GetByPrefix("app")
	if err != nil {
		return err
	}
	var settings map[string]interface{}
	if err := json.Unmarshal(settingsJSON, &settings); err != nil {
		return err
	}
	pid, _ := settings["app.portal_default_inbox_id"].(float64)
	inboxID := int(pid)
	if inboxID <= 0 {
		return envelope.NewError(envelope.InputError, "portal default inbox not set", nil)
	}
	inboxRecord, err := app.inbox.GetDBRecord(inboxID)
	if err != nil {
		return err
	}
	var config imodels.Config
	if err := json.Unmarshal(inboxRecord.Config, &config); err != nil {
		return err
	}
	if len(config.SMTP) == 0 {
		return envelope.NewError(envelope.GeneralError, "portal default inbox has no SMTP configured", nil)
	}
	from := strings.TrimSpace(inboxRecord.From)
	if from == "" {
		from = strings.TrimSpace(config.From)
	}
	if from == "" {
		return envelope.NewError(envelope.GeneralError, "portal default inbox has no From address", nil)
	}
	pools, err := inboxemail.NewSmtpPool(config.SMTP, config.OAuth)
	if err != nil {
		return err
	}
	defer func() {
		for _, p := range pools {
			p.Close()
		}
	}()
	em := smtppool.Email{
		From:    from,
		To:      msg.RecipientEmails,
		Subject: msg.Subject,
		Headers: textproto.MIMEHeader{},
	}
	if msg.ContentType == "plain" {
		em.Text = []byte(msg.Content)
	} else {
		em.HTML = []byte(msg.Content)
		if msg.AltContent != "" {
			em.Text = []byte(msg.AltContent)
		}
	}
	return pools[0].Send(em)
}

// handlePortalForgotPassword handles POST /api/v1/portal/auth/forgot-password (email). Sends set-password link to the contact if found.
func handlePortalForgotPassword(r *fastglue.Request) error {
	app := r.Context.(*App)

	settingsJSON, err := app.setting.GetByPrefix("app")
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	var settings map[string]interface{}
	if err := json.Unmarshal(settingsJSON, &settings); err != nil {
		return r.SendErrorEnvelope(http.StatusInternalServerError, app.i18n.Ts("globals.messages.errorFetching", "name", app.i18n.T("globals.terms.setting")), nil, envelope.GeneralError)
	}
	portalEnabled, _ := settings["app.portal_enabled"].(bool)
	if !portalEnabled {
		return r.SendErrorEnvelope(http.StatusForbidden, app.i18n.T("auth.invalidOrExpiredSession"), nil, envelope.PermissionError)
	}

	var req struct {
		Email string `json:"email"`
	}
	if err := r.Decode(&req, "json"); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.errorParsing", "name", "{globals.terms.request}"), nil, envelope.InputError)
	}
	email := strings.TrimSpace(strings.ToLower(req.Email))
	if email == "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.empty", "name", "`email`"), nil, envelope.InputError)
	}

	contact, err := app.user.Get(0, email, models.UserTypeContact)
	if err != nil {
		var envErr envelope.Error
		if errors.As(err, &envErr) && envErr.ErrorType == envelope.NotFoundError {
			return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("portal.noAccountWithEmail"), nil, envelope.InputError)
		}
		return sendErrorEnvelope(r, err)
	}
	if !contact.Enabled {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("portal.noAccountWithEmail"), nil, envelope.InputError)
	}

	token, err := app.user.SetResetPasswordTokenForContact(contact.ID)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	content, err := app.tmpl.RenderInMemoryTemplate(tmpl.TmplPortalResetPassword, map[string]string{
		"ResetToken": token,
	})
	if err != nil {
		app.lo.Error("error rendering portal reset password template", "error", err)
		return r.SendErrorEnvelope(http.StatusInternalServerError, app.i18n.T("globals.messages.errorSendingPasswordResetEmail"), nil, envelope.GeneralError)
	}
	if err := sendPortalEmail(app, notifier.Message{
		RecipientEmails: []string{contact.Email.String},
		Subject:         app.i18n.T("portal.resetPasswordEmailSubject"),
		Content:         content,
		Provider:        notifier.ProviderEmail,
	}); err != nil {
		app.lo.Error("error sending portal reset password email", "error", err)
		return r.SendErrorEnvelope(http.StatusInternalServerError, app.i18n.T("globals.messages.errorSendingPasswordResetEmail"), nil, envelope.GeneralError)
	}
	return r.SendEnvelope(map[string]string{"ok": "true"})
}

// portalContactOrgID returns the contact's organization ID when the contact has membership; otherwise 0.
func portalContactOrgID(app *App, contactID int) (int, error) {
	mem, ok, err := app.organization.GetMembershipForContact(int64(contactID))
	if err != nil {
		return 0, err
	}
	if !ok {
		return 0, nil
	}
	return mem.OrganizationID, nil
}

// contactCanAccessConversation returns true if the contact can access the conversation (owner or org-shared).
func contactCanAccessConversation(conv cmodels.Conversation, contactID int, orgID int) bool {
	if conv.ContactID == contactID {
		return true
	}
	if orgID > 0 && conv.OrganizationID.Valid && int(conv.OrganizationID.Int) == orgID {
		return true
	}
	return false
}

// handlePortalListConversations returns conversations visible to the contact (portalAuth).
func handlePortalListConversations(r *fastglue.Request) error {
	app := r.Context.(*App)
	contact, _ := r.RequestCtx.UserValue("contact").(models.User)

	orgID, err := portalContactOrgID(app, contact.ID)
	if err != nil {
		app.lo.Error("error getting contact org for portal list", "error", err)
		return r.SendErrorEnvelope(http.StatusInternalServerError, app.i18n.Ts("globals.messages.errorFetching", "name", "{globals.terms.conversation}"), nil, envelope.GeneralError)
	}

	page, _ := strconv.Atoi(string(r.RequestCtx.QueryArgs().Peek("page")))
	pageSize, _ := strconv.Atoi(string(r.RequestCtx.QueryArgs().Peek("page_size")))
	if pageSize <= 0 {
		pageSize = 20
	}
	if page < 1 {
		page = 1
	}

	list, total, err := app.conversation.GetConversationsForContact(contact.ID, orgID, page, pageSize)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	out := map[string]interface{}{
		"conversations": list,
		"total":         total,
		"page":          page,
		"page_size":     pageSize,
	}
	return r.SendEnvelope(out)
}

// handlePortalGetConversation returns a single conversation by uuid if the contact can access it (portalAuth). Messages exclude private notes.
func handlePortalGetConversation(r *fastglue.Request) error {
	app := r.Context.(*App)
	contact, _ := r.RequestCtx.UserValue("contact").(models.User)
	uuid := r.RequestCtx.UserValue("uuid").(string)

	conv, err := app.conversation.GetConversation(0, uuid, "")
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	orgID, err := portalContactOrgID(app, contact.ID)
	if err != nil {
		app.lo.Error("error getting contact org for portal detail", "error", err)
		return r.SendErrorEnvelope(http.StatusInternalServerError, app.i18n.Ts("globals.messages.errorFetching", "name", "{globals.terms.conversation}"), nil, envelope.GeneralError)
	}

	if !contactCanAccessConversation(conv, contact.ID, orgID) {
		return r.SendErrorEnvelope(http.StatusNotFound, app.i18n.Ts("globals.messages.notFound", "name", "{globals.terms.conversation}"), nil, envelope.NotFoundError)
	}

	// Get messages excluding private (portal must not see agent-only notes).
	privateFalse := false
	messages, _, err := app.conversation.GetConversationMessages(uuid, 1, 100, &privateFalse, nil)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	out := map[string]interface{}{
		"conversation": conv,
		"messages":     messages,
	}
	return r.SendEnvelope(out)
}

// handlePortalCreateConversation creates a new conversation (portalAuth). Uses portal_default_inbox_id and contact's org/share settings.
func handlePortalCreateConversation(r *fastglue.Request) error {
	app := r.Context.(*App)
	contact, _ := r.RequestCtx.UserValue("contact").(models.User)

	settingsJSON, err := app.setting.GetByPrefix("app")
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	var settings map[string]interface{}
	if err := json.Unmarshal(settingsJSON, &settings); err != nil {
		return r.SendErrorEnvelope(http.StatusInternalServerError, app.i18n.Ts("globals.messages.errorFetching", "name", app.i18n.T("globals.terms.setting")), nil, envelope.GeneralError)
	}
	defaultInboxID, _ := settings["app.portal_default_inbox_id"].(float64) // JSON number
	if defaultInboxID == 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.required", "name", "portal default inbox"), nil, envelope.InputError)
	}
	inboxID := int(defaultInboxID)

	var req struct {
		Subject               string `json:"subject"`
		Content                string `json:"content"`
		Attachments            []int  `json:"attachments"`
		ShareWithOrganization  *bool  `json:"share_with_organization"`
	}
	if err := r.Decode(&req, "json"); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.errorParsing", "name", "{globals.terms.request}"), nil, envelope.InputError)
	}
	if req.Content == "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.required", "name", "content"), nil, envelope.InputError)
	}

	contactChannelID, err := app.user.EnsureContactChannel(contact.ID, inboxID, contact.Email.String)
	if err != nil {
		app.lo.Error("error ensuring contact channel for portal", "error", err)
		return sendErrorEnvelope(r, err)
	}

	orgID := 0
	mem, hasOrg, _ := app.organization.GetMembershipForContact(int64(contact.ID))
	if hasOrg {
		orgID = mem.OrganizationID
	}
	shareWithOrg := false
	if hasOrg {
		if req.ShareWithOrganization != nil {
			shareWithOrg = *req.ShareWithOrganization
		} else {
			shareWithOrg = mem.ShareTicketsByDefault
		}
	}
	setOrgID := 0
	if shareWithOrg && orgID > 0 {
		setOrgID = orgID
	}

	conversationID, conversationUUID, err := app.conversation.CreateConversation(
		contact.ID, contactChannelID, inboxID,
		"", time.Now(), req.Subject,
		true, setOrgID,
	)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	media := resolveMediaFromIDs(app, req.Attachments)
	if _, err := app.conversation.CreateContactMessage(media, contact.ID, conversationUUID, req.Content, cmodels.ContentTypeHTML); err != nil {
		_ = app.conversation.DeleteConversation(conversationUUID)
		return sendErrorEnvelope(r, err)
	}

	conv, _ := app.conversation.GetConversation(conversationID, "", "")
	return r.SendEnvelope(conv)
}

// resolveMediaFromIDs loads media by ids (for portal message/create). Returns empty slice on any error.
func resolveMediaFromIDs(app *App, ids []int) []medModels.Media {
	var out []medModels.Media
	for _, id := range ids {
		m, err := app.media.Get(id, "")
		if err != nil {
			continue
		}
		out = append(out, m)
	}
	return out
}

// handlePortalSendMessage adds a message to a conversation (portalAuth).
func handlePortalSendMessage(r *fastglue.Request) error {
	app := r.Context.(*App)
	contact, _ := r.RequestCtx.UserValue("contact").(models.User)
	uuid := r.RequestCtx.UserValue("uuid").(string)

	conv, err := app.conversation.GetConversation(0, uuid, "")
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	orgID, err := portalContactOrgID(app, contact.ID)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	if !contactCanAccessConversation(conv, contact.ID, orgID) {
		return r.SendErrorEnvelope(http.StatusNotFound, app.i18n.Ts("globals.messages.notFound", "name", "{globals.terms.conversation}"), nil, envelope.NotFoundError)
	}

	var req struct {
		Message     string `json:"message"`
		Attachments []int  `json:"attachments"`
	}
	if err := r.Decode(&req, "json"); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.errorParsing", "name", "{globals.terms.request}"), nil, envelope.InputError)
	}
	if req.Message == "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.required", "name", "message"), nil, envelope.InputError)
	}

	media := resolveMediaFromIDs(app, req.Attachments)
	msg, err := app.conversation.CreateContactMessage(media, contact.ID, uuid, req.Message, cmodels.ContentTypeHTML)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(msg)
}

// handlePortalCloseConversation closes a conversation with a required comment (portalAuth).
func handlePortalCloseConversation(r *fastglue.Request) error {
	app := r.Context.(*App)
	contact, _ := r.RequestCtx.UserValue("contact").(models.User)
	uuid := r.RequestCtx.UserValue("uuid").(string)

	conv, err := app.conversation.GetConversation(0, uuid, "")
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	orgID, err := portalContactOrgID(app, contact.ID)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	if !contactCanAccessConversation(conv, contact.ID, orgID) {
		return r.SendErrorEnvelope(http.StatusNotFound, app.i18n.Ts("globals.messages.notFound", "name", "{globals.terms.conversation}"), nil, envelope.NotFoundError)
	}

	var req struct {
		Comment string `json:"comment"`
	}
	if err := r.Decode(&req, "json"); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.errorParsing", "name", "{globals.terms.request}"), nil, envelope.InputError)
	}
	if req.Comment == "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.required", "name", "comment"), nil, envelope.InputError)
	}

	if _, err := app.conversation.CreateContactMessage(nil, contact.ID, uuid, req.Comment, cmodels.ContentTypeText); err != nil {
		return sendErrorEnvelope(r, err)
	}
	// Contact as "actor" for status change (for activity).
	if err := app.conversation.UpdateConversationStatus(uuid, smodels.DefaultStatusIDClosed, "", "", contact); err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(map[string]string{"ok": "true"})
}
