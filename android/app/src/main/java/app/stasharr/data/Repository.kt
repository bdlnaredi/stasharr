package app.stasharr.data

import kotlinx.serialization.json.JsonArray
import kotlinx.serialization.json.JsonElement
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.JsonPrimitive
import kotlinx.serialization.json.buildJsonArray
import kotlinx.serialization.json.buildJsonObject
import kotlinx.serialization.json.decodeFromJsonElement
import kotlinx.serialization.json.jsonPrimitive
import kotlinx.serialization.json.put
import kotlinx.serialization.json.putJsonObject

private const val SCENE_CARD = """
fragment SceneCard on Scene {
  id title date rating100 resume_time play_count
  files { duration width height video_codec audio_codec }
  paths { screenshot preview }
  studio { id name }
  performers { id name gender }
  created_at
}
"""

private const val SCENE_FULL = """
fragment SceneFull on Scene {
  id title details date rating100 o_counter play_count resume_time in_watch_later
  files { path duration width height video_codec audio_codec size format }
  paths { screenshot preview stream caption }
  captions { language_code caption_type }
  studio { id name image_path }
  performers { id name image_path gender }
  tags { id name }
  scene_markers { id title seconds primary_tag { id name } }
  sceneStreams { url mime_type label }
}
"""

private const val PERFORMER = "id name image_path scene_count favorite disambiguation country birthdate details"
private const val PLAYLIST = "id name description watch_later smart scene_count cover { paths { screenshot } }"

/** A filter applied to scene lists. */
sealed class SceneScope {
    data object All : SceneScope()
    data object Audio : SceneScope()
    data class ByPerformer(val id: String) : SceneScope()
    data class ByStudio(val id: String) : SceneScope()
    data class ByTag(val id: String) : SceneScope()

    fun toFilter(): JsonObject? = when (this) {
        All -> null
        Audio -> buildJsonObject {
            putJsonObject("video_codec") { put("value", ""); put("modifier", "IS_NULL") }
            putJsonObject("audio_codec") { put("value", ""); put("modifier", "NOT_NULL") }
        }
        is ByPerformer -> buildJsonObject {
            putJsonObject("performers") {
                put("value", buildJsonArray { add(JsonPrimitive(id)) })
                put("modifier", "INCLUDES_ALL")
            }
        }
        is ByStudio -> buildJsonObject {
            putJsonObject("studios") {
                put("value", buildJsonArray { add(JsonPrimitive(id)) })
                put("modifier", "INCLUDES")
                put("depth", -1)
            }
        }
        is ByTag -> buildJsonObject {
            putJsonObject("tags") {
                put("value", buildJsonArray { add(JsonPrimitive(id)) })
                put("modifier", "INCLUDES_ALL")
                put("depth", 0)
            }
        }
    }
}

class Repository(private val api: Api) {
    private val json get() = api.json

    private inline fun <reified T> JsonElement?.decode(default: T): T =
        if (this == null) default else runCatching { json.decodeFromJsonElement<T>(this) }.getOrDefault(default)

    private fun vars(block: kotlinx.serialization.json.JsonObjectBuilder.() -> Unit) = buildJsonObject(block)

    private fun findFilter(page: Int, perPage: Int, sort: String?, desc: Boolean, q: String?) = buildJsonObject {
        put("page", page)
        put("per_page", perPage)
        if (sort != null) put("sort", sort)
        put("direction", if (desc) "DESC" else "ASC")
        if (!q.isNullOrBlank()) put("q", q)
    }

    suspend fun me(): Me? {
        val data = api.gql("query { me { username role can_view_reels multi_user_enabled can_download } }")
        return data["me"].decode<Me?>(null)
    }

    suspend fun scenes(
        sort: SceneSort = SceneSort.RecentlyAdded,
        page: Int = 1,
        perPage: Int = 30,
        q: String? = null,
        scope: SceneScope = SceneScope.All,
        seed: Int = 0,
    ): Page<Scene> {
        val sortKey = if (sort == SceneSort.Random) "random_$seed" else sort.sort
        val v = vars {
            put("filter", findFilter(page, perPage, sortKey, sort.desc, q))
            scope.toFilter()?.let { put("scene_filter", it) }
        }
        val data = api.gql(
            """query(${'$'}filter: FindFilterType, ${'$'}scene_filter: SceneFilterType) {
              findScenes(filter: ${'$'}filter, scene_filter: ${'$'}scene_filter) { count scenes { ...SceneCard } }
            }""" + SCENE_CARD, v
        )
        val res = data.obj("findScenes") ?: return Page(0, emptyList())
        return Page(res.int("count"), res["scenes"].decode(emptyList()))
    }

