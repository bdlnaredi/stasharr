-- Stasharr: smart playlists. A playlist with a rule fills itself from a
-- saved scene filter instead of a list of scenes.
ALTER TABLE `playlists` ADD COLUMN `smart_rule` text not null default '';
