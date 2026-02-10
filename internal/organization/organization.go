// Package organization handles organizations and their members (contacts).
package organization

import (
	"database/sql"
	"embed"
	"errors"
	"strings"

	"github.com/ghotso/libredesk/internal/dbutil"
	"github.com/ghotso/libredesk/internal/envelope"
	"github.com/ghotso/libredesk/internal/organization/models"
	"github.com/jmoiron/sqlx"
	"github.com/knadh/go-i18n"
	"github.com/volatiletech/null/v9"
	"github.com/zerodha/logf"
)

var (
	//go:embed queries.sql
	efs embed.FS
)

// Manager handles organization-related operations.
type Manager struct {
	lo   *logf.Logger
	i18n *i18n.I18n
	q    queries
}

// Opts contains options for initializing the Manager.
type Opts struct {
	DB   *sqlx.DB
	Lo   *logf.Logger
	I18n *i18n.I18n
}

type queries struct {
	GetOrganizations                  *sqlx.Stmt `query:"get-organizations"`
	GetOrganization                   *sqlx.Stmt `query:"get-organization"`
	InsertOrganization                *sqlx.Stmt `query:"insert-organization"`
	UpdateOrganization                *sqlx.Stmt `query:"update-organization"`
	DeleteOrganization                *sqlx.Stmt `query:"delete-organization"`
	GetOrganizationMembers            *sqlx.Stmt `query:"get-organization-members"`
	GetMembershipForContact           *sqlx.Stmt `query:"get-membership-for-contact"`
	GetMembershipsForContact          *sqlx.Stmt `query:"get-memberships-for-contact"`
	AddMember                         *sqlx.Stmt `query:"add-member"`
	RemoveMember                      *sqlx.Stmt `query:"remove-member"`
	UpdateMemberShareTicketsByDefault *sqlx.Stmt `query:"update-member-share-tickets-by-default"`
	ContactInOrganization             *sqlx.Stmt `query:"contact-in-organization"`
	GetOrganizationDomains            *sqlx.Stmt `query:"get-organization-domains"`
	AddOrganizationDomain             *sqlx.Stmt `query:"add-organization-domain"`
	RemoveOrganizationDomain          *sqlx.Stmt `query:"remove-organization-domain"`
	FindOrganizationsByEmailDomain    *sqlx.Stmt `query:"find-organizations-by-email-domain"`
}

// New creates and returns a new instance of the Manager.
func New(opts Opts) (*Manager, error) {
	var q queries
	if err := dbutil.ScanSQLFile("queries.sql", &q, opts.DB, efs); err != nil {
		return nil, err
	}
	return &Manager{
		q:    q,
		lo:   opts.Lo,
		i18n: opts.I18n,
	}, nil
}

// GetAll returns all organizations.
func (m *Manager) GetAll() ([]models.Organization, error) {
	var orgs []models.Organization
	if err := m.q.GetOrganizations.Select(&orgs); err != nil && !errors.Is(err, sql.ErrNoRows) {
		m.lo.Error("error fetching organizations", "error", err)
		return nil, envelope.NewError(envelope.GeneralError, m.i18n.Ts("globals.messages.errorFetching", "name", "{globals.terms.organization}"), nil)
	}
	return orgs, nil
}

// Get returns an organization by ID.
func (m *Manager) Get(id int) (models.Organization, error) {
	var org models.Organization
	if err := m.q.GetOrganization.Get(&org, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return org, envelope.NewError(envelope.NotFoundError, m.i18n.Ts("globals.messages.notFound", "name", "{globals.terms.organization}"), nil)
		}
		m.lo.Error("error fetching organization", "id", id, "error", err)
		return org, envelope.NewError(envelope.GeneralError, m.i18n.Ts("globals.messages.errorFetching", "name", "{globals.terms.organization}"), nil)
	}
	return org, nil
}

// Create creates a new organization.
func (m *Manager) Create(name, description string) (models.Organization, error) {
	var org models.Organization
	desc := null.StringFromPtr(nil)
	if description != "" {
		desc = null.StringFrom(description)
	}
	if err := m.q.InsertOrganization.Get(&org, name, desc); err != nil {
		m.lo.Error("error creating organization", "error", err)
		return org, envelope.NewError(envelope.GeneralError, m.i18n.Ts("globals.messages.errorCreating", "name", "{globals.terms.organization}"), nil)
	}
	return org, nil
}