    /** Scenes with a sort and filters, for Browse. */
    suspend fun findScenes(sort: SortState, page: Int, perPage: Int, q: String?, filter: JsonObject?): Page<Scene> {
        val v = vars {
            put("filter", findFilter(page, perPage, sort.serverKey, sort.desc, q))
            if (filter != null && filter.isNotEmpty()) put("scene_filter", filter)
        }
        val data = api.gql(
            """query(${'$'}filter: FindFilterType, ${'$'}scene_filter: SceneFilterType) {
              findScenes(filter: ${'$'}filter, scene_filter: ${'$'}scene_filter) { count scenes { ...SceneCard } }
            }""" + SCENE_CARD, v
        )
        val res = data.obj("findScenes") ?: return Page(0, emptyList())
        return Page(res.int("count"), res["scenes"].decode(emptyList()))
    }

    suspend fun findPerformers(sort: SortState, page: Int, perPage: Int, q: String?, filter: JsonObject?): Page<Performer> {
        val v = vars {
            put("filter", findFilter(page, perPage, sort.serverKey, sort.desc, q))
            if (filter != null && filter.isNotEmpty()) put("pf", filter)
        }
        val data = api.gql(
            """query(${'$'}filter: FindFilterType, ${'$'}pf: PerformerFilterType) {
              findPerformers(filter: ${'$'}filter, performer_filter: ${'$'}pf) { count performers { $PERFORMER } }
            }""", v
        )
        val res = data.obj("findPerformers") ?: return Page(0, emptyList())
        return Page(res.int("count"), res["performers"].decode(emptyList()))
    }

    suspend fun findStudios(sort: SortState, page: Int, perPage: Int, q: String?, filter: JsonObject?): Page<Studio> {
        val v = vars {
            put("filter", findFilter(page, perPage, sort.serverKey, sort.desc, q))
            if (filter != null && filter.isNotEmpty()) put("sf", filter)
        }
        val data = api.gql(
            """query(${'$'}filter: FindFilterType, ${'$'}sf: StudioFilterType) {
              findStudios(filter: ${'$'}filter, studio_filter: ${'$'}sf) { count studios { id name image_path scene_count } }
            }""", v
        )
        val res = data.obj("findStudios") ?: return Page(0, emptyList())
        return Page(res.int("count"), res["studios"].decode(emptyList()))
    }

    suspend fun findTags(sort: SortState, page: Int, perPage: Int, q: String?, filter: JsonObject?): Page<Tag> {
        val v = vars {
            put("filter", findFilter(page, perPage, sort.serverKey, sort.desc, q))
            if (filter != null && filter.isNotEmpty()) put("tf", filter)
        }
        val data = api.gql(
            """query(${'$'}filter: FindFilterType, ${'$'}tf: TagFilterType) {
              findTags(filter: ${'$'}filter, tag_filter: ${'$'}tf) { count tags { id name scene_count } }
            }""", v
        )
        val res = data.obj("findTags") ?: return Page(0, emptyList())
        return Page(res.int("count"), res["tags"].decode(emptyList()))
    }

    suspend fun scene(id: String): Scene? {
        val data = api.gql(
            "query(\$id: ID!) { findScene(id: \$id) { ...SceneFull } }$SCENE_FULL",
            vars { put("id", id) }
        )
        return data["findScene"].decode<Scene?>(null)
    }

    suspend fun continueWatching(limit: Int = 20): List<Scene> {
        val data = api.gql(
            "query(\$limit: Int) { continueWatching(limit: \$limit) { ...SceneCard } }$SCENE_CARD",
            vars { put("limit", limit) }
        )
        return data["continueWatching"].decode(emptyList())
    }

