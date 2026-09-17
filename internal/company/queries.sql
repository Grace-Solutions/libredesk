-- name: get-companies
SELECT
    COUNT(*) OVER() AS total,
    c.id,
    c.created_at,
    c.updated_at,
    c.name,
    c.description,
    c.website,
    c.phone_number_country_code,
    c.phone_number,
    c.country,
    c.parent_id,
    p.name AS parent_name,
    (SELECT COUNT(*) FROM users u WHERE u.company_id = c.id AND u.deleted_at IS NULL) AS contact_count,
    (SELECT COUNT(*) FROM companies ch WHERE ch.parent_id = c.id) AS child_count
FROM companies c
LEFT JOIN companies p ON p.id = c.parent_id
WHERE ($1 = '' OR c.name ILIKE '%' || $1 || '%')
    -- $2 = 0 lists every company, > 0 lists the direct children of that company.
    AND ($2 = 0 OR c.parent_id = $2)
ORDER BY c.name, c.id
LIMIT NULLIF($3, 0) OFFSET $4;

-- name: get-companies-compact
SELECT id, name FROM companies
WHERE ($1 = '' OR name ILIKE '%' || $1 || '%')
ORDER BY name, id
LIMIT NULLIF($2, 0) OFFSET $3;

-- name: get-companies-compact-by-ids
SELECT id, name FROM companies WHERE id = ANY($1) ORDER BY name, id;

-- name: get-company
SELECT
    c.id,
    c.created_at,
    c.updated_at,
    c.name,
    c.description,
    c.website,
    c.phone_number_country_code,
    c.phone_number,
    c.country,
    c.parent_id,
    p.name AS parent_name,
    (SELECT COUNT(*) FROM users u WHERE u.company_id = c.id AND u.deleted_at IS NULL) AS contact_count,
    (SELECT COUNT(*) FROM companies ch WHERE ch.parent_id = c.id) AS child_count
FROM companies c
LEFT JOIN companies p ON p.id = c.parent_id
WHERE c.id = $1;

-- name: insert-company
INSERT INTO companies (name, description, website, phone_number_country_code, phone_number, country, parent_id)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING id;

-- name: update-company
UPDATE companies
SET name = $2,
    description = $3,
    website = $4,
    phone_number_country_code = $5,
    phone_number = $6,
    country = $7,
    parent_id = $8,
    updated_at = now()
WHERE id = $1;

-- name: delete-company
DELETE FROM companies WHERE id = $1;

-- name: is-company-descendant
-- True when $2 sits anywhere below $1 in the tree. Guards against making a company
-- the child of its own sub-company, which would detach the cycle from every root.
WITH RECURSIVE descendants AS (
    SELECT id FROM companies WHERE parent_id = $1
    UNION
    SELECT c.id FROM companies c JOIN descendants d ON c.parent_id = d.id
)
SELECT EXISTS (SELECT 1 FROM descendants WHERE id = $2);
