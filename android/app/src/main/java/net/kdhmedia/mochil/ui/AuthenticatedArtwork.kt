package net.kdhmedia.mochil.ui

import android.content.Context
import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.MaterialTheme
import androidx.compose.runtime.Composable
import androidx.compose.runtime.setValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.getValue
import androidx.compose.runtime.remember
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.layout.ContentScale
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.unit.Dp
import androidx.compose.ui.unit.dp
import coil3.compose.AsyncImage
import coil3.network.NetworkHeaders
import coil3.network.httpHeaders
import coil3.request.ImageRequest
import coil3.size.Size

/**
 * The one request for a server artwork URL, headers included. Shared so a screen that draws the same file in two
 * places asks Coil for one bitmap instead of building a second request that decodes it again.
 */
internal fun authenticatedArtworkRequest(
    context: Context,
    url: String,
    authorizationHeader: String?,
    size: Size? = null,
): ImageRequest {
    val headers = NetworkHeaders.Builder().apply {
        if (authorizationHeader != null) set("Authorization", authorizationHeader)
    }.build()
    return ImageRequest.Builder(context)
        .data(url)
        .httpHeaders(headers)
        .apply { if (size != null) size(size) }
        .build()
}

/** Shared artwork loader for server media that may require device authorization. */
@Composable
fun AuthenticatedArtwork(
    url: String,
    authorizationHeader: String?,
    title: String,
    modifier: Modifier = Modifier,
    cornerRadius: Dp = 10.dp,
    contentScale: ContentScale = ContentScale.Crop,
) {
    val context = LocalContext.current
    val model = remember(url, authorizationHeader) {
        authenticatedArtworkRequest(context, url, authorizationHeader)
    }
    // The tile shows through while loading and when there is no artwork, so rows keep one shape. It shimmers only while an image is on its way.
    var loading by remember(model) { mutableStateOf(url.isNotBlank()) }
    Box(modifier.clip(RoundedCornerShape(cornerRadius)).then(if (loading) Modifier.shimmer() else Modifier.background(MaterialTheme.colorScheme.surfaceVariant))) {
        if (url.isNotBlank()) AsyncImage(
            model = model,
            contentDescription = "Artwork for $title",
            contentScale = contentScale,
            onState = { loading = it is coil3.compose.AsyncImagePainter.State.Loading || it is coil3.compose.AsyncImagePainter.State.Empty },
            modifier = Modifier.fillMaxSize(),
        )
    }
}