    suspend fun recommended(limit: Int = 20): List<Scene> {
        val data = api.gql(
            "query(\$limit: Int) { recommendedScenes(limit: \$limit) { ...SceneCard } }$SCENE_CARD",
            vars { put("limit", limit) }
        )
        return data["recommendedScenes"].decode(emptyList())
    }

    suspend fun similar(id: String, limit: Int = 20): List<Scene> {
        val data = api.gql(
            "query(\$id: ID!, \$limit: Int) { findSimilarScenes(scene_id: \$id, limit: \$limit) { ...SceneCard } }$SCENE_CARD",
            vars { put("id", id); put("limit", limit) }
        )
        return data["findSimilarScenes"].decode(emptyList())
    }

    suspend fun performers(page: Int = 1, perPage: Int = 40, q: String? = null, sort: String = "scenes_count"): Page<Performer> {
        val data = api.gql(
            """query(${'$'}filter: FindFilterType) {
              findPerformers(filter: ${'$'}filter) { count performers { $PERFORMER } }
            }""",
            vars { put("filter", findFilter(page, perPage, sort, sort != "name", q)) }
        )
        val res = data.obj("findPerformers") ?: return Page(0, emptyList())
        return Page(res.int("count"), res["performers"].decode(emptyList()))
    }

    suspend fun performer(id: String): Performer? {
        val data = api.gql("query(\$id: ID!) { findPerformer(id: \$id) { $PERFORMER } }", vars { put("id", id) })
        return data["findPerformer"].decode<Performer?>(null)
    }

    suspend fun studios(page: Int = 1, perPage: Int = 40, q: String? = null, sort: String = "scenes_count"): Page<Studio> {
        val data = api.gql(
            """query(${'$'}filter: FindFilterType) {
              findStudios(filter: ${'$'}filter) { count studios { id name image_path scene_count } }
            }""",
            vars { put("filter", findFilter(page, perPage, sort, sort != "name", q)) }
        )
        val res = data.obj("findStudios") ?: return Page(0, emptyList())
        return Page(res.int("count"), res["studios"].decode(emptyList()))
    }

    suspend fun studio(id: String): Studio? {
        val data = api.gql("query(\$id: ID!) { findStudio(id: \$id) { id name image_path scene_count } }", vars { put("id", id) })
        return data["findStudio"].decode<Studio?>(null)
    }

    suspend fun tags(page: Int = 1, perPage: Int = 60, q: String? = null): Page<Tag> {
        val data = api.gql(
            """query(${'$'}filter: FindFilterType) {
              findTags(filter: ${'$'}filter) { count tags { id name scene_count } }
            }""",
            vars { put("filter", findFilter(page, perPage, "scenes_count", true, q)) }
        )
        val res = data.obj("findTags") ?: return Page(0, emptyList())
        return Page(res.int("count"), res["tags"].decode(emptyList()))
    }

    suspend fun playlists(): List<Playlist> {
        val data = api.gql("query { findPlaylists { $PLAYLIST } }")
        return data["findPlaylists"].decode(emptyList())
    }

    suspend fun playlist(id: String): Pair<Playlist, List<Scene>>? {
        val data = api.gql(
            "query(\$id: ID!) { findPlaylist(id: \$id) { $PLAYLIST scenes { ...SceneCard } } }$SCENE_CARD",
            vars { put("id", id) }
        )
        val obj = data.obj("findPlaylist") ?: return null
        val p = obj.decode<Playlist?>(null) ?: return null
        return p to obj["scenes"].decode(emptyList())
    }

    /** Every smart playlist with its first scenes, for the home page. */
    suspend fun smartPlaylistRows(limit: Int = 20): List<Pair<Playlist, List<Scene>>> {
        val data = api.gql(
            "query(\$l: Int) { findPlaylists { $PLAYLIST scenes(limit: \$l) { ...SceneCard } } }$SCENE_CARD",
            vars { put("l", limit) }
        )
        return data.arr("findPlaylists").objects()
            .mapNotNull { o -> o.decode<Playlist?>(null)?.let { it to o["scenes"].decode<List<Scene>>(emptyList()) } }
            .filter { it.first.smart }
    }

