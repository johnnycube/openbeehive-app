-- `user` is a reserved word in Postgres; the quoted workaround in 0001/0010
-- kept history valid, but every new query had to remember the quotes — a bug
-- that stays invisible on SQLite and only surfaces on Postgres. `users` is
-- unreserved in both dialects. Valid in SQLite and Postgres alike; SQLite
-- rewrites dependent objects on RENAME (and no FK clause references the old
-- name anyway).
ALTER TABLE "user" RENAME TO users;
