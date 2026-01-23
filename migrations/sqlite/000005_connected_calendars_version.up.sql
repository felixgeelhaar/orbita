-- Add version column to connected_calendars for optimistic concurrency control
ALTER TABLE connected_calendars ADD COLUMN version INTEGER NOT NULL DEFAULT 0;
