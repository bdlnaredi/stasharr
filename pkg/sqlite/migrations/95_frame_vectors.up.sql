-- Stasharr: search by description. One small vector per sampled frame,
-- from the Stasharr Tagger, stored as signed bytes with a scale.
CREATE TABLE `scene_frame_vectors` (
  `scene_id` integer not null,
  `seconds` float not null,
  `scale` float not null,
  `vec` blob not null,
  primary key(`scene_id`, `seconds`),
  foreign key(`scene_id`) references `scenes`(`id`) on delete CASCADE
);

-- which scenes are indexed, and with which model
CREATE TABLE `scene_vector_runs` (
  `scene_id` integer not null primary key,
  `model` varchar(255) not null,
  `frames` integer not null default 0,
  `ran_at` datetime not null,
  foreign key(`scene_id`) references `scenes`(`id`) on delete CASCADE
);
