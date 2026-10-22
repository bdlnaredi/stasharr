# Stasharr

Stasharr is a fork of [Stash](https://github.com/stashapp/stash) that adds the
things Stash does not do on its own:

- **Multi-user logins** with roles (Admin, Editor, Viewer)
- **Tiers** that decide which part of the library each user can see
- **Per-user play history, O history, ratings and resume position**
- **Auto-import**: new files are scanned and identified automatically, using
  ThePornDB, StashDB and any other stash-boxes or scrapers you have set up
- **Reels**: automatic 15 second vertical clips of the best moments of a scene,
  with a swipe-through Reels page
- **Trash bin**: deleted files can be restored, and are deleted for good after
  30 days
- **Duplicate cleanup**: keeps the best copy of each duplicate and moves the
  rest to the trash bin, without losing tags or history
- **Playlists, Watch later and Continue watching**, per user
- **File renamer**: moves and renames files from their metadata with a
  template such as `{studio}/{date} - {title}`

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

An Unraid template is in [`unraid/stasharr.xml`](unraid/stasharr.xml).

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

A reel is a 15 second vertical (9:16) MP4 made from the best moments of a
scene. The moments are chosen in this order:

1. **Markers**: if the scene has markers (your own, or from ThePornDB/StashDB),
   the reel is built around them
2. **Interactive heatmap**: for scenes with a funscript, the most intense parts
3. **Audio and motion**: otherwise ffmpeg finds the loudest, busiest parts of
   the video, skipping intros and credits

Make reels from a scene's **⋯** menu (**Make reel**), from the scene's **Reels**
tab, for the whole library under Settings → Auto-import, or automatically for
new scenes. Watch them on the **Reels** page, which plays one reel per screen as
you scroll. Reels are stored in your generated folder under `reels/`.

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

## Building from source

Stasharr builds the same way as Stash; see
[docs/DEVELOPMENT.md](docs/DEVELOPMENT.md). Every push to `main` builds and
publishes the Docker image `ghcr.io/bdlnaredi/stasharr` with GitHub Actions.

## Licence

Stasharr is a fork of Stash and, like Stash, is licensed under the
[GNU AGPL v3](LICENSE). The original Stash README is kept in
[docs/UPSTREAM_README.md](docs/UPSTREAM_README.md).
