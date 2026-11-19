package net.kdhmedia.mochil.auth

import java.net.URI

/**
 * Mochil usually runs on the machine that already holds the media: your own network, or reached over
 * Tailscale from outside. Both are plain http:// on a private address, which is the setup most people
 * have, so https is required only once the address is something the public internet can reach.
 *
 * The rule is deliberately about the *host*, not a setting: a public hostname over http is refused
 * with a message that says what to do, and a private address is allowed without ceremony.
 */
internal object ServerAddress {
    fun normalize(value: String): String {
        val trimmed = value.trim().trimEnd('/')
        val uri = runCatching { URI(trimmed) }.getOrNull()
            ?: throw LoginException(0, "Enter a valid Mochil address.")
        val host = uri.host
        if (host.isNullOrBlank() || uri.userInfo != null || uri.query != null || uri.fragment != null) {
            throw LoginException(0, "Enter a valid Mochil address, such as http://mochil.local:8099.")
        }
        when (uri.scheme.lowercase()) {
            "https" -> return uri.toString()
            "http" -> {
                if (isPrivateHost(host)) return uri.toString()
                throw LoginException(
                    0,
                    "Use https:// for a public address. http:// only works on your own network or over Tailscale.",
                )
            }
            else -> throw LoginException(0, "Start the address with http:// or https://.")
        }
    }

    /** Private addresses, mDNS/`.local` names, Tailscale's CGNAT range and ULA/link-local IPv6. */
    fun isPrivateHost(host: String): Boolean {
        val bare = host.removePrefix("[").removeSuffix("]").lowercase()
        if (bare.isEmpty()) return false
        ipv4(bare)?.let { return it }
        if (bare == "::1" || bare == "localhost" || bare.endsWith(".local") ||
            bare.endsWith(".lan") || bare.endsWith(".internal") || bare.endsWith(".home.arpa") ||
            bare.endsWith(".ts.net")
        ) {
            return true
        }
        if (bare.contains(':')) {
            // IPv6: fc00::/7 unique-local and fe80::/10 link-local.
            val head = bare.substringBefore(':')
            if (head.startsWith("fc") || head.startsWith("fd") || head.startsWith("fe8") ||
                head.startsWith("fe9") || head.startsWith("fea") || head.startsWith("feb")
            ) {
                return true
            }
            return false
        }
        // A single label with no dot is a LAN name such as "nixos".
        return !bare.contains('.')
    }

    private fun ipv4(host: String): Boolean? {
        val parts = host.split('.')
        if (parts.size != 4) return null
        val octets = parts.map { it.toIntOrNull() ?: return null }
        if (octets.any { it < 0 || it > 255 }) return null
        val (a, b) = octets
        return when {
            a == 10 -> true
            a == 127 -> true
            a == 192 && b == 168 -> true
            a == 172 && b in 16..31 -> true
            a == 169 && b == 254 -> true
            a == 100 && b in 64..127 -> true // Tailscale and carrier-grade NAT
            else -> false
        }
    }
}
