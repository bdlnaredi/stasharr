-- Stasharr: how many times each reel was watched, and comments on scenes.
ALTER TABLE `reels` ADD COLUMN `views` integer not null default 0;

-- One conversation per scene. Comments written while watching a reel note
-- the reel. user_id 0 is the owner account; author keeps the name shown.
CREATE TABLE `site_comments` (
  `id` integer not null primary key autoincrement,
  `scene_id` integer not null,
  `reel_id` integer,
  `user_id` integer not null default 0,
  `author` varchar(255) not null,
  `body` text not null,
  `created_at` datetime not null,
  foreign key(`scene_id`) references `scenes`(`id`) on delete CASCADE
);
CREATE INDEX `index_site_comments_on_scene` ON `site_comments` (`scene_id`, `created_at`);
