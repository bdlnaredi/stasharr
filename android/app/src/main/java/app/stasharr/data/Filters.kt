package app.stasharr.data

import kotlinx.serialization.json.JsonArray
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.JsonObjectBuilder
import kotlinx.serialization.json.JsonPrimitive
import kotlinx.serialization.json.buildJsonObject
import kotlinx.serialization.json.jsonObject
import kotlinx.serialization.json.put
import kotlinx.serialization.json.putJsonObject
import java.time.LocalDate

/** A way to order a list: the server's sort key and a label. */
data class SortOption(val key: String, val label: String, val defaultDesc: Boolean = true)

/** How a list is ordered. Random lists use a seed so paging stays stable. */
data class SortState(val option: SortOption, val desc: Boolean = option.defaultDesc, val seed: Int = 1) {
    val serverKey: String get() = if (option.key == "random") "random_$seed" else option.key
}

object Sorts {
    val scenes = listOf(
        SortOption("created_at", "Recently added"),
        SortOption("date", "Release date"),
        SortOption("rating", "Rating"),
        SortOption("play_count", "Most played"),
        SortOption("last_played_at", "Last played"),
        SortOption("o_counter", "Most O"),
        SortOption("duration", "Length"),
        SortOption("resolution", "Resolution"),
        SortOption("filesize", "File size"),
        SortOption("title", "Title", defaultDesc = false),
        SortOption("random", "Random"),
    )
    val performers = listOf(
        SortOption("scenes_count", "Most scenes"),
        SortOption("name", "Name", defaultDesc = false),
        SortOption("created_at", "Recently added"),
        SortOption("latest_scene", "Latest scene"),
        SortOption("rating", "Rating"),
        SortOption("play_count", "Most played"),
        SortOption("o_counter", "Most O"),
        SortOption("birthdate", "Age", defaultDesc = false),
        SortOption("random", "Random"),
    )
    val studios = listOf(
        SortOption("scenes_count", "Most scenes"),
        SortOption("name", "Name", defaultDesc = false),
        SortOption("created_at", "Recently added"),
        SortOption("latest_scene", "Latest scene"),
        SortOption("rating", "Rating"),
        SortOption("o_counter", "Most O"),
        SortOption("random", "Random"),
    )
    val tags = listOf(
        SortOption("scenes_count", "Most scenes"),
        SortOption("name", "Name", defaultDesc = false),
        SortOption("created_at", "Recently added"),
        SortOption("performers_count", "Most performers"),
        SortOption("random", "Random"),
    )
}

private fun JsonObjectBuilder.intCriterion(key: String, value: Int, modifier: String, value2: Int? = null) {
    putJsonObject(key) {
        put("value", value)
        if (value2 != null) put("value2", value2)
        put("modifier", modifier)
    }
}

/** Minimum star rating as a filter; ratings are stored out of 100. */
private fun JsonObjectBuilder.minRating(stars: Int) {
    if (stars > 0) intCriterion("rating100", stars * 20 - 1, "GREATER_THAN")
}

enum class LengthFilter(val label: String) { Any("Any"), Short("Under 10 min"), Medium("10–30 min"), Long("Over 30 min") }
enum class WatchFilter(val label: String) { Any("Any"), Unwatched("Not watched"), InProgress("In progress"), Watched("Watched") }
// the server's "greater than" is above the whole named band, so each option
// names the band just below it: above 540p means 720p and up
enum class ResolutionFilter(val label: String, val enumValue: String?) {
    Any("Any", null), HD("720p+", "WEB_HD"), FullHD("1080p+", "STANDARD_HD"), UHD("4K", "QUAD_HD"),
}
enum class OrientationFilter(val label: String, val enumValue: String?) {
    Any("Any", null), Landscape("Landscape", "LANDSCAPE"), Portrait("Portrait", "PORTRAIT"),
}
enum class AddedFilter(val label: String, val days: Int?) {
    Any("Any time", null), Week("Past week", 7), Month("Past month", 30), Year("Past year", 365),
}

