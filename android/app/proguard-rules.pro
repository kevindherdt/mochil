# Shared shrinking rules for the direct updater and store-safe application.

# The avatar page calls the app through these methods by name.
-keepclassmembers class * {
    @android.webkit.JavascriptInterface <methods>;
}
