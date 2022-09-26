-- create "friendships" table
CREATE TABLE `friendships` (`user_id` integer NOT NULL, `friend_id` integer NOT NULL, PRIMARY KEY (`user_id`, `friend_id`), CONSTRAINT `user_id_fk` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE, CONSTRAINT `friend_id_fk` FOREIGN KEY (`friend_id`) REFERENCES `users` (`id`) ON DELETE CASCADE);

INSERT INTO `users` (`id`) VALUES (1), (2);
INSERT INTO `friendships` (`user_id`, `friend_id`) VALUES (1,2), (2,1);
