-- Modify "media" table
ALTER TABLE "media" ADD COLUMN "deleted_at" timestamptz NULL, ADD COLUMN "is_activated" boolean NOT NULL DEFAULT false;