// Update updates an organization.
func (m *Manager) Update(id int, name, description string) (models.Organization, error) {
	var org models.Organization
	desc := null.StringFromPtr(nil)
	if description != "" {
		desc = null.StringFrom(description)
	}
	if err := m.q.UpdateOrganization.Get(&org, id, name, desc); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return org, envelope.NewError(envelope.NotFoundError, m.i18n.Ts("globals.messages.notFound", "name", "{globals.terms.organization}"), nil)
		}
		m.lo.Error("error updating organization", "id", id, "error", err)
		return org, envelope.NewError(envelope.GeneralError, m.i18n.Ts("globals.messages.errorUpdating", "name", "{globals.terms.organization}"), nil)
	}
	return org, nil
}

// Delete deletes an organization.
func (m *Manager) Delete(id int) error {
	_, err := m.q.DeleteOrganization.Exec(id)
	if err != nil {
		m.lo.Error("error deleting organization", "id", id, "error", err)
		return envelope.NewError(envelope.GeneralError, m.i18n.Ts("globals.messages.errorDeleting", "name", "{globals.terms.organization}"), nil)
	}
	return nil
}

// GetMembers returns members of an organization.
func (m *Manager) GetMembers(organizationID int) ([]models.OrganizationMember, error) {
	var members []models.OrganizationMember
	if err := m.q.GetOrganizationMembers.Select(&members, organizationID); err != nil && !errors.Is(err, sql.ErrNoRows) {
		m.lo.Error("error fetching organization members", "organization_id", organizationID, "error", err)
		return nil, envelope.NewError(envelope.GeneralError, m.i18n.Ts("globals.messages.errorFetching", "name", "organization members"), nil)
	}
	return members, nil
}

// GetMembershipForContact returns the contact's organization membership if any.
func (m *Manager) GetMembershipForContact(contactID int64) (models.Membership, bool, error) {
	var mem models.Membership
	if err := m.q.GetMembershipForContact.Get(&mem, contactID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return mem, false, nil
		}
		m.lo.Error("error fetching membership for contact", "contact_id", contactID, "error", err)
		return mem, false, envelope.NewError(envelope.GeneralError, m.i18n.Ts("globals.messages.errorFetching", "name", "membership"), nil)
	}
	return mem, true, nil
}

// GetMembershipsForContact returns all organization memberships for a contact.
func (m *Manager) GetMembershipsForContact(contactID int64) ([]models.ContactOrganizationMembership, error) {
	var list []models.ContactOrganizationMembership
	if err := m.q.GetMembershipsForContact.Select(&list, contactID); err != nil && !errors.Is(err, sql.ErrNoRows) {
		m.lo.Error("error fetching memberships for contact", "contact_id", contactID, "error", err)
		return nil, envelope.NewError(envelope.GeneralError, m.i18n.Ts("globals.messages.errorFetching", "name", "organizations"), nil)
	}
	return list, nil
}

// AddMember adds a contact to an organization.
func (m *Manager) AddMember(organizationID int, contactID int64, shareTicketsByDefault bool) (models.OrganizationMember, error) {
	var member models.OrganizationMember
	if err := m.q.AddMember.Get(&member, organizationID, contactID, shareTicketsByDefault); err != nil {
		m.lo.Error("error adding organization member", "error", err)
		return member, envelope.NewError(envelope.GeneralError, m.i18n.Ts("globals.messages.errorCreating", "name", "organization member"), nil)
	}
	return member, nil
}

// RemoveMember removes a contact from an organization.
func (m *Manager) RemoveMember(organizationID int, contactID int64) error {
	_, err := m.q.RemoveMember.Exec(organizationID, contactID)
	if err != nil {
		m.lo.Error("error removing organization member", "error", err)
		return envelope.NewError(envelope.GeneralError, m.i18n.Ts("globals.messages.errorDeleting", "name", "organization member"), nil)
	}
	return nil
}

