-- Stasharr site: each user's liked scenes and favourite models.
-- user_id 0 is the owner account.
CREATE TABLE `site_favorites` (
  `user_id` integer not null default 0,
  `kind` varchar(16) not null,
  `target_id` integer not null,
  `created_at` datetime not null,
  primary key(`user_id`, `kind`, `target_id`)
);
CREATE INDEX `index_site_favorites_on_target` ON `site_favorites` (`kind`, `target_id`);
