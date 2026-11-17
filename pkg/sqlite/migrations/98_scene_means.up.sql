-- Stasharr: what each scene looks like on average (the mean of its search
-- vectors), for "Recommended for you" by taste.
CREATE TABLE `scene_mean_vectors` (
  `scene_id` integer not null primary key,
  `model` varchar(255) not null,
  `scale` float not null,
  `vec` blob not null,
  foreign key(`scene_id`) references `scenes`(`id`) on delete CASCADE
);
