-- Marketplace tables for the Orbita marketplace

-- Publishers table
CREATE TABLE IF NOT EXISTS marketplace_publishers (
    id UUID PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    slug VARCHAR(255) NOT NULL UNIQUE,
    email VARCHAR(255) NOT NULL,
    website VARCHAR(512),
    description TEXT,
    verified BOOLEAN NOT NULL DEFAULT FALSE,
    avatar_url VARCHAR(512),
    package_count INTEGER NOT NULL DEFAULT 0,
    total_downloads BIGINT NOT NULL DEFAULT 0,
    user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_marketplace_publishers_slug ON marketplace_publishers(slug);
CREATE INDEX idx_marketplace_publishers_user_id ON marketplace_publishers(user_id);
CREATE INDEX idx_marketplace_publishers_verified ON marketplace_publishers(verified);

-- Packages table (orbits and engines)
CREATE TABLE IF NOT EXISTS marketplace_packages (
    id UUID PRIMARY KEY,
    package_id VARCHAR(255) NOT NULL UNIQUE,
    type VARCHAR(50) NOT NULL CHECK (type IN ('orbit', 'engine')),
    name VARCHAR(255) NOT NULL,
    description TEXT,
    author VARCHAR(255),
    homepage VARCHAR(512),
    license VARCHAR(100),
    tags TEXT[] NOT NULL DEFAULT '{}',
    latest_version VARCHAR(50),
    downloads BIGINT NOT NULL DEFAULT 0,
    rating DECIMAL(3,2) NOT NULL DEFAULT 0.00,
    rating_count INTEGER NOT NULL DEFAULT 0,
    verified BOOLEAN NOT NULL DEFAULT FALSE,
    featured BOOLEAN NOT NULL DEFAULT FALSE,
    publisher_id UUID REFERENCES marketplace_publishers(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_marketplace_packages_package_id ON marketplace_packages(package_id);
CREATE INDEX idx_marketplace_packages_type ON marketplace_packages(type);
CREATE INDEX idx_marketplace_packages_publisher_id ON marketplace_packages(publisher_id);
CREATE INDEX idx_marketplace_packages_verified ON marketplace_packages(verified);
CREATE INDEX idx_marketplace_packages_featured ON marketplace_packages(featured);
CREATE INDEX idx_marketplace_packages_downloads ON marketplace_packages(downloads DESC);
CREATE INDEX idx_marketplace_packages_rating ON marketplace_packages(rating DESC);
CREATE INDEX idx_marketplace_packages_tags ON marketplace_packages USING GIN(tags);
CREATE INDEX idx_marketplace_packages_name_search ON marketplace_packages USING GIN(to_tsvector('english', name || ' ' || COALESCE(description, '')));

-- Versions table
CREATE TABLE IF NOT EXISTS marketplace_versions (
    id UUID PRIMARY KEY,
    package_id UUID NOT NULL REFERENCES marketplace_packages(id) ON DELETE CASCADE,
    version VARCHAR(50) NOT NULL,
    min_api_version VARCHAR(50),
    changelog TEXT,
    checksum VARCHAR(128),
    download_url VARCHAR(512),
    size BIGINT NOT NULL DEFAULT 0,
    downloads BIGINT NOT NULL DEFAULT 0,
    prerelease BOOLEAN NOT NULL DEFAULT FALSE,
    deprecated BOOLEAN NOT NULL DEFAULT FALSE,
    deprecation_message TEXT,
    published_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(package_id, version)
);

CREATE INDEX idx_marketplace_versions_package_id ON marketplace_versions(package_id);
CREATE INDEX idx_marketplace_versions_version ON marketplace_versions(version);
CREATE INDEX idx_marketplace_versions_published_at ON marketplace_versions(published_at DESC);
CREATE INDEX idx_marketplace_versions_prerelease ON marketplace_versions(prerelease);
CREATE INDEX idx_marketplace_versions_deprecated ON marketplace_versions(deprecated);

-- Package ratings table (for future rating system)
CREATE TABLE IF NOT EXISTS marketplace_ratings (
    id UUID PRIMARY KEY,
    package_id UUID NOT NULL REFERENCES marketplace_packages(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    rating INTEGER NOT NULL CHECK (rating >= 1 AND rating <= 5),
    review TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(package_id, user_id)
);

CREATE INDEX idx_marketplace_ratings_package_id ON marketplace_ratings(package_id);
CREATE INDEX idx_marketplace_ratings_user_id ON marketplace_ratings(user_id);

-- Function to update package rating when ratings change
CREATE OR REPLACE FUNCTION update_package_rating()
RETURNS TRIGGER AS $$
BEGIN
    UPDATE marketplace_packages
    SET
        rating = COALESCE((
            SELECT AVG(rating)::DECIMAL(3,2)
            FROM marketplace_ratings
            WHERE package_id = COALESCE(NEW.package_id, OLD.package_id)
        ), 0.00),
        rating_count = (
            SELECT COUNT(*)
            FROM marketplace_ratings
            WHERE package_id = COALESCE(NEW.package_id, OLD.package_id)
        ),
        updated_at = NOW()
    WHERE id = COALESCE(NEW.package_id, OLD.package_id);
    RETURN COALESCE(NEW, OLD);
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_update_package_rating
AFTER INSERT OR UPDATE OR DELETE ON marketplace_ratings
FOR EACH ROW EXECUTE FUNCTION update_package_rating();

-- Function to update publisher package count
CREATE OR REPLACE FUNCTION update_publisher_package_count()
RETURNS TRIGGER AS $$
BEGIN
    -- Update old publisher if changed
    IF TG_OP = 'UPDATE' AND OLD.publisher_id IS DISTINCT FROM NEW.publisher_id THEN
        IF OLD.publisher_id IS NOT NULL THEN
            UPDATE marketplace_publishers
            SET package_count = (
                SELECT COUNT(*) FROM marketplace_packages WHERE publisher_id = OLD.publisher_id
            )
            WHERE id = OLD.publisher_id;
        END IF;
    END IF;

    -- Update new/current publisher
    IF TG_OP IN ('INSERT', 'UPDATE') AND NEW.publisher_id IS NOT NULL THEN
        UPDATE marketplace_publishers
        SET package_count = (
            SELECT COUNT(*) FROM marketplace_packages WHERE publisher_id = NEW.publisher_id
        )
        WHERE id = NEW.publisher_id;
    END IF;

    -- Handle delete
    IF TG_OP = 'DELETE' AND OLD.publisher_id IS NOT NULL THEN
        UPDATE marketplace_publishers
        SET package_count = (
            SELECT COUNT(*) FROM marketplace_packages WHERE publisher_id = OLD.publisher_id
        )
        WHERE id = OLD.publisher_id;
    END IF;

    RETURN COALESCE(NEW, OLD);
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_update_publisher_package_count
AFTER INSERT OR UPDATE OR DELETE ON marketplace_packages
FOR EACH ROW EXECUTE FUNCTION update_publisher_package_count();
