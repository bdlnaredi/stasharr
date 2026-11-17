-- Stasharr: invites, tier upgrade requests, two-step sign in, trusted
-- devices and shared playlists.

-- one row per account with two-step sign in on; user_id 0 is the owner
CREATE TABLE `user_two_step` (
  `user_id` integer not null primary key,
  `secret` varchar(64) not null,
  -- JSON array of SHA-256 hashes of the unused recovery codes
  `recovery` text not null default '[]',
  `created_at` datetime not null
);

-- devices that passed two-step sign in and are trusted for 90 days
CREATE TABLE `user_devices` (
  `id` integer not null primary key autoincrement,
  `user_id` integer not null,
  `token_hash` varchar(64) not null,
  `name` varchar(255) not null default '',
  `created_at` datetime not null,
  `last_used_at` datetime not null
);
CREATE UNIQUE INDEX `index_user_devices_on_token` ON `user_devices` (`token_hash`);
CREATE INDEX `index_user_devices_on_user` ON `user_devices` (`user_id`);

CREATE TABLE `user_invites` (
  `id` integer not null primary key autoincrement,
  `code` varchar(64) not null,
  `tier_id` integer,
  `role` varchar(16) not null default 'VIEWER',
  `max_uses` integer not null default 1,
  `uses` integer not null default 0,
  `note` varchar(255) not null default '',
  `expires_at` datetime,
  `created_at` datetime not null,
  foreign key(`tier_id`) references `user_tiers`(`id`) on delete SET NULL
);
CREATE UNIQUE INDEX `index_user_invites_on_code` ON `user_invites` (`code`);

CREATE TABLE `tier_requests` (
  `id` integer not null primary key autoincrement,
  `user_id` integer not null,
  `tier_id` integer not null,
  `message` text not null default '',
  -- PENDING, APPROVED or DECLINED
  `status` varchar(16) not null default 'PENDING',
  `created_at` datetime not null,
  `decided_at` datetime,
  foreign key(`user_id`) references `users`(`id`) on delete CASCADE,
  foreign key(`tier_id`) references `user_tiers`(`id`) on delete CASCADE
);
CREATE INDEX `index_tier_requests_on_status` ON `tier_requests` (`status`);

-- '' private, 'users' every signed in account, 'site' a Staff pick on the site
ALTER TABLE `playlists` ADD COLUMN `sharing` varchar(16) not null default '';
