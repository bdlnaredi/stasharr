-- Stasharr: markers suggested by the Stasharr Tagger, waiting for review,
-- and which scenes it has looked at.
CREATE TABLE `ai_marker_suggestions` (
  `id` integer not null primary key autoincrement,
  `scene_id` integer not null,
  `label` varchar(255) not null,
  `seconds` float not null,
  `end_seconds` float not null,
  `score` float not null,
  `created_at` datetime not null,
  foreign key(`scene_id`) references `scenes`(`id`) on delete CASCADE
);
CREATE INDEX `index_ai_marker_suggestions_on_scene` ON `ai_marker_suggestions` (`scene_id`, `seconds`);

CREATE TABLE `ai_marker_runs` (
  `scene_id` integer not null primary key,
  `ran_at` datetime not null,
  `found` integer not null default 0,
  foreign key(`scene_id`) references `scenes`(`id`) on delete CASCADE
);