    /** Saves a scene filter and sort as a smart playlist. */
    suspend fun createSmartPlaylist(name: String, filter: JsonObject?, sort: String, desc: Boolean, limit: Int = 100): Playlist? {
        val input = buildJsonObject {
            put("name", name)
            put("smart_rule", buildJsonObject {
                put("scene_filter", filter ?: JsonObject(emptyMap()))
                put("sort", sort)
                put("direction", if (desc) "DESC" else "ASC")
                put("limit", limit)
            })
        }
        val data = api.gql(
            "mutation(\$input: PlaylistCreateInput!) { playlistCreate(input: \$input) { $PLAYLIST } }",
            vars { put("input", input) }
        )
        return data["playlistCreate"].decode<Playlist?>(null)
    }

    suspend fun siteLook(): SiteLook? {
        val data = api.gql(
            """query { siteConfig { enabled style name tagline accent models_gender applies_to_me app_look age_gate logo_url
              home_rows { type title enabled tag { id name } } promo { enabled text link image } } }"""
        )
        return data["siteConfig"].decode<SiteLook?>(null)
    }

    /** A page of a site list: FEATURED, TRENDING, POPULAR or TOP_RATED, optionally for one model. */
    suspend fun sitePage(kind: String, page: Int, perPage: Int, performerId: String? = null): Page<Scene> {
        val data = api.gql(
            "query(\$k: SiteSceneKind!, \$p: Int, \$n: Int, \$m: ID) { siteScenes(kind: \$k, page: \$p, per_page: \$n, performer_id: \$m) { count scenes { ...SceneCard } } }$SCENE_CARD",
            vars { put("k", kind); put("p", page); put("n", perPage); if (performerId != null) put("m", performerId) }
        )
        val res = data.obj("siteScenes") ?: return Page(0, emptyList())
        return Page(res.int("count"), res["scenes"].decode(emptyList()))
    }

    /** Scenes in any order with a scene filter, for the storefront lists. */
    suspend fun storeScenes(sort: String, desc: Boolean, page: Int, perPage: Int, filter: JsonObject?, q: String? = null): Page<Scene> {
        val v = vars {
            put("filter", findFilter(page, perPage, sort, desc, q))
            if (filter != null && filter.isNotEmpty()) put("scene_filter", filter)
        }
        val data = api.gql(
            """query(${'$'}filter: FindFilterType, ${'$'}scene_filter: SceneFilterType) {
              findScenes(filter: ${'$'}filter, scene_filter: ${'$'}scene_filter) { count scenes { ...SceneCard } }
            }""" + SCENE_CARD, v
        )
        val res = data.obj("findScenes") ?: return Page(0, emptyList())
        return Page(res.int("count"), res["scenes"].decode(emptyList()))
    }

    suspend fun models(femaleOnly: Boolean, sort: String, desc: Boolean, page: Int, perPage: Int, q: String? = null, letter: String? = null): Page<Performer> {
        val pf = buildJsonObject {
            putJsonObject("scene_count") { put("value", 0); put("modifier", "GREATER_THAN") }
            if (femaleOnly) putJsonObject("gender") { put("value", "FEMALE"); put("modifier", "EQUALS") }
            if (!letter.isNullOrBlank()) putJsonObject("name") { put("value", "^$letter"); put("modifier", "MATCHES_REGEX") }
        }
        val data = api.gql(
            "query(\$f: FindFilterType, \$p: PerformerFilterType) { findPerformers(filter: \$f, performer_filter: \$p) { count performers { $PERFORMER } } }",
            vars { put("f", findFilter(page, perPage, sort, desc, q)); put("p", pf) }
        )
        val res = data.obj("findPerformers") ?: return Page(0, emptyList())
        return Page(res.int("count"), res["performers"].decode(emptyList()))
    }

    suspend fun modelProfile(id: String): ModelProfile? {
        val data = api.gql(
            """query(${'$'}id: ID!) { findPerformer(id: ${'$'}id) { id name gender image_path scene_count country birthdate ethnicity
              hair_color eye_color height_cm measurements career_start career_end tattoos piercings details alias_list site_favorite } }""",
            vars { put("id", id) }
        )
        return data["findPerformer"].decode<ModelProfile?>(null)
    }

