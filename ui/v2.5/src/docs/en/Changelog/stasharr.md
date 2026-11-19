What changed in Stasharr, newest first. Server versions are the `v…` tags
of the Docker image `ghcr.io/bdlnaredi/stasharr`; Android versions are the
`android-v…` releases. Everything that is not listed here works as in
[Stash](https://github.com/stashapp/stash).

## Next release (not tagged yet)

### Layouts in the Android app
- The storefront in the app now follows the site's layout too: Bento tiles,
  the Mosaic wall, Spotlight's big scene with a numbered top 10, Showcase's
  3D cover-flow, Immersive's full-screen scene, Editorial's magazine cover
  and "In this issue", Brutal's blocks and marquee, and Swipe's deck (drag
  right to like, left to skip).
- Scene lists, cards, headings and portraits change with it: masonry in
  Mosaic, numbered lists in Spotlight, serif and black-and-white in
  Editorial, outlined boxes that press in for Brutal, and more.
- **You → Layout** picks a layout for this phone only; "Site default" follows
  the site.

### Profiles, mini-player, discreet mode and monthly recap
- **Profiles**: up to 5 Netflix-style profiles under one account, each with
  its own history, likes, My list and recap. "Who's watching?" shows once per
  browser session; switch any time from the account menu. Profiles can have
  a PIN, and so can the main profile (do set one, or anyone on a profile can
  switch to it). Only the main profile adds, renames or deletes profiles.
  Profiles get the main account's tier and are never more than viewers.
- **Mini-player**: leave a scene while it plays and it keeps going in a small
  window you can drag to any corner; ⤢ goes back to the scene at the same
  moment. It can be turned off on the Account page.
- **Discreet mode** (Account page, this browser only): press Esc twice (or `)
  to jump to a harmless page you choose, with the site's tab left out of the
  back button; a plain tab title and icon; pictures blurred until pointed
  at; and a PIN to get back in after hiding or being away.
- **Monthly recap** (account menu): time watched, plays, days and streaks,
  your top pornstars, categories and channels, the most replayed scene, and
  when you watch. Watch time is counted from this update on.
- Viewers can now turn on two-step sign in and ask for another tier
  themselves; before, the site refused them.

### Site layouts
- The site can now look completely different, not just in colour. Pick a
  layout in **Settings → Site → Layout**; every layout works with every style.
  - **Classic**: the original. Menu across the top, big banner, rows.
  - **Bento**: a floating glass menu bar and a home page of mixed-size tiles
    (featured scene, trending #1, top rated, star of the week, categories,
    just added).
  - **Mosaic**: a wide search bar, sticky category chips and a masonry wall
    of scene and model cards that play when you point at them.
  - **Spotlight**: a split screen. The chosen scene plays big on the left
    while a numbered top-10 list (Trending, New, Top rated, Most viewed,
    Picked for you) scrolls on the right.
  - **Showcase**: a 3D cover-flow of the featured scenes over a soft glow of
    the picture; swipe or use the arrows.
  - **Immersive**: the featured scene fills the screen, the rows sit on a
    frosted sheet over it, and the menu is a floating dock at the bottom.
- The layouts change every page, not only the home page:
  - **Bento**: rounded cards with the words inside, model names on the
    photo, a wide tile in every row of categories, and the scene page's
    details, cast and tags in tiles.
  - **Mosaic**: scenes, models, categories and channels all become masonry
    walls that play when pointed at; the sort buttons stick to the top as chips.
  - **Spotlight**: scenes become a numbered list whose pictures play when
    pointed at; models and categories become ranked lists; the scene page
    keeps "You may also like" in a list beside it, and a model's photo stays
    on the left while her scenes scroll.
  - **Showcase**: cards lean towards the pointer with a shine, models get
    round photos with a glowing ring, and the scene page glows with the
    scene's colours like ambient light.
  - **Immersive**: frosted glass everywhere, a play button on the card you
    point at, wide cinematic category cards, and the scene's picture blurred
    behind the whole scene page.
- Three more layouts:
  - **Editorial**: a glossy magazine. A masthead with the menu under it, a
    cover story, "In this issue", features, serif headlines, categories as an
    index of names, articles with a drop capital, and black-and-white
    portraits that turn colour when pointed at.
  - **Brutal**: bold and loud. Thick borders, hard shadows, giant capitals,
    stickers and a scrolling marquee; cards jump when pointed at and press in
    when clicked.
  - **Swipe**: a deck of scenes on the home page. Drag right (or ♥) to like,
    left (or ✕) to skip, or use the arrow keys. Soft rounded cards everywhere
    and a floating Like / My list bar on every scene.
- **Preview** opens the site in a layout without changing it for anyone.
- Viewers can pick their own layout from their account menu on the site;
  **Site default** goes back to yours.

### Fixed
- Signing in failed after updating to v3.5 until the database was migrated,
  and the login page said the password was wrong. Sign in now works before
  the migration, so you can sign in and press Migrate.
- The login page shows the real problem when the server has one, instead of
  "wrong username or password".

## v3.5 (2026-11-17)

### Smoother streaming
- A new **Auto** stream for every scene lists all its sizes (240p up to the
  original), with segments that line up, so the player moves to a smaller
  size when the connection slows down and back up when it recovers,
  instead of stalling.
- **Android app:** Player → Playback → "Start scenes with": Original, **Auto
  on mobile data** (the new default: the original file on Wi-Fi, Auto on
  mobile data) or Always auto. Auto is also in the quality list. Auto needs
  the server to convert, which takes some processor.

### Recommendations by taste
- "Recommended for you" now also learns what each person likes to look at,
  not just which performers, tags and studios: scenes that look like the
  ones they played, rated or liked rank higher. It uses the search index
  made by the Stasharr Tagger (Settings → AI markers → Search by
  description), so it gets better as more scenes are indexed.

### Health page
- **Settings → Health:** free space on every disk Stasharr uses (library,
  generated files, database, backups), the last backup and Google Drive
  copy, auto-import, the tagger, notifications and tasks that failed, with
  a list of problems at the top.
- A notification (new event: Low disk space and other problems) when a disk
  has less free space than you set (20 GB by default) or under 5%, at most
  every 12 hours.

### Cleanup suggestions
- **Settings → Cleanup suggestions:** scenes not watched for 3, 6, 12 or 24
  months, never watched, or rated two stars or less, biggest first, with the
  total space they take. Pick scenes and remove them; with the trash bin on
  they go there first and can be restored.

### Invites and tier upgrades
- **Settings → Users → Invites:** make a link that lets someone create their
  own account, in the tier and role (viewer or editor) you choose, for a
  number of accounts and days. The link opens a sign-up form on the login
  page, and the new account is signed in straight away. You get a
  notification for each new account (new event: New accounts and upgrade
  requests).
- **Upgrade requests:** people pick another tier on their Account page
  (site and admin interface) with an optional note; you approve or decline
  under Settings → Users, and approving moves them to that tier.

### Safer sign in
- **Wrong passwords are limited:** after 5 wrong passwords or codes for an
  account from one address (or 20 from one address) in 15 minutes, sign in
  from there is refused for 15 minutes.
- **Two-step sign in:** on the Account page, scan a QR code with an
  authenticator app (Google Authenticator, Microsoft Authenticator, Aegis…)
  and sign in with its 6-digit code after the password. You get 8 one-time
  recovery codes. A device signed in with a code is remembered for 90 days;
  "Forget devices" signs them out of that. Admins can turn it off for
  someone who lost their phone (Settings → Users → Reset 2-step). Works on
  the login page and in the Android app.

### Shared playlists and Staff picks
- Every playlist can be **Only me**, **Everyone with an account**, or (for
  admins) a **Staff pick on the site**. Shared playlists show under
  "Shared by others" in Playlists, on the site's My list page and in the
  Android app; people only see the scenes their tier allows.
- **Staff picks** is a new home row for the site (on by default): one rail
  per playlist you put on the site, each with its own page.

## v3.4 (2026-11-12)

### Reels picked by AI
- With the Stasharr Tagger running, it picks each reel's moments: it looks
  at a frame every few seconds and scores how much is happening (your AI
  marker labels) and how good the picture is. Talking, title cards, dark or
  blurry shots and empty rooms score low, and each reel of a scene shows a
  different act where the scene has them. Loudness, motion and markers
  still help to choose between equally good moments.
- When AI markers look at a scene, its reels are remade with these moments,
  so reels improve as the tagger works through the library. **Remake all**
  under Auto-import → Reels redoes every reel now.
- Switch: **Let the AI tagger pick the moments** (Auto-import → Reels, and
  under Settings → AI markers → Generated features). Reels picked this way
  say "picked by AI".

### "New" means new
- The **New** badge on the site and in the app is only on scenes released
  in the last week (or, without a release date, added in the last week).
  Older videos that were just imported no longer show as new.
- The red Trans, Lesbian and Bi labels also show under scene titles on
  cards, on the site and in the app.

### More accurate AI markers
- **It learns from your reviews.** Every suggestion you keep or throw away
  is remembered for its label, and frames that look like ones you threw
  away score lower next time (higher for ones you kept). Throwing away
  wrong cumshots makes them rarer. The label list shows how many it learned
  from, and **Forget what it learned** starts over. Suggestions made before
  this update also teach it when the scene is in the search index.
- **Several descriptions per label**, split by `|`, are averaged, which
  describes a label far better than one sentence.
- **When** limits a label to the start or the end of a scene, and **Sure**
  asks for more confidence for one label.
- **Cumshot** now uses four descriptions, is only looked for in the last
  30% of a scene and needs 50% confidence. New helper labels (face and body
  close-ups, which were mistaken for cumshots most) soak up those frames.
  Saved label lists get the new wording unless the wording was changed.
- **Look at every scene again** re-checks the whole library with the new
  labels; suggestions waiting for review are replaced, markers already
  kept stay.
- Needs the updated tagger image (`ghcr.io/bdlnaredi/stasharr-tagger`);
  an older one still works without learning.

### Switches for generated features
- **Settings → AI markers → Generated features:** turn Reels, AI markers,
  Tag clips, Search by description and the scene ribbons on or off for the
  whole server. Off stops making new ones and hides them on the site, in
  the admin interface and in the Android app; what was made is kept.

### Trans, lesbian and bisexual ribbons
- Scene pictures get a deep red ribbon with gold edges and white lettering
  across the corner:
  **Trans** when a performer is trans or the scene has a tag like Trans,
  Transsexual, TS or Shemale; **Lesbian** for a Lesbian tag, or two or more
  women and no men (unless a tag says a man is there, such as Blowjob or
  FFM); **Bi** for a Bisexual tag. On the site, in the admin interface and
  in the Android app, with a matching label next to the title on scene
  pages.

### Android app: TV, auto-download, widgets and admin tools
- **Android TV:** the same app installs on Android TV and shows in the TV's
  app row. The sections are down the left side, whatever the remote is on
  gets a pink outline, and in the player OK pauses, left and right skip 10
  seconds (fast-forward and rewind buttons too), and up or down shows the
  controls; Back hides them.
- **Auto-download:** Downloads → "Download new scenes from favourites".
  New scenes with pornstars you follow download by themselves (checked
  about every 30 minutes), optionally only on Wi-Fi, in 480p, 720p, 1080p or
  the original file. Keep the newest 3, 5, 10 or 20; older automatic
  downloads are deleted. Downloads you start yourself are never touched.
- **Home screen widgets:** "Continue watching" and "New from favourites",
  with pictures; tap a scene to open it. They refresh after you watch
  something and with each check for new scenes.
- **Admin tools** (You → Admin tools, admins only):
  - AI markers: review the tagger's suggestions with a picture of each
    moment, play it, keep or throw away one or a whole scene.
  - Auto-import: whether it is running, the last check, Check now and
    Identify again, and the scenes that still have no studio or performers.
  - Tasks: what the server is doing, with progress, and Stop.
  - Statistics: plays, who is watching, most watched scenes, performers and
    tags, and storage, for 7, 30 or 90 days or a year.

## v3.3 (2026-11-09)

### Backups to Google Drive
- **Settings → Maintenance → Google Drive:** connect your own Google Drive
  and every daily backup is uploaded after it is made, into a folder called
  "Stasharr backups". Choose how many to keep there (older ones are
  deleted), upload the newest backup by hand, and copy a backup from Drive
  back to the server to restore it.
- Uses your own Google Cloud OAuth client ("TVs and Limited Input
  devices"); the settings page walks through making one. You sign in with
  a code on google.com/device, and Stasharr can only see the files it put
  on your Drive.

### Cast from the site
- The site's video player has a **Chromecast** button (in Chrome and Edge).
  Casting needs the site to be opened over HTTPS, for example through a
  reverse proxy; on plain http the browser hides the button. The admin
  interface keeps its own switch under Settings → Interface.

### Android app: categories and channels
- **Tag clips play in the app:** category tiles on the home screen, the
  Categories page and Browse → Tags now play their clips side by side
  (up to four at once) while they are on screen, instead of one at a time.
  Previews off under You shows the still picture only.
- **View all** on the Categories and Channels home rows opens a page with
  every category or channel, a search box and sort options: most scenes,
  name, recently added, most performers or latest scene, rating, and
  random.
### Android app tells you about updates
- The app checks GitHub for a newer version when it opens (at most every 12
  hours) and shows what's new; **Download** opens the new version in your
  browser, and you tap the download to install it over the old one (your
  login is kept). **Later** waits for the next version. **You → Check for
  updates** checks straight away.
- 1.0.44 installed updates by itself, which needs the permission to install
  apps; Play Protect flags unknown apps with that permission as harmful.
  From 1.0.45 the app no longer has it, and installing is done by Android
  from the download, as with the first install.
- Each Android release now says what's new.

## v3.2 (2026-11-09)

### Reels on or off
- **Settings → Auto-import → Reels** has a switch for the whole server. Off
  hides reels everywhere (the menu, scene pages, the site's Reels page and
  home row, the Android app's Reels tab) and stops making new ones. Reels
  already made are kept, so switching back on brings them back.

## v3.1 (2026-11-09)

### Settings list scrolls
- Settings: the list of sections on the left scrolls on its own when it is
  taller than the window, so everything down to Troubleshooting mode can
  be reached.

## v3.0 (2026-11-09)

### Menus fit on one line
- Admin menu bar: Audio, Playlists, Reels and Describe are under **More**
  on wide screens (phones and tablets still list them all), and on screens
  up to 1600 pixels wide the menu shows icons only, with the name when
  pointing at one.
- Site header: a tighter menu on laptops, and the ☰ menu below 1340 pixels,
  so it stays on one line in every language.

## v2.9 (2026-11-09)

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
