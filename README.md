# Stasharr

Stasharr is a fork of [Stash](https://github.com/stashapp/stash) that adds the
things Stash does not do on its own:

- **Multi-user logins** with roles (Admin, Editor, Viewer)
- **Tiers** that decide which part of the library each user can see
- **Per-user play history, O history, ratings and resume position**
- **Auto-import**: new files are scanned and identified automatically, using
  ThePornDB, StashDB and any other stash-boxes or scrapers you have set up
- **Reels**: automatic 15 second vertical clips of the best moments of a scene,
  with a swipe-through Reels page, view counts and comments
- **Trash bin**: deleted files can be restored, and are deleted for good after
  30 days
- **Duplicate cleanup**: keeps the best copy of each duplicate and moves the
  rest to the trash bin, without losing tags or history
- **Playlists, Watch later and Continue watching**, per user
- **File renamer**: moves and renames files from their metadata with a
  template such as `{studio}/{date} - {title}`
- **Recommendations**: "Similar" scenes on every scene page and "Recommended
  for you" on the home page, per user
- **Phone app**: install Stasharr on your phone's home screen
- **Android app**: a native app with a Plex-style home screen (or the site's
  storefront), full screen player, picture-in-picture, Chromecast, downloads
  and Reels
- **Audio files**: mp3, m4a, flac and other audio files are added as scenes
- **Site front end**: viewers get a storefront-style site (featured banner,
  top pornstars, trending, categories) in five styles and five languages,
  with likes, favourites, ratings, comments and view counts
- **AI markers**: the optional Stasharr Tagger container recognises what
  happens in a scene and suggests markers, on the CPU or an NVIDIA card
- **Smart playlists, notifications, scheduled backups and metadata refresh**
- **A new login page** with pictures or previews from the library behind it
- **Tag clips**: every tag gets a short looping clip of its moment

Everything else works exactly like Stash, and an existing Stash database is
upgraded in place the first time Stasharr starts. The web interface runs on
port **2704**.


⚠️ WARNING: EARLY-STAGE SOFTWARE

This application is new and under active development. Please read before installing.

It may be broken. Features may be incomplete, unstable, or not work as expected. Bugs are likely.
Updates will be frequent. Releases may be pushed often and without notice, and may include breaking changes to configuration, container paths, environment variables, or data formats.
Data loss is possible. Updates, bugs, or misconfiguration could corrupt, overwrite, or delete your data.
Back up first. Before installing or updating, back up your appdata and any data the container can access. Do not point this container at the only copy of anything important.
Pin your version. Consider using a specific image tag instead of latest, and disable auto-updates (including the CA Auto Update plugin and Watchtower) until the project matures.

NO WARRANTY / NO LIABILITY: This software is provided “as is,” without warranty of any kind, express or implied. The author(s) and contributors accept no liability for any data loss, system damage, downtime, or other harm arising from its installation or use. You use this software entirely at your own risk.

## Install with Docker

```yaml
services:
  stasharr:
    image: ghcr.io/bdlnaredi/stasharr:latest
    container_name: stasharr
    restart: unless-stopped
    ports:
      - "2704:2704"
    environment:
      - STASH_STASH=/data/
      - STASH_GENERATED=/generated/
      - STASH_METADATA=/metadata/
      - STASH_CACHE=/cache/
      - STASH_PORT=2704
      - TZ=Asia/Kolkata
    volumes:
      - ./config:/root/.stash
      - ./data:/data
      - ./metadata:/metadata
      - ./cache:/cache
      - ./blobs:/blobs
      - ./generated:/generated
```

Then open `http://YOUR-SERVER:2704`.

**Moving from Stash:** point the `/root/.stash` volume at your existing Stash
config folder (and the other volumes at the same folders as before). Back up
`stash-go.sqlite` first: Stasharr adds new tables to the database, and after
that the database can no longer be opened by upstream Stash.

An Unraid template is in [`unraid/stasharr.xml`](unraid/stasharr.xml). For AI
markers, also run the Stasharr Tagger (see [AI markers](#ai-markers)); the
`docker-compose.yml` in this repository has it ready as an optional block.

See [CHANGELOG.md](CHANGELOG.md) for what changed in each version. The same
list is under **Settings → Changelog** in Stasharr.

## Users, roles and tiers

1. Go to **Settings → Security** and set a username and password. This is the
   **owner** account. It always has full access and is never listed as a
   normal user.
2. Go to **Settings → Users & tiers** to add people.

| Role   | Can do |
|--------|--------|
| Admin  | Everything, including settings, users and deleting files |
| Editor | Edit metadata, scrape, identify, run tasks, make reels. Cannot change settings, manage users or delete files from disk |
| Viewer | Browse and play what their tier allows. Has their own ratings, play count, O count and resume position. Cannot edit anything |

A **tier** is a library access policy you give to a group of users, for
example *Free*, *Premium* and *VIP*:

- **Library folders**: only content inside these folders is visible
- **Hidden tags**: content with any of these tags is hidden, and so are the tags
- **Required tags**: only content with at least one of these tags is visible
- **Downloads** and **Reels** can be switched on or off per tier

Users without a tier see the whole library. Admins are never restricted.

Each user can also get their own **API key** (key button in the user list) for
apps and scripts. Users change their own password under **Settings → Account**.

### What is per user

| Per user | Shared by everyone |
|----------|--------------------|
| Scene rating | Image and gallery ratings |
| Scene play count and play history | Image O counts |
| Scene O count and O history | Performer, studio and tag favourites |
| Resume position and total play time | Saved filters and interface settings |

The owner account keeps using the original Stash values, so nothing you have
recorded so far is lost.

## Auto-import

**Settings → Auto-import** turns on the library watcher. When new videos
appear in your library folders, Stasharr waits until they stop changing (so
downloads can finish), then:

1. **Scans** only the folders that changed, including the phash fingerprints
   that ThePornDB and StashDB match on
2. **Identifies** the new scenes. It uses your saved Identify defaults from
   Settings → Tasks → Identify if you have them; otherwise ThePornDB first,
   then StashDB, then your other stash-boxes, then scrapers that can identify
   scenes from file details
3. **Generates** covers, previews and sprites with your default Generate settings
4. Optionally **makes a reel** for each new scene

Network shares (SMB/NFS) and some Docker setups do not report new files to the
app, so the watcher also checks the library on a timer (every 30 minutes by
default; you can change it or turn it off).

Add your ThePornDB and StashDB API keys under **Settings → Metadata Providers →
Stash-box endpoints** first.

## Reels

A reel is a continuous 15 second vertical (9:16) MP4 clip of a scene. Each
scene gets 3 reels by default, each from a different high point, so a reel
plays like a real moment from the scene rather than a montage. The high
points are found from:

1. **Interactive heatmap**: for scenes with a funscript, the most intense parts
2. **Audio and motion**: otherwise ffmpeg finds the loudest, busiest parts,
   skipping intros and credits. It only reads the audio and the key frames, so
   this is quick
3. **Markers**: the part right after each marker (your own, or from
   ThePornDB/StashDB) is preferred

Reels are kept apart, so they do not overlap; short scenes get fewer reels.
The reel length (5–90 seconds) and reels per scene (1–5) are set under
Settings → Auto-import. Making a reel only encodes the 15 seconds it uses, at
720×1280, with two encoder threads so the server stays responsive.

Make reels from a scene's **⋯** menu (**Make reel**), from the scene's **Reels**
tab, for the whole library under Settings → Auto-import (**Make reels**, or
**Remake all** to replace older reels), or automatically for new scenes.
Watch them on the **Reels** page, which plays one reel per screen as you
scroll. Reels are stored in your generated folder under `reels/`.

Each reel has its own picture, taken from the reel a third of the way in
(older reels get one the first time they are shown). The Reels player on the
site and in the Android app plays reels in a new random order every visit.
A reel counts as watched after two seconds of play, and the count shows on
reel cards and in the player. The 💬 button opens the scene's comments
next to the reel.

## AI markers

Stasharr can look through your scenes and suggest markers (Kissing,
Blowjob, Cowgirl, Doggy Style…) with the **Stasharr Tagger**, a small add-on
container that recognises what is happening in video frames with an
OpenCLIP model. Markers show as chapters on scene pages and help reels pick
their moments.

| Image | Runs on | Model (default) | First download |
|---|---|---|---|
| `ghcr.io/bdlnaredi/stasharr-tagger:latest` | processor | ViT-B-32 | about 600 MB |
| `ghcr.io/bdlnaredi/stasharr-tagger:cuda` | NVIDIA graphics card | ViT-L-14 (more accurate) | about 1.7 GB |

### Install on Unraid

1. For an NVIDIA card, install the **Nvidia-Driver** plugin from Community
   Applications first and check that it lists your card.
2. Open a terminal (the **>_** icon) and download the templates:

   ```
   wget -O /boot/config/plugins/dockerMan/templates-user/my-stasharr-tagger-nvidia.xml https://raw.githubusercontent.com/bdlnaredi/stasharr/main/unraid/stasharr-tagger-nvidia.xml
   wget -O /boot/config/plugins/dockerMan/templates-user/my-stasharr-tagger.xml https://raw.githubusercontent.com/bdlnaredi/stasharr/main/unraid/stasharr-tagger.xml
   ```

   (Saving the file from GitHub's normal page saves a web page, not the
   template, and gives an empty form.)
3. **Docker → Add Container**, pick **stasharr-tagger-nvidia** (or
   **stasharr-tagger** for the processor) under User templates and press
   **Apply**. The NVIDIA template already has `--runtime=nvidia`; leave
   `NVIDIA_VISIBLE_DEVICES` on `all` or paste one card's GPU UUID.
4. Watch the container's log: the first start downloads the model. It is
   ready when the log shows `Uvicorn running on http://0.0.0.0:2705`.

### Install with Docker Compose

Uncomment the `stasharr-tagger` block in [`docker-compose.yml`](docker-compose.yml).
For an NVIDIA card use the `:cuda` image and uncomment its `deploy` section
(needs the NVIDIA Container Toolkit).

### Use it

1. In Stasharr open **Settings → AI markers** and enter the tagger's
   address, for example `http://192.168.0.10:2705` (or
   `http://stasharr-tagger:2705` with Compose). The status shows **Ready**
   and, with a graphics card, its name. If it says CPU when you expected the
   card, the card is not reaching the container.
2. Press **Look at the rest** to go through the library. New scenes are
   looked at automatically after auto-import (switch: **Look at new
   scenes**). Progress is under Settings → Tasks.
3. Suggestions wait under **Waiting for review**. Click one to play that
   moment, then keep it (✓) or throw it away (✕), or do a whole scene at
   once. Kept suggestions become markers with a tag of the same name. Turn
   off **Review before saving** to save them straight away.

**Settings:** **Look every** (seconds between frames: fewer is faster, more
is more exact), **How sure** (higher gives fewer, more reliable markers) and
**Shortest marker**. Under **What to look for** you can rename, reword, turn
off or add labels; plain visual descriptions work best. Labels that do not
make markers (Talking, Title card, Scenery) help it ignore the parts in
between.

**Speed**, for a 30-minute scene looking every 4 seconds: about a minute or
two on a recent desktop processor, 5–15 minutes on a low-power one
(Celeron, N100), and well under a minute on an NVIDIA card. Only key frames
are decoded and the tagger runs at low priority with 2 threads by default
(`TAGGER_THREADS`).

The model is a general one, not trained for adult content, so expect some
wrong suggestions; the review list is there to catch them.

## Trash bin

Files you delete from the library (scenes, images, galleries, or duplicates
cleaned up) are moved to a trash bin instead of being removed straight away.
By default this is a `.stasharr-trash` folder at the top of each library
folder, so deleting is instant and needs no extra disk. If you set a trash
folder under **Settings → System**, that folder is used instead.

Under **Settings → Trash bin** you can restore files, delete them for good,
empty the bin, and choose how many days files are kept (30 by default; 0 keeps
them until you empty the bin). Restored files are scanned and identified again
like a new download. Files in the trash bin are never scanned.

## Duplicate cleanup

**Settings → Duplicate cleanup** finds copies of the same video using the
same fingerprints as Stash's duplicate checker. In each group it keeps the
best copy: highest resolution, then highest bitrate, then largest file. The
tags, performers, galleries, groups, URLs, stash IDs, play and O history,
ratings, markers and playlist entries of the other copies are copied to the
one that is kept, and the other files go to the trash bin. Press **Find
duplicates** to see the list first, untick any group you want to leave alone,
then press **Clean up**.

## Playlists, Watch later and Continue watching

Every user has their own playlists:

- **Watch later**: the bookmark button on a scene adds it to Watch later
- **Playlists**: the list button next to it adds the scene to a playlist, or
  select scenes in the Scenes list and choose **Add to playlist** from the ⋯
  menu. Open **Playlists** in the menu to play a playlist in order or
  shuffled, reorder it, rename it or delete it
- **Continue watching**: scenes you started but did not finish appear at the
  top of the home page and the Playlists page, and pick up where you left off

### Smart playlists

A smart playlist keeps itself up to date from a saved filter, for example
"4K, rated 4 stars or more, not watched". Filter and sort the Scenes list,
then choose **Save as smart playlist** from the ⋯ menu (in the Android app:
**Filters → Save as smart playlist**). Each smart playlist appears as a row
on the home page. To change its filter, set up the Scenes list again and save
it over the old one.

## Notifications

Under **Settings → Notifications**, send messages to Discord, Telegram, ntfy,
Gotify or any webhook when auto-import adds new scenes, when a new scene
cannot be identified, when a task fails, and when a scheduled backup or
metadata refresh finishes. **Send test** checks the settings.

## Backups and metadata refresh

Under **Settings → Backups & refresh**:

- **Back up every day** at the chosen hour, keeping the newest few (7 by
  default). Scheduled backups go in a `stasharr-auto` folder inside the
  backup folder. Any backup in the list can be **restored**; the current
  database is backed up first, so a restore can be undone.
- **Refresh automatically** checks StashDB, ThePornDB and your other
  stash-boxes again every few days for scenes, performers and studios you
  have already matched, and adds new details, images, performers and tags.

## File renamer

**Settings → File renamer** moves and renames video files using their
metadata, inside the library folder they are already in. The default template
is `{studio}/{date} - {title}`; the file extension is added for you, and `/`
makes folders. Tokens: `{title}`, `{studio}`, `{parent_studio}`, `{date}`,
`{year}`, `{month}`, `{performers}`, `{performer}`, `{code}`, `{director}`,
`{resolution}`, `{codec}`, `{stash_id}` and `{id}`. Parts that are empty for a
scene are left out together with their separators.

Press **Preview** to see what would change, then **Rename library**. You can
also rename one scene from its ⋯ menu (**Rename file**), or selected scenes
from the Scenes list. Turn on **Rename new scenes automatically** to rename
files as soon as auto-import has identified them. Scenes without a title are
left alone, subtitles and funscripts with the same name move with the video,
and empty folders left behind are removed.

## Recommendations

Recommendations come from the metadata you already have, so they need no AI
and no extra service:

- **Similar** (a tab on every scene page) lists scenes that share performers,
  the studio, the studio's network and tags with the scene. Rare tags count
  for more than tags that half the library has.
- **Recommended for you** (on the home page) looks at what you play, O and
  rate, builds a picture of the performers, studios and tags you like, and
  suggests scenes you have not watched yet. Low ratings push similar scenes
  down. Every user gets their own, and it only shows once you have some
  history.

## Site front end

**Settings → Site look** turns on a storefront-style site for viewers (or
viewers and editors) instead of the library interface. Admins keep the
normal interface and open the site from the menu.

- **Home:** a featured banner that plays previews, a just-added ticker, an
  optional promo banner and rows the admin arranges, renames and hides:
  Continue watching, Latest, Top pornstars, Trending, Most viewed, Picked
  for you, New from your favourites, Liked scenes, Reels, Top rated,
  Categories, Channels, and rows for any tag. The featured scenes are the
  top rated, most popular, newest, random, or scenes with a chosen tag.
- **Pages:** scenes with sort tabs and filters (category, length, quality,
  channel), pornstars A–Z with rich pornstar pages, categories, channels,
  live search while typing, and My list.
- **Scene pages:** chapters from markers, Up next, theatre mode, likes,
  ratings (yours and everyone's average), view counts and comments.
- **Reels:** a home row and a swipe-through player with views and comments.
- **Five styles:** Velvet, Neon, Crimson, Blush and Onyx, plus your own logo
  and accent colour. Admins can preview a style before using it.
- **Five languages:** English, हिन्दी, Français, Русский and Español,
  chosen by each visitor from the globe button (the first visit follows the
  browser's language). Names, titles and your own row headings stay as you
  typed them.
- **Age check** before the site opens, asked once per browser.
- **Comments:** under every scene and on its reels. People can delete their
  own comments; admins can delete any.
- **Same look in the Android app** for site users (switch **Same look in
  the Android app**).

### Tag clips

Every tag without its own picture gets a 4-second silent loop from its best
scene (highest rated, then most played), starting just after a marker for
the tag when there is one (yours or an AI marker). Scenes another tag's clip
already uses come last, so tags do not repeat the same scene. Without a
marker, and with the Stasharr Tagger running, the clip starts at the frame
that looks most like the tag; otherwise at a reel moment or busy part that
no other tag's clip from that scene uses. The clips play on category
tiles, at the top of category pages, on tag cards in the admin interface
and in the Android app, and only load while they are on screen. Viewers see
a clip only when their tier lets them see the scene it came from.

Make them under **Settings → AI markers → Tag clips** (**Make clips** for
tags without one, **Remake all** after adding markers) or with **Make new
clip** on a tag's page. New tags get one after auto-import. They are kept in
the generated folder under `tagclips/`, about 0.5 MB each.

### Alerts for favourite pornstars

When a pornstar you follow (**Add to favourites** on her page) gets a new
scene, the bell in the site's header shows how many are new and lists them;
opening it marks them seen. The owner account also gets alerts for
performers marked favourite in the library. The Android app shows the same
list (My list, or You), a badge on that tab, and a phone notification about
every 30 minutes when there is something new; tap it to open the scene.
Notifications can be turned off in the app.

### Login page

The login box sits in the top right corner over drifting pictures or
moving previews from the library. Under **Settings → Site look → Login
page** choose **Moving previews**, **Scene pictures** or **Plain**, and
optionally only scenes with one tag. The login page is visible to anyone
who can reach your server, so use the tag if only some scenes should be
shown there.

## Statistics

The **Statistics** page (the chart icon in the menu bar) has a Stasharr part
for admins: plays per day, the most watched scenes, performers, tags and
reels, a table of who is watching (plays, last played and last signed in for
each account), what was played recently, scenes added per month, picture
quality, and how much space video files and each generated folder (reels,
previews, tag clips…) take. Choose 7 days, 30 days, 90 days or a year.

## Phone app

Stasharr can be installed on your phone's home screen and opens full screen,
like an app. Go to **Settings → Account → Install on this device**, or use
your browser's "Install app" / "Add to Home screen" menu item. On phones the
menu bar also has one-tap buttons for Reels and Playlists, and turning the
phone sideways while a video plays goes full screen.

Phones only install apps from secure `https://` addresses. If you open
Stasharr as `http://your-server:2704`, put it behind https first, for example
with [Tailscale Serve](https://tailscale.com/kb/1312/serve) or a reverse proxy
such as Caddy, Nginx Proxy Manager or SWAG. Over plain http you can still use
"Add to Home screen" as a shortcut.

## Audio files

Audio files in your library folders (`mp3`, `m4a`, `m4b`, `aac`, `flac`,
`wav`, `ogg`, `oga`, `opus`, `wma`) are added as scenes, so they can be
identified, tagged, rated, put in playlists and played like videos. Their
cover is the embedded album art, or a picture of the sound wave when there is
none. Previews, sprites, fingerprints and reels are skipped for them, since
they have no picture. Formats browsers cannot play (such as `wma`) are
converted to mp3 while they play. Audio files are listed under **Audio** in
the menu and marked AUDIO on their cards. Turn this off, or change the
extensions, under **Settings → Auto-import → Audio files**, then run a scan.

## Android app

Stasharr has its own Android app, made to feel like Plex or Netflix: a big
swipeable banner of picks for you, shelves for Continue watching,
Recommended, Recently added, Watch later, performers and studios, rich scene
pages with cast, chapters, tags and "More like this", Browse and search, a
full screen player that remembers where you stopped and keeps playing in a
small window (picture-in-picture), and swipe-up Reels.

**Install:** on your phone, open this repository's
[Releases page](https://github.com/bdlnaredi/stasharr/releases), find the
newest **Stasharr for Android** and download the `.apk` file. Open it and
allow your browser to install apps when Android asks. New versions install
over the old one.

**Sign in:** enter your server's address, for example `192.168.1.20:2704`
(the app adds `http://` and the port for you), and your Stasharr username and
password. Leave them empty if your server has no login. Away from home, use a
VPN such as Tailscale, or an https address.

**Storefront:** site users get the site's storefront in the app (admins can
switch it on under **You → Site layout**): featured banner, ticker, promo,
the admin's home rows, Scenes with sort tabs and filters, Pornstars A–Z,
pornstar pages, My list, likes and ratings.

**Previews and reels:** once scrolling stops, the banner or the top scene
card plays its preview (switch under You). Reels play in random order, count
views and show views and comments.

**Chromecast:** tap the cast button in the player to move the scene to the
TV from where you are, and back to the phone when casting ends.

**Downloads:** tap **Download** on a scene to watch it without the server.
Pick the original file or a smaller 1080p, 720p or 480p copy, which the
server makes first (it takes a few minutes and is kept on the server for a
week). Downloads are under **You → Downloads** and the download button on
Home, and still play when the server can't be reached. A user's tier must
allow downloading.

The app plays the original file first, as Android plays far more formats
than a browser, and falls back to the server's conversions when it cannot.
Every change to the `android/` folder builds a new version automatically.

## Building from source

Stasharr builds the same way as Stash; see
[docs/DEVELOPMENT.md](docs/DEVELOPMENT.md). Every push to `main` builds and
publishes the Docker image `ghcr.io/bdlnaredi/stasharr` with GitHub Actions.

## Licence

Stasharr is a fork of Stash and, like Stash, is licensed under the
[GNU AGPL v3](LICENSE). The original Stash README is kept in
[docs/UPSTREAM_README.md](docs/UPSTREAM_README.md).
