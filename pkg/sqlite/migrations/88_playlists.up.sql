-- Stasharr: playlists and Watch later

CREATE TABLE `playlists` (
  `id` integer not null primary key autoincrement,
  -- 0 = the owner account; other users are removed with their account
  `user_id` integer not null default 0,
  `name` varchar(255) not null,
  `description` text not null default '',
  `watch_later` boolean not null default 0,
  `created_at` datetime not null,
  `updated_at` datetime not null
);
CREATE INDEX `index_playlists_on_user_id` ON `playlists` (`user_id`);
CREATE UNIQUE INDEX `index_playlists_watch_later` ON `playlists` (`user_id`) WHERE `watch_later` = 1;

CREATE TABLE `playlists_scenes` (
  `playlist_id` integer not null,
  `scene_id` integer not null,
  `position` integer not null,
  `added_at` datetime not null,
  primary key(`playlist_id`, `scene_id`),
  foreign key(`playlist_id`) references `playlists`(`id`) on delete CASCADE,
  foreign key(`scene_id`) references `scenes`(`id`) on delete CASCADE
);
CREATE INDEX `index_playlists_scenes_on_scene_id` ON `playlists_scenes` (`scene_id`);
