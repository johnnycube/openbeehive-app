-- 0010_password_auth.sql
-- Email + password sign-up. Sign-up always creates role='user'; the instance
-- admin is set from BEEHIVE_ADMIN_EMAIL / BEEHIVE_ADMIN_PASSWORD on boot.
ALTER TABLE "user" ADD COLUMN password_hash TEXT NOT NULL DEFAULT '';
ALTER TABLE "user" ADD COLUMN role TEXT NOT NULL DEFAULT 'user';        -- admin | user
ALTER TABLE "user" ADD COLUMN email_verified BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE "user" ADD COLUMN verification_token TEXT NOT NULL DEFAULT '';
