-- Fork: multi-user support, per-user history/ratings and generated reels.

CREATE TABLE `user_tiers` (
  `id` integer not null primary key autoincrement,
  `name` varchar(255) not null,
  `description` text,
  -- JSON array of library folder paths; empty = all paths
  `allowed_paths` text not null default '[]',
  -- JSON arrays of tag ids
  `blocked_tag_ids` text not null default '[]',
  `required_tag_ids` text not null default '[]',
  `can_download` boolean not null default 1,
  `can_view_reels` boolean not null default 1,
  `created_at` datetime not null,
  `updated_at` datetime not null
);
CREATE UNIQUE INDEX `index_user_tiers_on_name` ON `user_tiers` (`name`);

CREATE TABLE `users` (
  `id` integer not null primary key autoincrement,
  `username` varchar(255) not null,
  `password_hash` varchar(255) not null,
  `role` varchar(16) not null default 'VIEWER',
  `tier_id` integer,
  `api_key` varchar(255),
  `disabled` boolean not null default 0,
  `last_login_at` datetime,
  `created_at` datetime not null,
  `updated_at` datetime not null,
  foreign key(`tier_id`) references `user_tiers`(`id`) on delete SET NULL
);
CREATE UNIQUE INDEX `index_users_on_username` ON `users` (`username` COLLATE NOCASE);
CREATE UNIQUE INDEX `index_users_on_api_key` ON `users` (`api_key`) WHERE `api_key` IS NOT NULL;

-- per-user rating and resume position; the owner keeps using the scenes table
CREATE TABLE `users_scenes` (
  `user_id` integer not null,
  `scene_id` integer not null,
  `rating` tinyint,
  `resume_time` float not null default 0,
  `play_duration` float not null default 0,
  `updated_at` datetime not null,
  primary key(`user_id`, `scene_id`),
  foreign key(`user_id`) references `users`(`id`) on delete CASCADE,
  foreign key(`scene_id`) references `scenes`(`id`) on delete CASCADE
);
CREATE INDEX `index_users_scenes_on_scene_id` ON `users_scenes` (`scene_id`);

-- play and O history become per user; NULL user_id = owner (existing rows)
ALTER TABLE `scenes_view_dates` ADD COLUMN `user_id` integer REFERENCES `users`(`id`) ON DELETE CASCADE;
ALTER TABLE `scenes_o_dates` ADD COLUMN `user_id` integer REFERENCES `users`(`id`) ON DELETE CASCADE;
CREATE INDEX `index_scenes_view_dates_user` ON `scenes_view_dates` (`user_id`, `scene_id`);
CREATE INDEX `index_scenes_o_dates_user` ON `scenes_o_dates` (`user_id`, `scene_id`);

-- generated short vertical reels
CREATE TABLE `reels` (
  `id` integer not null primary key autoincrement,
  `scene_id` integer not null,
  `path` text not null,
  `duration` float not null,
  `width` integer not null,
  `height` integer not null,
  -- how segments were chosen: MARKERS, HEATMAP or ANALYSIS
  `source` varchar(16) not null,
  -- JSON array of {start, end}
  `segments` text not null default '[]',
  `created_at` datetime not null,
  foreign key(`scene_id`) references `scenes`(`id`) on delete CASCADE
);
CREATE INDEX `index_reels_on_scene_id` ON `reels` (`scene_id`);