// UpdateMemberShareTicketsByDefault updates the share_tickets_by_default flag for a member.
func (m *Manager) UpdateMemberShareTicketsByDefault(organizationID int, contactID int64, shareTicketsByDefault bool) (models.OrganizationMember, error) {
	var member models.OrganizationMember
	if err := m.q.UpdateMemberShareTicketsByDefault.Get(&member, organizationID, contactID, shareTicketsByDefault); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return member, envelope.NewError(envelope.NotFoundError, m.i18n.Ts("globals.messages.notFound", "name", "organization member"), nil)
		}
		m.lo.Error("error updating member share_tickets_by_default", "error", err)
		return member, envelope.NewError(envelope.GeneralError, m.i18n.Ts("globals.messages.errorUpdating", "name", "organization member"), nil)
	}
	return member, nil
}

// ContactInOrganization returns whether the contact is a member of the organization.
func (m *Manager) ContactInOrganization(organizationID int, contactID int64) (bool, error) {
	var exists bool
	if err := m.q.ContactInOrganization.Get(&exists, organizationID, contactID); err != nil {
		return false, err
	}
	return exists, nil
}

// GetDomains returns all domains for an organization.
func (m *Manager) GetDomains(organizationID int) ([]models.OrganizationDomain, error) {
	var list []models.OrganizationDomain
	if err := m.q.GetOrganizationDomains.Select(&list, organizationID); err != nil && !errors.Is(err, sql.ErrNoRows) {
		m.lo.Error("error fetching organization domains", "organization_id", organizationID, "error", err)
		return nil, envelope.NewError(envelope.GeneralError, m.i18n.Ts("globals.messages.errorFetching", "name", "domains"), nil)
	}
	return list, nil
}

// AddDomain adds a domain to an organization (normalized to lowercase).
func (m *Manager) AddDomain(organizationID int, domain string) (models.OrganizationDomain, error) {
	domain = strings.TrimSpace(strings.ToLower(domain))
	if domain == "" {
		return models.OrganizationDomain{}, envelope.NewError(envelope.InputError, m.i18n.Ts("globals.messages.required", "name", "domain"), nil)
	}
	var d models.OrganizationDomain
	if err := m.q.AddOrganizationDomain.Get(&d, organizationID, domain); err != nil {
		m.lo.Error("error adding organization domain", "error", err)
		return d, envelope.NewError(envelope.GeneralError, m.i18n.Ts("globals.messages.errorCreating", "name", "domain"), nil)
	}
	return d, nil
}

// RemoveDomain removes a domain from an organization.
func (m *Manager) RemoveDomain(organizationID int, domain string) error {
	domain = strings.TrimSpace(strings.ToLower(domain))
	_, err := m.q.RemoveOrganizationDomain.Exec(organizationID, domain)
	if err != nil {
		m.lo.Error("error removing organization domain", "error", err)
		return envelope.NewError(envelope.GeneralError, m.i18n.Ts("globals.messages.errorDeleting", "name", "domain"), nil)
	}
	return nil
}

// OrganizationIDsByEmailDomain returns organization IDs that have the given email domain.
// emailDomain should be the part after @ (e.g. "example.com").
func (m *Manager) OrganizationIDsByEmailDomain(emailDomain string) ([]int, error) {
	emailDomain = strings.TrimSpace(strings.ToLower(emailDomain))
	if emailDomain == "" {
		return nil, nil
	}
	var rows []struct {
		OrganizationID int `db:"organization_id"`
	}
	if err := m.q.FindOrganizationsByEmailDomain.Select(&rows, emailDomain); err != nil && !errors.Is(err, sql.ErrNoRows) {
		m.lo.Error("error finding organizations by domain", "error", err)
		return nil, err
	}
	ids := make([]int, len(rows))
	for i := range rows {
		ids[i] = rows[i].OrganizationID
	}
	return ids, nil
}

// AddContactToOrganizationsByEmailDomain adds the contact to any organization that has a domain matching the contact's email.
// Used e.g. when processing incoming messages so new contacts are auto-assigned to orgs by email domain.
func (m *Manager) AddContactToOrganizationsByEmailDomain(contactID int64, email string) {
	email = strings.TrimSpace(strings.ToLower(email))
	if email == "" {
		return
	}
	at := strings.LastIndex(email, "@")
	if at < 0 || at == len(email)-1 {
		return
	}
	domain := email[at+1:]
	orgIDs, err := m.OrganizationIDsByEmailDomain(domain)
	if err != nil || len(orgIDs) == 0 {
		return
	}
	for _, orgID := range orgIDs {
		_, _ = m.AddMember(orgID, contactID, false)
	}
}
