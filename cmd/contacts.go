package main

import (
	"encoding/json"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	amodels "github.com/ghotso/libredesk/internal/auth/models"
	"github.com/ghotso/libredesk/internal/envelope"
	notifier "github.com/ghotso/libredesk/internal/notification"
	"github.com/ghotso/libredesk/internal/stringutil"
	tmpl "github.com/ghotso/libredesk/internal/template"
	"github.com/ghotso/libredesk/internal/user/models"
	"github.com/valyala/fasthttp"
	"github.com/volatiletech/null/v9"
	"github.com/zerodha/fastglue"
)

type createContactNoteReq struct {
	Note string `json:"note"`
}

type createContactReq struct {
	Email                   string `json:"email"`
	FirstName               string `json:"first_name"`
	LastName                string `json:"last_name"`
	PhoneNumber             string `json:"phone_number"`
	PhoneNumberCountryCode  string `json:"phone_number_country_code"`
	AvatarURL               string `json:"avatar_url"`
	OrganizationID          *int   `json:"organization_id"`           // add to existing org
	CreateOrganizationName  string `json:"create_organization_name"` // create org and add contact
	ShareTicketsByDefault   bool   `json:"share_tickets_by_default"` // when adding to org
}

type blockContactReq struct {
	Enabled bool `json:"enabled"`
}

// handleGetContacts returns a list of contacts from the database.
func handleGetContacts(r *fastglue.Request) error {
	var (
		app     = r.Context.(*App)
		order   = string(r.RequestCtx.QueryArgs().Peek("order"))
		orderBy = string(r.RequestCtx.QueryArgs().Peek("order_by"))
		filters = string(r.RequestCtx.QueryArgs().Peek("filters"))
		total   = 0
	)
	page, pageSize := getPagination(r)
	contacts, err := app.user.GetContacts(page, pageSize, order, orderBy, filters)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	if len(contacts) > 0 {
		total = contacts[0].Total
	}
	return r.SendEnvelope(envelope.PageResults{
		Results:    contacts,
		Total:      total,
		PerPage:    pageSize,
		TotalPages: (total + pageSize - 1) / pageSize,
		Page:       page,
	})
}

// defaultInboxIDForContact returns the inbox ID to use for a new contact's channel (portal_default_inbox_id or first inbox).
func defaultInboxIDForContact(app *App) (int, error) {
	settingsJSON, err := app.setting.GetByPrefix("app")
	if err == nil {
		var settings map[string]interface{}
		if jsonErr := json.Unmarshal(settingsJSON, &settings); jsonErr == nil {
			if v, ok := settings["app.portal_default_inbox_id"]; ok {
				if id, ok := v.(float64); ok && id > 0 {
					return int(id), nil
				}
			}
		}
	}
	inboxes, err := app.inbox.GetAll()
	if err != nil || len(inboxes) == 0 {
		return 0, err
	}
	return inboxes[0].ID, nil
}

