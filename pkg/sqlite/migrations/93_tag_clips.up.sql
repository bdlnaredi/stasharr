-- Stasharr: a short looping clip for each tag, cut from its best scene.
-- The files live in the generated folder under tagclips/<tag id>.mp4/.jpg.
CREATE TABLE `tag_clips` (
  `tag_id` integer not null primary key,
  `scene_id` integer not null,
  `seconds` float not null,
  `created_at` datetime not null,
  foreign key(`tag_id`) references `tags`(`id`) on delete CASCADE,
  foreign key(`scene_id`) references `scenes`(`id`) on delete CASCADE
);