    suspend fun coStars(id: String): List<CoStar> {
        val data = api.gql(
            "query(\$id: ID!) { siteCoStars(performer_id: \$id, limit: 12) { scenes performer { $PERFORMER } } }",
            vars { put("id", id) }
        )
        return data["siteCoStars"].decode(emptyList())
    }

    suspend fun sceneExtras(id: String): SceneExtras? {
        val data = api.gql(
            "query(\$id: ID!) { findScene(id: \$id) { id rating100 site_liked site_rating { average count } } }",
            vars { put("id", id) }
        )
        return data["findScene"].decode<SceneExtras?>(null)
    }

    suspend fun toggleLike(sceneId: String): Boolean {
        val data = api.gql("mutation(\$id: ID!) { siteToggleLike(scene_id: \$id) }", vars { put("id", sceneId) })
        return data["siteToggleLike"]?.jsonPrimitive?.content == "true"
    }

    suspend fun toggleFavoriteModel(id: String): Boolean {
        val data = api.gql("mutation(\$id: ID!) { siteToggleFavoriteModel(performer_id: \$id) }", vars { put("id", id) })
        return data["siteToggleFavoriteModel"]?.jsonPrimitive?.content == "true"
    }

    suspend fun likedScenes(page: Int = 1, perPage: Int = 40): Page<Scene> {
        val data = api.gql(
            "query(\$p: Int, \$n: Int) { siteLikedScenes(page: \$p, per_page: \$n) { count scenes { ...SceneCard } } }$SCENE_CARD",
            vars { put("p", page); put("n", perPage) }
        )
        val res = data.obj("siteLikedScenes") ?: return Page(0, emptyList())
        return Page(res.int("count"), res["scenes"].decode(emptyList()))
    }

    suspend fun fromFavorites(perPage: Int = 20): List<Scene> {
        val data = api.gql(
            "query(\$n: Int) { siteFromFavorites(per_page: \$n) { scenes { ...SceneCard } } }$SCENE_CARD",
            vars { put("n", perPage) }
        )
        return data.obj("siteFromFavorites")?.get("scenes").decode(emptyList())
    }

    suspend fun favoriteModels(): List<Performer> {
        val data = api.gql("query { siteFavoriteModels { $PERFORMER } }")
        return data["siteFavoriteModels"].decode(emptyList())
    }

    /** Site lists counted across every user: TRENDING, POPULAR or TOP_RATED. */
    suspend fun siteScenes(kind: String, perPage: Int = 20): List<Scene> {
        val data = api.gql(
            "query(\$k: SiteSceneKind!, \$n: Int) { siteScenes(kind: \$k, per_page: \$n) { scenes { ...SceneCard } } }$SCENE_CARD",
            vars { put("k", kind); put("n", perPage) }
        )
        return data.obj("siteScenes")?.get("scenes").decode(emptyList())
    }

    /** Pornstars with the most scenes, women only when the site says so. */
    suspend fun topModels(femaleOnly: Boolean, perPage: Int = 20): List<Performer> {
        val data = api.gql(
            "query(\$f: FindFilterType, \$p: PerformerFilterType) { findPerformers(filter: \$f, performer_filter: \$p) { performers { $PERFORMER } } }",
            vars {
                put("f", buildJsonObject { put("per_page", perPage); put("sort", "scenes_count"); put("direction", "DESC") })
                if (femaleOnly) put("p", buildJsonObject { putJsonObject("gender") { put("value", "FEMALE"); put("modifier", "EQUALS") } })
            }
        )
        return data.obj("findPerformers")?.get("performers").decode(emptyList())
    }

    /** The most used tags, for a categories row. */
    suspend fun topTags(perPage: Int = 16): List<Tag> {
        val data = api.gql(
            "query(\$f: FindFilterType) { findTags(filter: \$f) { tags { id name scene_count } } }",
            vars { put("f", buildJsonObject { put("per_page", perPage); put("sort", "scenes_count"); put("direction", "DESC") }) }
        )
        return data.obj("findTags")?.get("tags").decode<List<Tag>>(emptyList()).filter { it.sceneCount > 0 }
    }

    suspend fun offlineQualities(): List<OfflineQuality> {
        val data = api.gql("query { offlineQualities { id label height } }")
        return data["offlineQualities"].decode(emptyList())
    }