// handleCreateContact creates a new contact. Optionally adds to an existing organization or creates one.
func handleCreateContact(r *fastglue.Request) error {
	app := r.Context.(*App)
	var req createContactReq
	if err := r.Decode(&req, "json"); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.errorParsing", "name", "{globals.terms.request}"), nil, envelope.InputError)
	}
	req.Email = strings.TrimSpace(strings.ToLower(req.Email))
	if req.Email == "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.empty", "name", "email"), nil, envelope.InputError)
	}
	if !stringutil.ValidEmail(req.Email) {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.invalid", "name", "email"), nil, envelope.InputError)
	}
	if strings.TrimSpace(req.FirstName) == "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.empty", "name", "first_name"), nil, envelope.InputError)
	}
	existing, _ := app.user.GetContact(0, req.Email)
	if existing.ID > 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("contact.alreadyExistsWithEmail"), nil, envelope.InputError)
	}
	inboxID, err := defaultInboxIDForContact(app)
	if err != nil || inboxID <= 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.errorFetching", "name", "{globals.terms.inbox}"), nil, envelope.GeneralError)
	}
	contact := models.User{
		Email:     null.StringFrom(req.Email),
		FirstName: strings.TrimSpace(req.FirstName),
		LastName:  strings.TrimSpace(req.LastName),
		AvatarURL: null.NewString(strings.TrimSpace(req.AvatarURL), strings.TrimSpace(req.AvatarURL) != ""),
		PhoneNumber:            null.NewString(strings.TrimSpace(req.PhoneNumber), strings.TrimSpace(req.PhoneNumber) != ""),
		PhoneNumberCountryCode: null.NewString(strings.TrimSpace(req.PhoneNumberCountryCode), strings.TrimSpace(req.PhoneNumberCountryCode) != ""),
		InboxID:         inboxID,
		SourceChannelID: null.StringFrom(req.Email),
	}
	if err := app.user.CreateContact(&contact); err != nil {
		return sendErrorEnvelope(r, err)
	}
	// Optionally create organization and add contact, or add to existing org.
	orgIDToAdd := 0
	if req.CreateOrganizationName != "" {
		org, err := app.organization.Create(strings.TrimSpace(req.CreateOrganizationName), "")
		if err != nil {
			return sendErrorEnvelope(r, err)
		}
		orgIDToAdd = org.ID
	} else if req.OrganizationID != nil && *req.OrganizationID > 0 {
		orgIDToAdd = *req.OrganizationID
	}
	if orgIDToAdd > 0 {
		_, err := app.organization.AddMember(orgIDToAdd, int64(contact.ID), req.ShareTicketsByDefault)
		if err != nil {
			return sendErrorEnvelope(r, err)
		}
	}
	// Auto-add contact to organizations whose domain matches contact email.
	addContactToOrganizationsByEmailDomain(app, int64(contact.ID), req.Email)
	c, err := app.user.GetContact(contact.ID, "")
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(c)
}

// handleGetTags returns a contact from the database.
func handleGetContact(r *fastglue.Request) error {
	var (
		app   = r.Context.(*App)
		id, _ = strconv.Atoi(r.RequestCtx.UserValue("id").(string))
	)
	if id <= 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.invalid", "name", "`id`"), nil, envelope.InputError)
	}
	c, err := app.user.GetContact(id, "")
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(c)
}

// handleSendContactSetPasswordEmail sends a set-password email to the contact (portal login).
func handleSendContactSetPasswordEmail(r *fastglue.Request) error {
	app := r.Context.(*App)
	id, _ := strconv.Atoi(r.RequestCtx.UserValue("id").(string))
	if id <= 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.invalid", "name", "`id`"), nil, envelope.InputError)
	}
	contact, err := app.user.GetContact(id, "")
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	if !contact.Email.Valid || contact.Email.String == "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("contact.noEmailForSetPassword"), nil, envelope.InputError)
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
		app.lo.Error("error sending contact set password email", "error", err)
		return r.SendErrorEnvelope(http.StatusInternalServerError, app.i18n.T("globals.messages.errorSendingPasswordResetEmail"), nil, envelope.GeneralError)
	}
	return r.SendEnvelope(map[string]string{"ok": "true"})
}

