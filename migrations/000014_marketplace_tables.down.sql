-- Drop marketplace tables

DROP TRIGGER IF EXISTS trigger_update_publisher_package_count ON marketplace_packages;
DROP FUNCTION IF EXISTS update_publisher_package_count();

DROP TRIGGER IF EXISTS trigger_update_package_rating ON marketplace_ratings;
DROP FUNCTION IF EXISTS update_package_rating();

DROP TABLE IF EXISTS marketplace_ratings;
DROP TABLE IF EXISTS marketplace_versions;
DROP TABLE IF EXISTS marketplace_packages;
DROP TABLE IF EXISTS marketplace_publishers;
