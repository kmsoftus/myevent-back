ALTER TABLE users
    DROP COLUMN IF EXISTS marketing_opt_in,
    DROP COLUMN IF EXISTS accepted_terms,
    DROP COLUMN IF EXISTS contact_phone;