// handleUpdateContact updates a contact in the database.
func handleUpdateContact(r *fastglue.Request) error {
	var (
		app   = r.Context.(*App)
		id, _ = strconv.Atoi(r.RequestCtx.UserValue("id").(string))
	)
	if id <= 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.invalid", "name", "`id`"), nil, envelope.InputError)
	}

	contact, err := app.user.GetContact(id, "")
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	form, err := r.RequestCtx.MultipartForm()
	if err != nil {
		app.lo.Error("error parsing form data", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, app.i18n.Ts("globals.messages.errorParsing", "name", "{globals.terms.request}"), nil, envelope.GeneralError)
	}

	// Parse form data
	firstName := ""
	if v, ok := form.Value["first_name"]; ok && len(v) > 0 {
		firstName = string(v[0])
	}
	lastName := ""
	if v, ok := form.Value["last_name"]; ok && len(v) > 0 {
		lastName = string(v[0])
	}
	email := ""
	if v, ok := form.Value["email"]; ok && len(v) > 0 {
		email = strings.TrimSpace(string(v[0]))
	}
	phoneNumber := ""
	if v, ok := form.Value["phone_number"]; ok && len(v) > 0 {
		phoneNumber = string(v[0])
	}
	phoneNumberCountryCode := ""
	if v, ok := form.Value["phone_number_country_code"]; ok && len(v) > 0 {
		phoneNumberCountryCode = string(v[0])
	}
	avatarURL := ""
	if v, ok := form.Value["avatar_url"]; ok && len(v) > 0 {
		avatarURL = string(v[0])
	}
	newPassword := ""
	if v, ok := form.Value["new_password"]; ok && len(v) > 0 {
		newPassword = string(v[0])
	}

	// Set nulls to empty strings.
	if avatarURL == "null" {
		avatarURL = ""
	}
	if phoneNumberCountryCode == "null" {
		phoneNumberCountryCode = ""
	}
	if phoneNumber == "null" {
		phoneNumber = ""
	}

	// Validate mandatory fields.
	if email == "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.empty", "name", "email"), nil, envelope.InputError)
	}
	if !stringutil.ValidEmail(email) {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.invalid", "name", "email"), nil, envelope.InputError)
	}
	if firstName == "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.empty", "name", "first_name"), nil, envelope.InputError)
	}

	// Another contact with same new email?
	existingContact, _ := app.user.GetContact(0, email)
	if existingContact.ID > 0 && existingContact.ID != id {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("contact.alreadyExistsWithEmail"), nil, envelope.InputError)
	}

	contactToUpdate := models.User{
		FirstName:              firstName,
		LastName:               lastName,
		Email:                  null.StringFrom(email),
		AvatarURL:              null.NewString(avatarURL, avatarURL != ""),
		PhoneNumber:            null.NewString(phoneNumber, phoneNumber != ""),
		PhoneNumberCountryCode: null.NewString(phoneNumberCountryCode, phoneNumberCountryCode != ""),
	}

	if err := app.user.UpdateContact(id, contactToUpdate); err != nil {
		return sendErrorEnvelope(r, err)
	}
	// Auto-add contact to organizations whose domain matches contact email.
	addContactToOrganizationsByEmailDomain(app, int64(id), email)

	// Set portal password for contact if provided.
	if newPassword != "" {
		if err := app.user.SetPasswordForUser(id, newPassword); err != nil {
			return sendErrorEnvelope(r, err)
		}
	}

	// Delete avatar?
	if avatarURL == "" && contact.AvatarURL.Valid {
		fileName := filepath.Base(contact.AvatarURL.String)
		app.media.Delete(fileName)
		contact.AvatarURL.Valid = false
		contact.AvatarURL.String = ""
	}

	// Upload avatar?
	files, ok := form.File["files"]
	if ok && len(files) > 0 {
		if err := uploadUserAvatar(r, contact, files); err != nil {
			return sendErrorEnvelope(r, err)
		}
	}

	// Refetch contact and return it
	contact, err = app.user.GetContact(id, "")
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(contact)
}

// handleGetContactOrganizations returns all organization memberships for a contact.
func handleGetContactOrganizations(r *fastglue.Request) error {
	var (
		app          = r.Context.(*App)
		contactID, _ = strconv.ParseInt(r.RequestCtx.UserValue("id").(string), 10, 64)
	)
	if contactID <= 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.invalid", "name", "`id`"), nil, envelope.InputError)
	}
	// Ensure the user can read this contact (same permission as GET contact).
	if _, err := app.user.GetContact(int(contactID), ""); err != nil {
		return sendErrorEnvelope(r, err)
	}
	memberships, err := app.organization.GetMembershipsForContact(contactID)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(memberships)
}

// handleGetContactNotes returns all notes for a contact.
func handleGetContactNotes(r *fastglue.Request) error {
	var (
		app          = r.Context.(*App)
		contactID, _ = strconv.Atoi(r.RequestCtx.UserValue("id").(string))
	)
	if contactID <= 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.invalid", "name", "`id`"), nil, envelope.InputError)
	}
	notes, err := app.user.GetNotes(contactID)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(notes)
}