    suspend fun prepareOfflineDownload(sceneId: String, quality: String): OfflineDownload {
        val data = api.gql(
            "mutation(\$id: ID!, \$q: String!) { prepareOfflineDownload(scene_id: \$id, quality: \$q) { status progress size error url } }",
            vars { put("id", sceneId); put("q", quality) }
        )
        return data["prepareOfflineDownload"].decode(OfflineDownload())
    }

    suspend fun offlineDownload(sceneId: String, quality: String): OfflineDownload {
        val data = api.gql(
            "query(\$id: ID!, \$q: String!) { offlineDownload(scene_id: \$id, quality: \$q) { status progress size error url } }",
            vars { put("id", sceneId); put("q", quality) }
        )
        return data["offlineDownload"].decode(OfflineDownload())
    }

    suspend fun watchLaterScenes(): List<Scene> {
        val wl = playlists().firstOrNull { it.watchLater } ?: return emptyList()
        return playlist(wl.id)?.second ?: emptyList()
    }

    suspend fun toggleWatchLater(sceneId: String): Boolean {
        val data = api.gql("mutation(\$id: ID!) { watchLaterToggle(scene_id: \$id) }", vars { put("id", sceneId) })
        return data["watchLaterToggle"]?.jsonPrimitive?.content == "true"
    }

    suspend fun addToPlaylist(playlistId: String, sceneId: String) {
        api.gql(
            "mutation(\$id: ID!, \$s: [ID!]!) { playlistAddScenes(id: \$id, scene_ids: \$s) { id } }",
            vars { put("id", playlistId); put("s", JsonArray(listOf(JsonPrimitive(sceneId)))) }
        )
    }

    suspend fun createPlaylist(name: String, sceneId: String?): Playlist? {
        val input = buildJsonObject {
            put("name", name)
            if (sceneId != null) put("scene_ids", JsonArray(listOf(JsonPrimitive(sceneId))))
        }
        val data = api.gql(
            "mutation(\$input: PlaylistCreateInput!) { playlistCreate(input: \$input) { $PLAYLIST } }",
            vars { put("input", input) }
        )
        return data["playlistCreate"].decode<Playlist?>(null)
    }

    /** Reels newest first, or shuffled with a seed (kept for the next pages). */
    suspend fun reels(page: Int, perPage: Int = 15, seed: Int = 0): Page<Reel> {
        val data = api.gql(
            """query(${'$'}page: Int, ${'$'}per: Int, ${'$'}seed: Int) {
              findReels(page: ${'$'}page, per_page: ${'$'}per, seed: ${'$'}seed) { count reels {
                id stream cover duration segments { start end }
                scene { id title studio { id name } performers { id name } paths { screenshot } }
              } }
            }""",
            vars { put("page", page); put("per", perPage); put("seed", seed) }
        )
        val res = data.obj("findReels") ?: return Page(0, emptyList())
        return Page(res.int("count"), res["reels"].decode(emptyList()))
    }

    suspend fun saveActivity(sceneId: String, resumeTime: Double, playDuration: Double) {
        api.gql(
            "mutation(\$id: ID!, \$r: Float, \$d: Float) { sceneSaveActivity(id: \$id, resume_time: \$r, playDuration: \$d) }",
            vars { put("id", sceneId); put("r", resumeTime); put("d", playDuration) }
        )
    }

    suspend fun addPlay(sceneId: String) {
        api.gql("mutation(\$id: ID!) { sceneAddPlay(id: \$id) { count } }", vars { put("id", sceneId) })
    }

    suspend fun setRating(sceneId: String, rating100: Int?) {
        val input = buildJsonObject {
            put("id", sceneId)
            if (rating100 == null) put("rating100", kotlinx.serialization.json.JsonNull) else put("rating100", rating100)
        }
        api.gql("mutation(\$input: SceneUpdateInput!) { sceneUpdate(input: \$input) { id } }", vars { put("input", input) })
    }

    suspend fun addO(sceneId: String): Int {
        val data = api.gql("mutation(\$id: ID!) { sceneAddO(id: \$id) { count } }", vars { put("id", sceneId) })
        return data.obj("sceneAddO")?.int("count") ?: 0
    }
}
