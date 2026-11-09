# Changelog

What changed in Stasharr, newest first. Server versions are the `v…` tags
of the Docker image `ghcr.io/bdlnaredi/stasharr`; Android versions are the
`android-v…` releases. Everything that is not listed here works as in
[Stash](https://github.com/stashapp/stash).

## Next release (not tagged yet)

### Reels on or off
- **Settings → Auto-import → Reels** has a switch for the whole server. Off
  hides reels everywhere (the menu, scene pages, the site's Reels page and
  home row, the Android app's Reels tab) and stops making new ones. Reels
  already made are kept, so switching back on brings them back.

### Menus fit on one line
- Admin menu bar: Audio, Playlists, Reels and Describe are under **More**
  on wide screens (phones and tablets still list them all), and on screens
  up to 1600 pixels wide the menu shows icons only, with the name when
  pointing at one.
- Site header: a tighter menu on laptops, and the ☰ menu below 1340 pixels,
  so it stays on one line in every language.
- Settings: the list of sections on the left scrolls on its own when it is
  taller than the window, so everything down to Troubleshooting mode can
  be reached.

### Android app
- **Comments in the app:** under every scene, and on reels through a new
  comment button that opens them over the reel while it keeps playing.
  Write, read and delete your own (admins can delete any).
### Statistics
- The **Statistics** page (chart icon) has a new part for admins: plays per
  day, most watched scenes, performers, tags and reels, who is watching
  (plays, last played, last signed in per account), recently played, scenes
  added per month, picture quality, and storage used by video files and by
  each generated folder. Pick 7 days, 30 days, 90 days or a year.
### New from your favourite pornstars
- **Alerts** when a pornstar you follow has a new scene: a bell with the
  number of new scenes in the site's header, and a list of them; opening it
  marks them seen. For the owner account, performers marked favourite in the
  library count too.
- **Android:** a phone notification for new scenes (tap to open the scene),
  checked about every 30 minutes; a badge on My list (site look) or You, and
  a "New from your favourites" list. Turn notifications off under You or on
  the alerts list.
### Search by description
- Type what you want to see, such as "blonde in red lingerie on a couch",
  and get the scenes and the moment in each that looks most like it, with
  a picture of that moment. On the site it appears as **Matching moments**
  on the search results page; in the admin interface under **Describe** in
  the menu. Clicking a result plays from that moment.
- Needs the Stasharr Tagger (update its image: it has two new functions).
  Settings → AI markers → Search by description shows progress and indexes
  the rest; new scenes are added after auto-import. The tagger looks at one
  frame every 10 seconds of each scene once.

### Tag clips
- **Every tag gets its own moving picture:** a 4-second silent loop cut from
  the tag's best scene (highest rated, most played), starting at a marker
  for the tag when there is one, so "Cowgirl" shows that moment. Without a
  marker it uses the scene's reel moment or its busiest part.
- Clips play on the site's category tiles and at the top of each category
  page, on tag cards in the admin interface, and in the Android app. They
  only load while on screen. A still picture from the clip shows for people
  who prefer less motion.
- **Each tag gets a different clip.** Tags without a marker no longer share
  the same scene and moment: scenes another tag already uses come last,
  and clips from the same scene start at least 20 seconds apart. With the
  Stasharr Tagger running, the clip starts at the frame that looks most
  like the tag (for example a close-up for a body tag, or the hair colour
  for Blonde).
- Viewers only see a clip when their tier lets them see the scene it came
  from. Tags with a picture you set keep it.
- **Settings → AI markers → Tag clips:** progress, Make clips and Remake
  all. Each tag's page has **Make new clip**. New tags get clips after
  auto-import.

## v2.8 (2026-11-02)

### AI markers and the Stasharr Tagger
- **AI markers.** Stasharr can look through scenes and suggest markers
  (Kissing, Blowjob, Cowgirl, Doggy Style…). It takes a small key frame
  every few seconds, asks the Stasharr Tagger what each frame shows, and
  turns confident stretches into markers with a tag of the same name.
- **Stasharr Tagger**, a new add-on container:
  `ghcr.io/bdlnaredi/stasharr-tagger:latest` for the CPU and `:cuda` for
  NVIDIA graphics cards. It uses an OpenCLIP model, downloaded on first
  start.
- **Settings → AI markers:** tagger address and status (shows the NVIDIA
  card in use), progress, frame spacing, confidence, shortest marker,
  look at new scenes automatically, and an editable list of labels and
  their descriptions.
- **Review list:** play the suggested moment, then keep or throw away each
  suggestion or a whole scene at once. Review can be turned off to save
  markers straight away.
- **Unraid templates** for the tagger on the CPU and on an NVIDIA card
  (`--runtime=nvidia`, `NVIDIA_VISIBLE_DEVICES`), and an optional tagger
  block in `docker-compose.yml` with the graphics card section ready to
  uncomment.

### Views and comments
- **Reel views** are counted (after two seconds of play, once per browser
  every 30 minutes) on the site, the Reels page and the Android app, and
  shown on reel cards and in the player.
- **Scene views:** how many times a scene was played by everyone, on scene
  pages and scene cards on the site.
- **Comments** on scenes: under the scene on the site, and in a comments
  panel on every reel (reel comments are marked "on a reel"). People can
  delete their own comments; admins can delete any.

### Reels
- Every reel has **its own picture**, taken from the reel itself, instead
  of the scene's wide screenshot. Older reels get one the first time they
  are shown.
- The reels player on the site and in the app plays reels in a **new random
  order on every visit**; a reel picked from a row still plays first.

### Site
- **Languages:** English, हिन्दी (Hindi), Français, Русский and Español,
  picked from the globe button in the header. The first visit follows the
  browser's language. Counts use each language's plural forms and dates
  follow the chosen language.
- **New login page:** library pictures or moving previews drift behind a
  frosted sign-in box in the top right corner. Choose moving previews,
  pictures or plain under Site → Login page, optionally only scenes with
  one tag.
- Fixed the model banner sitting against the right edge on phones.

### Android app (android-v1.0.28 to android-v1.0.34)
- **Storefront mode:** site users (or admins who switch on Site layout
  under You) get the website's storefront in the app: featured banner,
  just-added ticker, promo banner, the admin's home rows, Scenes with sort
  tabs and filters, Pornstars A–Z, pornstar pages, My list, likes,
  everyone's rating and the age check.
- **Previews play by themselves** once scrolling stops (switch under You;
  press and hold still works).
- Reels show the video as soon as it plays instead of a screenshot, start
  when swiped to, use the reel's own picture, play in random order, count
  views and show views and comments.

## v2.7 (2026-10-31)

### Site
- **Live search:** results appear while typing.
- The featured banner fits long titles, also on phones held sideways.
- **Likes, favourite pornstars and ratings** per user; the average of
  everyone's ratings on scene pages.
- **Richer pornstar pages:** cover picture, facts, most popular scenes,
  frequently seen with, all scenes.
- **Scene pages:** chapters from markers, Up next with a countdown, theatre
  mode, More of her and You may also like.
- **Filters** for category, length, quality and channel; sort by latest,
  trending, most viewed, top rated, release date, longest or random.
- **Home layout:** the admin arranges, renames and hides home rows, and can
  add rows for any tag.
- **Promo banner**, **age check**, **logo upload** and **accent colour**.
- **Reels** on the site: a home row and a swipe-through player.
- **Same look in the Android app** for site users.

## v2.6 (2026-10-31)
- Site: long titles in the featured banner stay readable.

## v2.5 (2026-10-31)

### Site front end
- A **storefront-style site** for viewers instead of the library interface:
  a big featured banner, a just-added ticker, top pornstars, trending and
  top rated scenes, categories and channels. Five styles to choose from
  (Velvet, Neon, Crimson, Blush, Onyx), for viewers or for viewers and
  editors. Admins keep the normal interface and can open the site from the
  menu.

### Server
- **Smart playlists** that fill themselves from a filter.
- **Notifications** to Discord, Telegram, ntfy, Gotify or any webhook for
  new scenes, scenes that could not be identified, failed tasks, backups
  and metadata refreshes.
- **Scheduled backups** with restore, and **automatic metadata refresh**
  from StashDB, ThePornDB and other stash-boxes.

### Android app
- **Chromecast:** cast a scene to the TV from where you are; MP4/H.264
  plays directly, everything else through the server.
- **Offline downloads:** original file or a 1080p, 720p or 480p copy.
- **New player** with gestures (hold for 2×, double-tap to skip, swipe to
  scrub, brightness and volume), speed, zoom, screen shape, quality, audio
  and subtitle tracks, chapters, lock, and a choice of decoder.
- Picture-in-picture keeps playing and has skip buttons.
- Previews on scene pages and the banner, and sort and filter options on
  every Browse tab.
- Release builds are signed with a private key.

## v1.0.4 (2026-10-25)
- **Native Android app:** a Plex-style home screen with a featured banner
  and shelves, browse and search, scene, performer, studio and tag pages,
  playlists, reels, and a full screen player with resume,
  picture-in-picture and play queues.

## v0.1.7 (2026-10-24)
- **Recommendations:** a Similar tab on scene pages and a per-user
  Recommended for you row on the home page.
- **Install as an app** on phones, with one-tap Reels and Playlists.
- **Audio files** are imported as scenes, with album art or waveform covers.

## v0.1.6 (2026-10-24)
- **Reels** are one continuous clip each (15 seconds by default) and every
  scene gets several (3 by default) from different high points. Markers
  favour the part after them. Reels per scene setting and Remake all.

## v0.1.3 (2026-10-21)
- **Trash bin**, **duplicate cleanup**, **playlists, Watch later and
  Continue watching**, and a **file renamer**.
- Auto-import creates missing studios, performers and tags, and has an
  Identify again button.

## v0.1.1 (2026-10-21)
- First release of the fork: **multi-user logins** with roles and
  **tiers**, per-user history, ratings and resume, **auto-import** (scan and
  identify new files with ThePornDB, StashDB and other stash-boxes) and
  automatic **reels**.
- Auto-import picks up files that were moved or copied into the library,
  shows when it last checked, and has a Check now button.