// handleCreateContactNote creates a note for a contact.
func handleCreateContactNote(r *fastglue.Request) error {
	var (
		app          = r.Context.(*App)
		contactID, _ = strconv.Atoi(r.RequestCtx.UserValue("id").(string))
		auser        = r.RequestCtx.UserValue("user").(amodels.User)
		req          = createContactNoteReq{}
	)
	if err := r.Decode(&req, "json"); err != nil {
		return sendErrorEnvelope(r, envelope.NewError(envelope.InputError, app.i18n.Ts("globals.messages.errorParsing", "name", "{globals.terms.request}"), nil))
	}
	if len(req.Note) == 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.empty", "name", "note"), nil, envelope.InputError)
	}
	n, err := app.user.CreateNote(contactID, auser.ID, req.Note)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	n, err = app.user.GetNote(n.ID)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(n)
}

// handleDeleteContactNote deletes a note for a contact.
func handleDeleteContactNote(r *fastglue.Request) error {
	var (
		app          = r.Context.(*App)
		contactID, _ = strconv.Atoi(r.RequestCtx.UserValue("id").(string))
		noteID, _    = strconv.Atoi(r.RequestCtx.UserValue("note_id").(string))
		auser        = r.RequestCtx.UserValue("user").(amodels.User)
	)
	if contactID <= 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.invalid", "name", "`id`"), nil, envelope.InputError)
	}
	if noteID <= 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.invalid", "name", "`note_id`"), nil, envelope.InputError)
	}

	agent, err := app.user.GetAgent(auser.ID, "")
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	// Allow deletion of only own notes and not those created by others, but also allow `Admin` to delete any note.
	if !agent.HasAdminRole() {
		note, err := app.user.GetNote(noteID)
		if err != nil {
			return sendErrorEnvelope(r, err)
		}
		if note.UserID != auser.ID {
			return r.SendErrorEnvelope(fasthttp.StatusForbidden, app.i18n.Ts("globals.messages.canOnlyDeleteOwn", "name", "{globals.terms.note}"), nil, envelope.InputError)
		}
	}

	app.lo.Info("deleting contact note", "note_id", noteID, "contact_id", contactID, "actor_id", auser.ID)

	if err := app.user.DeleteNote(noteID, contactID); err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(true)
}

// addContactToOrganizationsByEmailDomain adds the contact to any organization that has a domain matching the contact's email.
func addContactToOrganizationsByEmailDomain(app *App, contactID int64, email string) {
	email = strings.TrimSpace(strings.ToLower(email))
	if email == "" {
		return
	}
	at := strings.LastIndex(email, "@")
	if at < 0 || at == len(email)-1 {
		return
	}
	domain := email[at+1:]
	orgIDs, err := app.organization.OrganizationIDsByEmailDomain(domain)
	if err != nil || len(orgIDs) == 0 {
		return
	}
	for _, orgID := range orgIDs {
		_, _ = app.organization.AddMember(orgID, contactID, false)
	}
}

// handleBlockContact blocks a contact.
func handleBlockContact(r *fastglue.Request) error {
	var (
		app          = r.Context.(*App)
		contactID, _ = strconv.Atoi(r.RequestCtx.UserValue("id").(string))
		auser        = r.RequestCtx.UserValue("user").(amodels.User)
		req          = blockContactReq{}
	)

	if contactID <= 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.invalid", "name", "`id`"), nil, envelope.InputError)
	}

	if err := r.Decode(&req, "json"); err != nil {
		return sendErrorEnvelope(r, envelope.NewError(envelope.InputError, app.i18n.Ts("globals.messages.errorParsing", "name", "{globals.terms.request}"), nil))
	}

	app.lo.Info("setting contact block status", "contact_id", contactID, "enabled", req.Enabled, "actor_id", auser.ID)

	if err := app.user.ToggleEnabled(contactID, models.UserTypeContact, req.Enabled); err != nil {
		return sendErrorEnvelope(r, err)
	}

	contact, err := app.user.GetContact(contactID, "")
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(contact)
}