/** Filters for scene lists. */
data class SceneFilters(
    val minRating: Int = 0,
    val resolution: ResolutionFilter = ResolutionFilter.Any,
    val length: LengthFilter = LengthFilter.Any,
    val watched: WatchFilter = WatchFilter.Any,
    val orientation: OrientationFilter = OrientationFilter.Any,
    val added: AddedFilter = AddedFilter.Any,
    val organized: Boolean = false,
    val favoritePerformers: Boolean = false,
    val hasMarkers: Boolean = false,
    val interactive: Boolean = false,
    val tags: List<TagRef> = emptyList(),
) {
    val count: Int
        get() = listOf(
            minRating > 0, resolution != ResolutionFilter.Any, length != LengthFilter.Any,
            watched != WatchFilter.Any, orientation != OrientationFilter.Any, added != AddedFilter.Any,
            organized, favoritePerformers, hasMarkers, interactive, tags.isNotEmpty(),
        ).count { it }

    fun toJson(): JsonObject = buildJsonObject {
        minRating(minRating)
        resolution.enumValue?.let { putJsonObject("resolution") { put("value", it); put("modifier", "GREATER_THAN") } }
        when (length) {
            LengthFilter.Any -> Unit
            LengthFilter.Short -> intCriterion("duration", 600, "LESS_THAN")
            LengthFilter.Medium -> intCriterion("duration", 600, "BETWEEN", 1800)
            LengthFilter.Long -> intCriterion("duration", 1800, "GREATER_THAN")
        }
        when (watched) {
            WatchFilter.Any -> Unit
            WatchFilter.Unwatched -> intCriterion("play_count", 0, "EQUALS")
            WatchFilter.Watched -> intCriterion("play_count", 0, "GREATER_THAN")
            WatchFilter.InProgress -> intCriterion("resume_time", 0, "GREATER_THAN")
        }
        orientation.enumValue?.let {
            putJsonObject("orientation") { put("value", JsonArray(listOf(JsonPrimitive(it)))) }
        }
        added.days?.let { d ->
            putJsonObject("created_at") {
                put("value", LocalDate.now().minusDays(d.toLong()).toString())
                put("modifier", "GREATER_THAN")
            }
        }
        if (organized) put("organized", true)
        if (favoritePerformers) put("performer_favorite", true)
        if (hasMarkers) put("has_markers", "true")
        if (interactive) put("interactive", true)
        if (tags.isNotEmpty()) {
            putJsonObject("tags") {
                put("value", JsonArray(tags.map { JsonPrimitive(it.id) }))
                put("modifier", "INCLUDES_ALL")
                put("depth", 0)
            }
        }
    }
}

/** Puts two scene filters together; tag lists are combined. */
fun mergeFilters(a: JsonObject?, b: JsonObject?): JsonObject? {
    if (a == null || a.isEmpty()) return b?.takeIf { it.isNotEmpty() }
    if (b == null || b.isEmpty()) return a
    val out = a.toMutableMap()
    for ((k, v) in b) {
        val existing = out[k]
        out[k] = if (k == "tags" && existing is JsonObject && v is JsonObject) {
            val ids = ((existing["value"] as? JsonArray ?: JsonArray(emptyList())) +
                (v["value"] as? JsonArray ?: JsonArray(emptyList()))).distinct()
            JsonObject(v.jsonObject + ("value" to JsonArray(ids)) + ("modifier" to JsonPrimitive("INCLUDES_ALL")))
        } else {
            v
        }
    }
    return JsonObject(out)
}

enum class Gender(val label: String) {
    FEMALE("Female"), MALE("Male"), TRANSGENDER_FEMALE("Trans female"), TRANSGENDER_MALE("Trans male"),
    NON_BINARY("Non-binary"), INTERSEX("Intersex"),
}

/** Filters for the performer list. */
data class PerformerFilters(
    val genders: Set<Gender> = emptySet(),
    val favorites: Boolean = false,
    val minRating: Int = 0,
    val minScenes: Int = 0,
) {
    val count: Int get() = listOf(genders.isNotEmpty(), favorites, minRating > 0, minScenes > 0).count { it }

    fun toJson(): JsonObject = buildJsonObject {
        if (genders.isNotEmpty()) {
            putJsonObject("gender") {
                put("value_list", JsonArray(genders.map { JsonPrimitive(it.name) }))
                put("modifier", "INCLUDES")
            }
        }
        if (favorites) put("filter_favorites", true)
        minRating(minRating)
        if (minScenes > 0) intCriterion("scene_count", minScenes - 1, "GREATER_THAN")
    }
}

/** Filters for the studio list. */
data class StudioFilters(
    val favorites: Boolean = false,
    val minRating: Int = 0,
    val minScenes: Int = 0,
) {
    val count: Int get() = listOf(favorites, minRating > 0, minScenes > 0).count { it }

    fun toJson(): JsonObject = buildJsonObject {
        if (favorites) put("favorite", true)
        minRating(minRating)
        if (minScenes > 0) intCriterion("scene_count", minScenes - 1, "GREATER_THAN")
    }
}

/** Filters for the tag list. */
data class TagFilters(
    val favorites: Boolean = false,
    val minScenes: Int = 0,
) {
    val count: Int get() = listOf(favorites, minScenes > 0).count { it }

    fun toJson(): JsonObject = buildJsonObject {
        if (favorites) put("favorite", true)
        if (minScenes > 0) {
            putJsonObject("scene_count") {
                put("value", minScenes - 1)
                put("modifier", "GREATER_THAN")
                put("depth", 0)
            }
        }
    }
}
