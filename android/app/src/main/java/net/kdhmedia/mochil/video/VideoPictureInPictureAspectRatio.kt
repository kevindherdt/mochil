package net.kdhmedia.mochil.video

/**
 * The width/height aspect ratio to request for Picture-in-Picture, falling back to 16:9 when
 * either dimension is unknown (<= 0) and clamped to the range PictureInPictureParams accepts.
 */
internal fun videoPictureInPictureAspectRatio(width: Int, height: Int): Double {
    val w = width.takeIf { it > 0 } ?: 16
    val h = height.takeIf { it > 0 } ?: 9
    return (w.toDouble() / h).coerceIn(1.0 / 2.39, 2.39)
}
