-- Installed packages table for tracking locally installed marketplace packages
CREATE TABLE IF NOT EXISTS installed_packages (
    id UUID PRIMARY KEY,
    package_id VARCHAR(255) NOT NULL,
    version VARCHAR(50) NOT NULL,
    type VARCHAR(50) NOT NULL CHECK (type IN ('orbit', 'engine')),
    install_path TEXT NOT NULL,
    checksum VARCHAR(128),
    installed_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    enabled BOOLEAN NOT NULL DEFAULT true,
    user_id UUID NOT NULL,

    CONSTRAINT unique_package_per_user UNIQUE (package_id, user_id)
);

-- Indexes for common queries
CREATE INDEX IF NOT EXISTS idx_installed_packages_user_id ON installed_packages(user_id);
CREATE INDEX IF NOT EXISTS idx_installed_packages_package_id ON installed_packages(package_id);
CREATE INDEX IF NOT EXISTS idx_installed_packages_type ON installed_packages(type);
CREATE INDEX IF NOT EXISTS idx_installed_packages_enabled ON installed_packages(enabled) WHERE enabled = true;

-- Publisher API tokens for marketplace authentication
CREATE TABLE IF NOT EXISTS marketplace_api_tokens (
    id UUID PRIMARY KEY,
    publisher_id UUID NOT NULL REFERENCES marketplace_publishers(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    token_hash VARCHAR(255) NOT NULL,
    scopes TEXT[] NOT NULL DEFAULT '{}',
    last_used_at TIMESTAMP WITH TIME ZONE,
    expires_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    revoked_at TIMESTAMP WITH TIME ZONE,

    CONSTRAINT unique_token_name_per_publisher UNIQUE (publisher_id, name)
);

CREATE INDEX IF NOT EXISTS idx_marketplace_api_tokens_publisher ON marketplace_api_tokens(publisher_id);
CREATE INDEX IF NOT EXISTS idx_marketplace_api_tokens_hash ON marketplace_api_tokens(token_hash);
