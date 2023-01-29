-- disable the enforcement of foreign-keys constraints
PRAGMA foreign_keys = off;
-- create "new_users" table
CREATE TABLE `new_users` (`id` integer NOT NULL, PRIMARY KEY (`id`));
-- copy rows from old table "users" to new temporary table "new_users"
INSERT INTO `new_users` (`id`) SELECT `id` FROM `users`;
-- drop "users" table after copying rows
DROP TABLE `users`;
-- rename temporary table "new_users" to "users"
ALTER TABLE `new_users` RENAME TO `users`;
-- insert faulty data
INSERT INTO `friendships` (`user_id`, `friend_id`) VALUES (3,2);
-- enable back the enforcement of foreign-keys constraints
PRAGMA foreign_keys = on;
