package main

import (
	"strconv"

	"github.com/ghotso/libredesk/internal/envelope"
	umodels "github.com/ghotso/libredesk/internal/user/models"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
)

type createOrganizationRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type updateOrganizationRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type addOrganizationMemberRequest struct {
	ContactID             int64 `json:"contact_id"`
	ShareTicketsByDefault bool  `json:"share_tickets_by_default"`
}

type updateOrganizationMemberRequest struct {
	ShareTicketsByDefault bool `json:"share_tickets_by_default"`
}

func handleGetOrganizations(r *fastglue.Request) error {
	app := r.Context.(*App)
	orgs, err := app.organization.GetAll()
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(orgs)
}

func handleGetOrganization(r *fastglue.Request) error {
	app := r.Context.(*App)
	id, _ := strconv.Atoi(r.RequestCtx.UserValue("id").(string))
	if id < 1 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.invalid", "name", "`id`"), nil, envelope.InputError)
	}
	org, err := app.organization.Get(id)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(org)
}

func handleCreateOrganization(r *fastglue.Request) error {
	app := r.Context.(*App)
	var req createOrganizationRequest
	if err := r.Decode(&req, "json"); err != nil {
		return sendErrorEnvelope(r, envelope.NewError(envelope.InputError, app.i18n.Ts("globals.messages.errorParsing", "name", "{globals.terms.request}"), nil))
	}
	if req.Name == "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.required", "name", "`name`"), nil, envelope.InputError)
	}
	org, err := app.organization.Create(req.Name, req.Description)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(org)
}

func handleUpdateOrganization(r *fastglue.Request) error {
	app := r.Context.(*App)
	id, _ := strconv.Atoi(r.RequestCtx.UserValue("id").(string))
	if id < 1 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.invalid", "name", "`id`"), nil, envelope.InputError)
	}
	var req updateOrganizationRequest
	if err := r.Decode(&req, "json"); err != nil {
		return sendErrorEnvelope(r, envelope.NewError(envelope.InputError, app.i18n.Ts("globals.messages.errorParsing", "name", "{globals.terms.request}"), nil))
	}
	org, err := app.organization.Update(id, req.Name, req.Description)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(org)
}

func handleDeleteOrganization(r *fastglue.Request) error {
	app := r.Context.(*App)
	id, err := strconv.Atoi(r.RequestCtx.UserValue("id").(string))
	if err != nil || id < 1 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.invalid", "name", "`id`"), nil, envelope.InputError)
	}
	if err := app.organization.Delete(id); err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(true)
}

func handleGetOrganizationMembers(r *fastglue.Request) error {
	app := r.Context.(*App)
	id, _ := strconv.Atoi(r.RequestCtx.UserValue("id").(string))
	if id < 1 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.invalid", "name", "`id`"), nil, envelope.InputError)
	}
	members, err := app.organization.GetMembers(id)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(members)
}

func handleAddOrganizationMember(r *fastglue.Request) error {
	app := r.Context.(*App)
	orgID, _ := strconv.Atoi(r.RequestCtx.UserValue("id").(string))
	if orgID < 1 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.invalid", "name", "`id`"), nil, envelope.InputError)
	}
	var req addOrganizationMemberRequest
	if err := r.Decode(&req, "json"); err != nil {
		return sendErrorEnvelope(r, envelope.NewError(envelope.InputError, app.i18n.Ts("globals.messages.errorParsing", "name", "{globals.terms.request}"), nil))
	}
	if req.ContactID <= 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.required", "name", "`contact_id`"), nil, envelope.InputError)
	}
	// Verify contact exists and is type contact.
	_, err := app.user.Get(int(req.ContactID), "", umodels.UserTypeContact)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	member, err := app.organization.AddMember(orgID, req.ContactID, req.ShareTicketsByDefault)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(member)
}

func handleRemoveOrganizationMember(r *fastglue.Request) error {
	app := r.Context.(*App)
	orgID, _ := strconv.Atoi(r.RequestCtx.UserValue("id").(string))
	contactID, _ := strconv.ParseInt(r.RequestCtx.UserValue("contact_id").(string), 10, 64)
	if orgID < 1 || contactID <= 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.invalid", "name", "`id` or `contact_id`"), nil, envelope.InputError)
	}
	if err := app.organization.RemoveMember(orgID, contactID); err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(true)
}

func handleUpdateOrganizationMember(r *fastglue.Request) error {
	app := r.Context.(*App)
	orgID, _ := strconv.Atoi(r.RequestCtx.UserValue("id").(string))
	contactID, _ := strconv.ParseInt(r.RequestCtx.UserValue("contact_id").(string), 10, 64)
	if orgID < 1 || contactID <= 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.invalid", "name", "`id` or `contact_id`"), nil, envelope.InputError)
	}
	var req updateOrganizationMemberRequest
	if err := r.Decode(&req, "json"); err != nil {
		return sendErrorEnvelope(r, envelope.NewError(envelope.InputError, app.i18n.Ts("globals.messages.errorParsing", "name", "{globals.terms.request}"), nil))
	}
	member, err := app.organization.UpdateMemberShareTicketsByDefault(orgID, contactID, req.ShareTicketsByDefault)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(member)
}

func handleGetOrganizationDomains(r *fastglue.Request) error {
	app := r.Context.(*App)
	id, _ := strconv.Atoi(r.RequestCtx.UserValue("id").(string))
	if id < 1 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.invalid", "name", "`id`"), nil, envelope.InputError)
	}
	domains, err := app.organization.GetDomains(id)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(domains)
}

type addOrganizationDomainRequest struct {
	Domain string `json:"domain"`
}

func handleAddOrganizationDomain(r *fastglue.Request) error {
	app := r.Context.(*App)
	id, _ := strconv.Atoi(r.RequestCtx.UserValue("id").(string))
	if id < 1 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.invalid", "name", "`id`"), nil, envelope.InputError)
	}
	var req addOrganizationDomainRequest
	if err := r.Decode(&req, "json"); err != nil {
		return sendErrorEnvelope(r, envelope.NewError(envelope.InputError, app.i18n.Ts("globals.messages.errorParsing", "name", "{globals.terms.request}"), nil))
	}
	d, err := app.organization.AddDomain(id, req.Domain)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(d)
}

func handleRemoveOrganizationDomain(r *fastglue.Request) error {
	app := r.Context.(*App)
	id, _ := strconv.Atoi(r.RequestCtx.UserValue("id").(string))
	if id < 1 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.invalid", "name", "`id`"), nil, envelope.InputError)
	}
	domain := string(r.RequestCtx.QueryArgs().Peek("domain"))
	if domain == "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.required", "name", "domain"), nil, envelope.InputError)
	}
	if err := app.organization.RemoveDomain(id, domain); err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(true)
}
