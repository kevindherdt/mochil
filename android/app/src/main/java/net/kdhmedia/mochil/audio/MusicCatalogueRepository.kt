package net.kdhmedia.mochil.audio

import kotlinx.coroutines.CoroutineDispatcher
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.async
import kotlinx.coroutines.coroutineScope
import kotlinx.coroutines.withContext
import net.kdhmedia.mochil.auth.DeviceCredential
import net.kdhmedia.mochil.diagnostics.PerformanceDiagnostics

/** Blocking catalogue transport and measurement, kept outside UI state ownership. */
internal class MusicCatalogueRepository(
    private val credential: DeviceCredential,
    private val client: AudioClient,
    private val io: CoroutineDispatcher = Dispatchers.IO,
) {
    suspend fun groupTracks(libraryId: String, grouping: MusicGrouping, group: AudioGroup, artist: AudioGroup? = null): List<AudioTrack> = measured("load_audio_tracks") {
        client.groupTracks(credential, libraryId, grouping, group.name, artist?.name, group.id)
    }

    suspend fun artistTracks(libraryId: String, artist: AudioGroup): List<AudioTrack> = measured("load_audio_tracks") {
        client.groupTracks(credential, libraryId, MusicGrouping.Artists, artist.name, groupId = artist.id)
    }

    suspend fun artistAlbums(libraryId: String, artist: AudioGroup): List<AudioGroup> = measured("load_audio_groups") {
        client.artistAlbums(credential, libraryId, artist.name)
    }

    suspend fun favorites(): List<AudioTrack> = measured("load_audio_favorites") { client.favorites(credential) }

    suspend fun setFavorite(track: AudioTrack, favorite: Boolean) = measured("update_audio_favorite") {
        client.setFavorite(credential, track, favorite)
    }

    suspend fun discovery(libraryId: String): MusicDiscoveryResult = withContext(io) {
        coroutineScope {
            val albums = async { PerformanceDiagnostics.measure("network", "load_discovery_albums") { client.groups(credential, libraryId, MusicGrouping.Albums, 0, limit = DISCOVERY_GROUP_LIMIT).items } }
            val artists = async { PerformanceDiagnostics.measure("network", "load_discovery_artists") { client.groups(credential, libraryId, MusicGrouping.Artists, 0, limit = DISCOVERY_GROUP_LIMIT).items } }
            val recent = async { PerformanceDiagnostics.measure("network", "load_discovery_recent") { client.tracks(credential, libraryId, null, limit = DISCOVERY_TRACK_LIMIT, sort = "recent").items } }
            val played = async { PerformanceDiagnostics.measure("network", "load_discovery_played") { client.recentlyPlayed(credential, libraryId, DISCOVERY_TRACK_LIMIT) } }
            MusicDiscoveryResult(albums.await(), artists.await(), recent.await(), played.await())
        }
    }

    suspend fun recentlyPlayed(libraryId: String): List<AudioTrack> = measured("load_recently_played") {
        client.recentlyPlayed(credential, libraryId, RECENTLY_PLAYED_LIMIT)
    }

    suspend fun groups(libraryId: String, grouping: MusicGrouping, offset: Int, query: String): AudioGroupPage = measured("load_audio_groups") {
        client.groups(credential, libraryId, grouping, offset, query)
    }

    suspend fun tracks(libraryId: String, cursor: String?, query: String, sort: String, played: String, offset: Int): AudioPage = measured("load_audio_tracks") {
        client.tracks(credential, libraryId, cursor, query, sort = sort, played = played, offset = offset)
    }

    suspend fun navigation(libraryId: String, grouping: MusicGrouping, query: String): List<AudioNavigationBucket> =
        withContext(io) { client.navigation(credential, libraryId, grouping, query) }

    private suspend fun <T> measured(name: String, block: () -> T): T = withContext(io) {
        PerformanceDiagnostics.measure("network", name, block)
    }
}

internal const val DISCOVERY_GROUP_LIMIT = 10
internal const val DISCOVERY_TRACK_LIMIT = 8
internal const val RECENTLY_PLAYED_LIMIT = 20

internal data class MusicDiscoveryResult(
    val albums: List<AudioGroup>,
    val artists: List<AudioGroup>,
    val recent: List<AudioTrack>,
    val played: List<AudioTrack>,
)
