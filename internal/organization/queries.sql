-- name: get-organizations
SELECT id, created_at, updated_at, name, description FROM organizations ORDER BY updated_at DESC;

-- name: get-organization
SELECT id, created_at, updated_at, name, description FROM organizations WHERE id = $1;

-- name: insert-organization
INSERT INTO organizations (name, description) VALUES ($1, $2) RETURNING id, created_at, updated_at, name, description;

-- name: update-organization
UPDATE organizations SET name = $2, description = $3, updated_at = now() WHERE id = $1 RETURNING id, created_at, updated_at, name, description;

-- name: delete-organization
DELETE FROM organizations WHERE id = $1;

-- name: get-organization-members
SELECT om.id, om.created_at, om.updated_at, om.organization_id, om.contact_id, om.share_tickets_by_default,
  u.first_name AS contact_first_name, u.last_name AS contact_last_name, u.email AS contact_email
FROM organization_members om
JOIN users u ON u.id = om.contact_id
WHERE om.organization_id = $1 AND u.deleted_at IS NULL AND u.type = 'contact'
ORDER BY u.first_name, u.last_name;

-- name: get-membership-for-contact
SELECT organization_id, share_tickets_by_default FROM organization_members WHERE contact_id = $1 LIMIT 1;

-- name: get-memberships-for-contact
SELECT om.organization_id, o.name AS organization_name, om.share_tickets_by_default
FROM organization_members om
JOIN organizations o ON o.id = om.organization_id
WHERE om.contact_id = $1
ORDER BY o.name;

-- name: add-member
INSERT INTO organization_members (organization_id, contact_id, share_tickets_by_default)
VALUES ($1, $2, $3)
ON CONFLICT (organization_id, contact_id) DO UPDATE SET share_tickets_by_default = $3, updated_at = now()
RETURNING id, created_at, updated_at, organization_id, contact_id, share_tickets_by_default;

-- name: remove-member
DELETE FROM organization_members WHERE organization_id = $1 AND contact_id = $2;

-- name: update-member-share-tickets-by-default
UPDATE organization_members SET share_tickets_by_default = $3, updated_at = now()
WHERE organization_id = $1 AND contact_id = $2
RETURNING id, created_at, updated_at, organization_id, contact_id, share_tickets_by_default;

-- name: contact-in-organization
SELECT EXISTS(SELECT 1 FROM organization_members WHERE organization_id = $1 AND contact_id = $2);

-- name: get-organization-domains
SELECT id, created_at, organization_id, domain FROM organization_domains WHERE organization_id = $1 ORDER BY domain;

-- name: add-organization-domain
INSERT INTO organization_domains (organization_id, domain) VALUES ($1, $2)
ON CONFLICT (organization_id, domain) DO UPDATE SET organization_id = organization_domains.organization_id
RETURNING id, created_at, organization_id, domain;

-- name: remove-organization-domain
DELETE FROM organization_domains WHERE organization_id = $1 AND domain = $2;

-- name: find-organizations-by-email-domain
SELECT organization_id FROM organization_domains WHERE LOWER(domain) = LOWER($1);
