package models

import (
	"time"

	"github.com/volatiletech/null/v9"
)

// Organization represents an organization that groups contacts.
type Organization struct {
	ID          int         `db:"id" json:"id"`
	CreatedAt   time.Time   `db:"created_at" json:"created_at"`
	UpdatedAt   time.Time   `db:"updated_at" json:"updated_at"`
	Name        string      `db:"name" json:"name"`
	Description null.String `db:"description" json:"description"`
}

// OrganizationMember represents a contact's membership in an organization.
type OrganizationMember struct {
	ID                     int       `db:"id" json:"id"`
	CreatedAt              time.Time `db:"created_at" json:"created_at"`
	UpdatedAt              time.Time `db:"updated_at" json:"updated_at"`
	OrganizationID         int       `db:"organization_id" json:"organization_id"`
	ContactID              int64     `db:"contact_id" json:"contact_id"`
	ShareTicketsByDefault  bool      `db:"share_tickets_by_default" json:"share_tickets_by_default"`
	ContactFirstName       string    `db:"contact_first_name" json:"contact_first_name,omitempty"`
	ContactLastName        string    `db:"contact_last_name" json:"contact_last_name,omitempty"`
	ContactEmail           null.String `db:"contact_email" json:"contact_email,omitempty"`
}

// Membership holds organization membership info for a contact (for access and create logic).
type Membership struct {
	OrganizationID        int  `db:"organization_id" json:"organization_id"`
	ShareTicketsByDefault bool `db:"share_tickets_by_default" json:"share_tickets_by_default"`
}

// ContactOrganizationMembership is a contact's membership in an organization (for list by contact).
type ContactOrganizationMembership struct {
	OrganizationID        int  `db:"organization_id" json:"organization_id"`
	OrganizationName      string `db:"organization_name" json:"organization_name"`
	ShareTicketsByDefault bool  `db:"share_tickets_by_default" json:"share_tickets_by_default"`
}

// OrganizationDomain is a domain associated with an organization (for auto-adding contacts by email).
type OrganizationDomain struct {
	ID             int       `db:"id" json:"id"`
	CreatedAt      time.Time `db:"created_at" json:"created_at"`
	OrganizationID int       `db:"organization_id" json:"organization_id"`
	Domain         string    `db:"domain" json:"domain"`
}
