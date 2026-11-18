-- Stasharr: Netflix-style profiles, PINs and watch time per day.

-- Each profile is its own user account, so its history, likes and My list
-- stay apart, and it belongs to a main account. parent_id 0 is the owner.
CREATE TABLE `user_profiles` (
  `user_id` integer not null primary key,
  `parent_id` integer not null,
  `name` varchar(64) not null,
  `color` varchar(16) not null default '',
  `created_at` datetime not null,
  foreign key(`user_id`) references `users`(`id`) on delete CASCADE
);
CREATE INDEX `index_user_profiles_on_parent` ON `user_profiles` (`parent_id`);

-- optional PINs for profiles and main accounts; account_id 0 is the owner
CREATE TABLE `profile_pins` (
  `account_id` integer not null primary key,
  `pin_hash` varchar(255) not null
);

-- seconds watched per account per day, for the monthly recap; user_id 0 is
-- the owner
CREATE TABLE `user_watch_days` (
  `user_id` integer not null,
  `day` varchar(10) not null,
  `seconds` float not null default 0,
  primary key(`user_id`, `day`)
);
