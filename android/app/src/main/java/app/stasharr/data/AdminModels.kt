package app.stasharr.data

import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable

// What admins see in the app's Admin screen.

@Serializable
data class AiSuggestion(
    val id: String,
    val scene: Scene,
    val label: String,
    val seconds: Double,
    @SerialName("end_seconds") val endSeconds: Double,
    val score: Double,
)

@Serializable
data class AiSuggestions(
    val count: Int = 0,
    val scenes: Int = 0,
    val suggestions: List<AiSuggestion> = emptyList(),
)

@Serializable
data class AutoImportStatus(
    val enabled: Boolean = false,
    val identify: Boolean = false,
    val running: Boolean = false,
    @SerialName("poll_minutes") val pollMinutes: Int = 0,
    @SerialName("watched_paths") val watchedPaths: List<String> = emptyList(),
    @SerialName("last_check_at") val lastCheckAt: String? = null,
    @SerialName("last_result") val lastResult: String? = null,
)

@Serializable
data class ServerJob(
    val id: String,
    val status: String,
    val description: String,
    val progress: Double? = null,
    val error: String? = null,
    val subTasks: List<String>? = null,
)

@Serializable
data class StatsTotals(
    val scenes: Int = 0,
    val duration: Double = 0.0,
    val size: Double = 0.0,
    val performers: Int = 0,
    val tags: Int = 0,
    val studios: Int = 0,
    val reels: Int = 0,
    @SerialName("reel_views") val reelViews: Int = 0,
    val comments: Int = 0,
    val users: Int = 0,
    val plays: Int = 0,
)

@Serializable
data class StatsCount(val id: String, val name: String, val count: Int)

@Serializable
data class StatsScene(val scene: Scene, val plays: Int)

@Serializable
data class StatsDay(val day: String, val plays: Int)

@Serializable
data class StatsUser(
    val id: String,
    val name: String,
    val role: String,
    val plays: Int,
    @SerialName("last_play") val lastPlay: String? = null,
    @SerialName("last_login") val lastLogin: String? = null,
)

@Serializable
data class StatsPlay(val user: String, val scene: Scene, val at: String)

@Serializable
data class StatsFolder(val name: String, val bytes: Double)

@Serializable
data class ServerStats(
    val days: Int = 30,
    val totals: StatsTotals = StatsTotals(),
    @SerialName("plays_by_day") val playsByDay: List<StatsDay> = emptyList(),
    @SerialName("top_scenes") val topScenes: List<StatsScene> = emptyList(),
    @SerialName("top_performers") val topPerformers: List<StatsCount> = emptyList(),
    @SerialName("top_tags") val topTags: List<StatsCount> = emptyList(),
    val users: List<StatsUser> = emptyList(),
    @SerialName("recent_plays") val recentPlays: List<StatsPlay> = emptyList(),
    val generated: List<StatsFolder> = emptyList(),
    @SerialName("generated_total") val generatedTotal: Double = 0.0,
)
