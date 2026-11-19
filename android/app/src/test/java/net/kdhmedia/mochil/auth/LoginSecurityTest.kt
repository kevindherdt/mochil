package net.kdhmedia.mochil.auth

import org.json.JSONObject
import org.junit.Assert.assertEquals
import org.junit.Assert.assertThrows
import org.junit.Test

class LoginSecurityTest {
    @Test fun serverCannotRelocateCredentialAwayFromApprovedOrigin() {
        val credential = credentialFromServerResponse(
            "https://media.example",
            JSONObject("""{"server_url":"https://attacker.example","device_id":"phone","device_token":"secret"}"""),
        )

        assertEquals("https://media.example", credential.serverUrl)
        assertEquals("phone", credential.deviceId)
        assertEquals("secret", credential.token)
    }

    @Test fun malformedCredentialsAreRejectedBeforePersistence() {
        assertThrows(LoginException::class.java) {
            credentialFromServerResponse("https://media.example", JSONObject("""{"device_id":"","device_token":"secret"}"""))
        }
        assertThrows(LoginException::class.java) {
            credentialFromServerResponse("https://media.example", JSONObject("""{"device_id":"phone","device_token":""}"""))
        }
    }
}
