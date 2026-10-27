BEGIN;

CREATE TABLE users (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    email text NOT NULL UNIQUE CHECK (email = lower(email) AND email = btrim(email)),
    password_hash text NOT NULL,
    first_name text NOT NULL,
    last_name text NOT NULL,
    phone text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    is_active boolean NOT NULL DEFAULT true
);

CREATE FUNCTION set_updated_at() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    NEW.updated_at = now();
    RETURN NEW;
END;
$$;
CREATE TRIGGER users_updated_at BEFORE UPDATE ON users
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TABLE user_roles (
    user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role text NOT NULL CHECK (role IN ('client', 'driver', 'admin')),
    PRIMARY KEY (user_id, role)
);

-- Every account, including a bootstrapped admin or approved driver, is a client.
CREATE FUNCTION grant_client_role() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    INSERT INTO user_roles (user_id, role) VALUES (NEW.id, 'client');
    RETURN NEW;
END;
$$;
CREATE TRIGGER users_client_role AFTER INSERT ON users
    FOR EACH ROW EXECUTE FUNCTION grant_client_role();

CREATE TABLE refresh_tokens (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    family_id uuid NOT NULL,
    token_hash text NOT NULL UNIQUE CHECK (token_hash ~ '^[0-9a-f]{64}$'),
    expires_at timestamptz NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    revoked_at timestamptz,
    replaced_by uuid REFERENCES refresh_tokens(id) ON DELETE SET NULL,
    user_agent text NOT NULL DEFAULT '',
    ip inet,
    CHECK (expires_at > created_at)
);
CREATE INDEX refresh_tokens_user_idx ON refresh_tokens (user_id);
CREATE INDEX refresh_tokens_family_idx ON refresh_tokens (family_id);
CREATE INDEX refresh_tokens_expiry_idx ON refresh_tokens (expires_at);

CREATE TABLE driver_applications (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    contact_phone text NOT NULL,
    first_name text NOT NULL,
    last_name text NOT NULL,
    pin_code_encrypted bytea NOT NULL,
    car_make text NOT NULL,
    car_model text NOT NULL,
    car_year smallint NOT NULL CHECK (car_year >= 1886),
    car_plate text NOT NULL,
    car_color text NOT NULL,
    seats smallint NOT NULL CHECK (seats > 0),
    status text NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'approved', 'rejected')),
    reviewed_by uuid REFERENCES users(id) ON DELETE SET NULL,
    reviewed_at timestamptz,
    reject_reason text,
    created_at timestamptz NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX driver_applications_one_pending_per_user
    ON driver_applications (user_id) WHERE status = 'pending';
CREATE INDEX driver_applications_user_created_idx ON driver_applications (user_id, created_at DESC);
CREATE INDEX driver_applications_status_created_idx ON driver_applications (status, created_at);

CREATE TABLE districts (
    id smallserial PRIMARY KEY,
    name_az text NOT NULL UNIQUE,
    slug text NOT NULL UNIQUE,
    search_key text NOT NULL CHECK (search_key ~ '^[a-z0-9 -]+$'),
    requires_permit boolean NOT NULL DEFAULT false,
    is_active boolean NOT NULL DEFAULT true,
    sort_order integer NOT NULL DEFAULT 0
);
CREATE INDEX districts_active_order_idx ON districts (sort_order, name_az) WHERE is_active;

CREATE TABLE rides (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    driver_id uuid NOT NULL REFERENCES users(id),
    from_district_id smallint NOT NULL REFERENCES districts(id),
    to_district_id smallint NOT NULL REFERENCES districts(id),
    departure_at timestamptz NOT NULL,
    seats_total smallint NOT NULL CHECK (seats_total > 0),
    seats_available smallint NOT NULL,
    price_azn numeric(8,2) NOT NULL CHECK (price_azn >= 0),
    notes text NOT NULL DEFAULT '',
    status text NOT NULL DEFAULT 'published' CHECK (status IN ('published', 'in_progress', 'completed', 'cancelled')),
    created_at timestamptz NOT NULL DEFAULT now(),
    CHECK (from_district_id <> to_district_id),
    CHECK (seats_available BETWEEN 0 AND seats_total)
);
CREATE UNIQUE INDEX rides_one_active_per_driver
    ON rides (driver_id) WHERE status IN ('published', 'in_progress');
CREATE INDEX rides_search_idx ON rides (from_district_id, to_district_id, departure_at)
    WHERE status = 'published' AND seats_available > 0;

COMMIT;
