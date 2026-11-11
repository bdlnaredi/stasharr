-- Stasharr: AI markers learn from reviews. A suggestion keeps what its
-- frames looked like (a small vector); when an admin keeps or throws it
-- away, that becomes an example for its label.
ALTER TABLE `ai_marker_suggestions` ADD COLUMN `model` varchar(255);
ALTER TABLE `ai_marker_suggestions` ADD COLUMN `scale` float;
ALTER TABLE `ai_marker_suggestions` ADD COLUMN `vec` blob;

CREATE TABLE `ai_marker_examples` (
  `id` integer not null primary key autoincrement,
  `label` varchar(255) not null,
  `kept` boolean not null,
  `model` varchar(255) not null,
  `scale` float not null,
  `vec` blob not null,
  `scene_id` integer,
  `created_at` datetime not null
);
CREATE INDEX `index_ai_marker_examples_on_label` ON `ai_marker_examples` (`model`, `label`);
