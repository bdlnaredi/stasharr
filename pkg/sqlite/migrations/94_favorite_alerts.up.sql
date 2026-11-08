-- Stasharr: when each user last looked at their alerts for new scenes from
-- favourite pornstars. user_id 0 is the owner account.
CREATE TABLE `site_alerts_seen` (
  `user_id` integer not null primary key,
  `seen_at` datetime not null
);
