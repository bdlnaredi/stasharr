package app.stasharr.data

import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable

@Serializable
data class Paths(
    val screenshot: String? = null,
    val preview: String? = null,
    val stream: String? = null,
    val caption: String? = null,
)

@Serializable
data class Caption(
    @SerialName("language_code") val lang: String = "",
    @SerialName("caption_type") val type: String = "",
)

@Serializable
data class MediaFile(
    val path: String = "",
    val duration: Double = 0.0,
    val width: Int = 0,
    val height: Int = 0,
    @SerialName("video_codec") val videoCodec: String = "",
    @SerialName("audio_codec") val audioCodec: String = "",
    val size: Long = 0,
)

@Serializable
data class StudioRef(
    val id: String,
    val name: String,
    @SerialName("image_path") val imagePath: String? = null,
)

@Serializable
data class PerformerRef(
    val id: String,
    val name: String,
    @SerialName("image_path") val imagePath: String? = null,
)

@Serializable
data class TagRef(val id: String, val name: String)

@Serializable
data class Marker(
    val id: String,
    val title: String = "",
    val seconds: Double = 0.0,
    @SerialName("primary_tag") val primaryTag: TagRef? = null,
) {
    val label: String get() = title.ifBlank { primaryTag?.name ?: "Marker" }
}

@Serializable
data class StreamEndpoint(
    val url: String,
    @SerialName("mime_type") val mimeType: String? = null,
    val label: String? = null,
)

@Serializable
data class Scene(
    val id: String,
    val title: String? = null,
    val details: String? = null,
    val date: String? = null,
    val rating100: Int? = null,
    @SerialName("o_counter") val oCounter: Int? = null,
    @SerialName("play_count") val playCount: Int? = null,
    @SerialName("resume_time") val resumeTime: Double? = null,
    val files: List<MediaFile> = emptyList(),
    val paths: Paths = Paths(),
    val studio: StudioRef? = null,
    val performers: List<PerformerRef> = emptyList(),
    val tags: List<TagRef> = emptyList(),
    @SerialName("scene_markers") val markers: List<Marker> = emptyList(),
    @SerialName("sceneStreams") val streams: List<StreamEndpoint> = emptyList(),
    @SerialName("in_watch_later") val inWatchLater: Boolean = false,
    val captions: List<Caption>? = null,
) {
    val file: MediaFile? get() = files.firstOrNull()
    val duration: Double get() = file?.duration ?: 0.0
    val isAudio: Boolean
        get() = file?.let { it.videoCodec.isBlank() && it.width == 0 && it.height == 0 } ?: false

    val displayTitle: String
        get() = title?.takeIf { it.isNotBlank() }
            ?: file?.path?.substringAfterLast('/')?.substringAfterLast('\\')?.substringBeforeLast('.')
            ?: "Untitled"

    /** 0..1 how far the user got, or 0 when not started. */
    val progress: Float
        get() {
            val r = resumeTime ?: 0.0
            val d = duration
            return if (r > 0 && d > 0) (r / d).toFloat().coerceIn(0f, 1f) else 0f
        }

    val year: String? get() = date?.take(4)
}

@Serializable
data class Performer(
    val id: String,
    val name: String,
    @SerialName("image_path") val imagePath: String? = null,
    @SerialName("scene_count") val sceneCount: Int = 0,
    val favorite: Boolean = false,
    val disambiguation: String? = null,
    val country: String? = null,
    val birthdate: String? = null,
    val details: String? = null,
)

@Serializable
data class Studio(
    val id: String,
    val name: String,
    @SerialName("image_path") val imagePath: String? = null,
    @SerialName("scene_count") val sceneCount: Int = 0,
)

@Serializable
data class Tag(
    val id: String,
    val name: String,
    @SerialName("scene_count") val sceneCount: Int = 0,
)

@Serializable
data class PlaylistCover(val paths: Paths = Paths())

@Serializable
data class Playlist(
    val id: String,
    val name: String,
    val description: String = "",
    @SerialName("watch_later") val watchLater: Boolean = false,
    @SerialName("scene_count") val sceneCount: Int = 0,
    val cover: PlaylistCover? = null,
)

@Serializable
data class ReelScene(
    val id: String,
    val title: String? = null,
    val studio: StudioRef? = null,
    val performers: List<PerformerRef> = emptyList(),
    val paths: Paths = Paths(),
)

@Serializable
data class ReelSegment(val start: Double = 0.0, val end: Double = 0.0)

@Serializable
data class Reel(
    val id: String,
    val stream: String,
    val duration: Double = 0.0,
    val scene: ReelScene,
    val segments: List<ReelSegment> = emptyList(),
) {
    /** Where the reel starts in the full scene, in seconds. */
    val start: Double get() = segments.firstOrNull()?.start ?: 0.0
}

@Serializable
data class Me(
    val username: String,
    val role: String = "ADMIN",
    @SerialName("can_view_reels") val canViewReels: Boolean = true,
    @SerialName("multi_user_enabled") val multiUserEnabled: Boolean = false,
)

data class Page<T>(val count: Int, val items: List<T>)

/** Ways to order scene lists. */
enum class SceneSort(val label: String, val sort: String, val desc: Boolean = true) {
    RecentlyAdded("Recently added", "created_at"),
    Newest("Newest", "date"),
    TopRated("Top rated", "rating"),
    MostPlayed("Most played", "play_count"),
    LastPlayed("Last played", "last_played_at"),
    Longest("Longest", "duration"),
    Random("Random", "random"),
    Title("A–Z", "title", desc = false),
}
