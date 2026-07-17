-- 0010_password_auth.sql
-- Email + password onboarding. The first account created on a fresh instance
-- becomes the admin (role='admin'); later accounts are regular users.
ALTER TABLE user ADD COLUMN password_hash TEXT NOT NULL DEFAULT '';
ALTER TABLE user ADD COLUMN role TEXT NOT NULL DEFAULT 'user';        -- admin | user
ALTER TABLE user ADD COLUMN email_verified BOOLEAN NOT NULL DEFAULT 0;
ALTER TABLE user ADD COLUMN verification_token TEXT NOT NULL DEFAULT '';
