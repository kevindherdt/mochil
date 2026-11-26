import org.gradle.api.DefaultTask
import org.gradle.api.provider.Property
import org.gradle.api.tasks.Input
import org.gradle.api.tasks.TaskAction
import java.io.File

plugins {
    id("com.android.application")
    id("org.jetbrains.kotlin.plugin.compose")
}

abstract class VerifyReleaseSigning : DefaultTask() {
    @get:Input abstract val ready: Property<Boolean>
    @get:Input abstract val storePath: Property<String>

    @TaskAction
    fun verify() {
        check(ready.get()) {
            "Release signing requires MOCHIL_KEYSTORE_PATH, MOCHIL_KEYSTORE_PASSWORD, MOCHIL_KEY_ALIAS, and MOCHIL_KEY_PASSWORD."
        }
        check(File(storePath.get()).isFile) { "MOCHIL_KEYSTORE_PATH does not point to a regular file." }
    }
}

abstract class VerifyOptimizedSigning : DefaultTask() {
    @get:Input abstract val storePath: Property<String>

    @TaskAction
    fun verify() {
        check(File(storePath.get()).isFile) {
            "Optimized updates require the existing persistent signing key. Restore android/.signing/optimized.keystore or set MOCHIL_OPTIMIZED_KEYSTORE_PATH. A new debug key cannot update existing installations."
        }
    }
}

val releaseStoreFile = providers.environmentVariable("MOCHIL_KEYSTORE_PATH").orNull
val releaseStorePassword = providers.environmentVariable("MOCHIL_KEYSTORE_PASSWORD").orNull
val releaseKeyAlias = providers.environmentVariable("MOCHIL_KEY_ALIAS").orNull
val releaseKeyPassword = providers.environmentVariable("MOCHIL_KEY_PASSWORD").orNull
// The release number, read once. The code rises with the name and clears the whole hard-coded past:
// releases up to 0.1.61 were all shipped as 444, so anything at or below that cannot install over an
// existing app. 0.1.61 -> 16100, 0.2.0 -> 20000.
val mochilVersion: String = rootProject.file("../VERSION").let { file ->
    require(file.isFile) { "VERSION is missing at ${file.absolutePath}" }
    file.readText().trim().ifEmpty { error("VERSION is empty") }
}
// A preview build for trying a change on a phone before it is a release: -PmochilPreview=N (1..99) adds N to the code
// and "-previewN" to the name. The code has two spare digits (a release is (M*10000+m*100+p)*100), so a preview installs
// over everything older and the next release, which must come from a higher VERSION, installs over it. A preview can
// never be published: the release workflow refuses an APK whose versionName is not VERSION.
val mochilPreview: Int = providers.gradleProperty("mochilPreview").orNull?.let { raw ->
    raw.toIntOrNull()?.takeIf { it in 1..99 } ?: error("-PmochilPreview must be 1..99, got '$raw'")
} ?: 0
val mochilVersionCode: Int = mochilVersion.split(".").let { parts ->
    require(parts.size == 3) { "VERSION must be MAJOR.MINOR.PATCH, got '$mochilVersion'" }
    val numbers = parts.map { part -> part.toIntOrNull() ?: error("VERSION part '$part' is not a number") }
    require(numbers.all { it in 0..99 }) { "VERSION parts must be 0-99, got '$mochilVersion'" }
    (numbers[0] * 10000 + numbers[1] * 100 + numbers[2]) * 100 + mochilPreview
}

val releaseSigningReady = listOf(releaseStoreFile, releaseStorePassword, releaseKeyAlias, releaseKeyPassword).all { !it.isNullOrBlank() }
val optimizedStoreFile = providers.environmentVariable("MOCHIL_OPTIMIZED_KEYSTORE_PATH").orNull
    ?.let(::file)
    ?: rootProject.file(".signing/optimized.keystore")
val optimizedSigningReady = optimizedStoreFile.isFile

android {
    namespace = "net.kdhmedia.mochil"
    compileSdk = 37

    defaultConfig {
        applicationId = "net.kdhmedia.mochil"
        minSdk = 26
        targetSdk = 37
        // Both come from the repository's VERSION file. They used to be literals, and stayed at
        // 0.1.20 / 444 for every release after it: the app then told every phone it was already up to
        // date, the installer refused to replace a package with an equal versionCode, and a palette
        // that had been removed on the server lived on in whatever APK people had. A version that can
        // disagree with the release is a version that will.
        versionName = mochilVersion
        versionCode = mochilVersionCode
        if (mochilPreview > 0) versionNameSuffix = "-preview$mochilPreview"
    }
    flavorDimensions += "distribution"
    productFlavors {
        create("direct") {
            dimension = "distribution"
            buildConfigField("boolean", "SELF_UPDATE_ENABLED", "true")
            // The published release asset, under the constant name so the URL survives releases.
            // Used only when the paired server offers an update but the file it serves is missing or
            // older than what it advertises -- a self-hosted copy of the APK is the one part of an
            // update that goes stale without anyone noticing. The downloaded package is validated
            // (package name, version, signing key) before the installer sees it either way.
            buildConfigField(
                "String",
                "RELEASE_APK_URL",
                "\"https://github.com/kevindherdt/mochil/releases/latest/download/mochil-android.apk\"",
            )
        }
        create("store") {
            dimension = "distribution"
            buildConfigField("boolean", "SELF_UPDATE_ENABLED", "false")
        }
    }

    buildFeatures {
        compose = true
        buildConfig = true
    }
    testOptions.unitTests.isIncludeAndroidResources = true
    // The JVM ignores TMPDIR, so Robolectric's native runtime (fonts, ICU, ~160 MB per run) and every
    // test's temp dir land in java.io.tmpdir — /tmp, which is a tmpfs on the build box, i.e. host RAM.
    // scripts/android.sh passes -PtestTmpDir=<disk path> to keep them off it.
    providers.gradleProperty("testTmpDir").orNull?.let { tmp ->
        tasks.withType<Test>().configureEach {
            jvmArgs("-Djava.io.tmpdir=$tmp")
        }
    }
    // Screenshot goldens (src/test/screenshots) verify on a normal run; -PrecordGoldens=true rewrites them.
    if (providers.gradleProperty("recordGoldens").orNull == "true") {
        tasks.withType<Test>().configureEach {
            systemProperty("mochil.goldens", "record")
        }
    }
    sourceSets.getByName("test").resources.directories.add(rootProject.file("../contracts").path)
    compileOptions {
        isCoreLibraryDesugaringEnabled = true
        sourceCompatibility = JavaVersion.VERSION_17
        targetCompatibility = JavaVersion.VERSION_17
    }
    packaging.resources.excludes += "/META-INF/{AL2.0,LGPL2.1}"

    signingConfigs {
        if (optimizedSigningReady) create("optimizedPersistent") {
            storeFile = optimizedStoreFile
            // The alias and password were the debug keystore's, spelled out here, so pointing
            // MOCHIL_OPTIMIZED_KEYSTORE_PATH at a real release key failed with "keystore password was
            // incorrect" no matter what the environment said. They follow the environment now and
            // fall back to the old values so an existing debug-alias keystore still builds.
            storePassword = releaseStorePassword ?: "android"
            keyAlias = releaseKeyAlias ?: "androiddebugkey"
            keyPassword = releaseKeyPassword ?: "android"
        }
        if (releaseSigningReady) create("release") {
            storeFile = file(requireNotNull(releaseStoreFile))
            storePassword = releaseStorePassword
            keyAlias = releaseKeyAlias
            keyPassword = releaseKeyPassword
        }
    }

    buildTypes {
        release {
            isMinifyEnabled = true
            isShrinkResources = true
            signingConfig = signingConfigs.findByName("release")
            proguardFiles(getDefaultProguardFile("proguard-android-optimize.txt"), "proguard-rules.pro")
        }
        create("optimized") {
            initWith(getByName("release"))
            signingConfig = signingConfigs.findByName("optimizedPersistent")
            matchingFallbacks += listOf("release")
        }
    }
}

val verifyReleaseSigning by tasks.registering(VerifyReleaseSigning::class) {
    group = "verification"
    description = "Fails release packaging unless all Mochil signing environment variables are present."
    ready.set(releaseSigningReady)
    storePath.set(releaseStoreFile.orEmpty())
}

tasks.matching {
    (it.name.startsWith("assemble") || it.name.startsWith("bundle")) && it.name.endsWith("Release")
}.configureEach { dependsOn(verifyReleaseSigning) }

val verifyOptimizedSigning by tasks.registering(VerifyOptimizedSigning::class) {
    group = "verification"
    description = "Prevents private update packages from silently changing signing identity."
    storePath.set(optimizedStoreFile.absolutePath)
}

tasks.matching {
    (it.name.startsWith("assemble") || it.name.startsWith("bundle") || it.name.startsWith("package")) && it.name.endsWith("Optimized")
}.configureEach { dependsOn(verifyOptimizedSigning) }

dependencies {
    val composeBom = platform("androidx.compose:compose-bom:2026.09.00")
    implementation(composeBom)
    androidTestImplementation(composeBom)

    implementation("androidx.activity:activity-compose:1.13.0")
    implementation("androidx.compose.material3:material3")
    implementation("androidx.compose.material:material-icons-core")
    implementation("androidx.compose.material:material-icons-extended")
    implementation("androidx.compose.ui:ui")
    implementation("androidx.compose.ui:ui-tooling-preview")
    implementation("androidx.lifecycle:lifecycle-runtime-compose:2.11.0")
    implementation("androidx.lifecycle:lifecycle-viewmodel-compose:2.11.0")
    implementation("androidx.fragment:fragment-ktx:1.9.1")
    implementation("androidx.media3:media3-exoplayer:1.11.1")
    implementation("androidx.media3:media3-exoplayer-hls:1.11.1")
    implementation("androidx.media3:media3-session:1.11.1")
    implementation("androidx.media3:media3-ui:1.11.1")
    implementation("io.coil-kt.coil3:coil-compose:3.6.3")
    implementation("io.coil-kt.coil3:coil-network-okhttp:3.6.3")
    implementation("com.squareup.okhttp3:okhttp:5.5.1")
    implementation("org.jetbrains.kotlinx:kotlinx-serialization-json:1.12.0")
    implementation("org.readium.kotlin-toolkit:readium-shared:3.4.0")
    implementation("org.readium.kotlin-toolkit:readium-streamer:3.4.0")
    implementation("org.readium.kotlin-toolkit:readium-navigator:3.4.0")
    coreLibraryDesugaring("com.android.tools:desugar_jdk_libs:2.1.5")

    testImplementation(composeBom)
    testImplementation("androidx.compose.ui:ui-test-junit4")
    testImplementation("junit:junit:4.13.2")
    testImplementation("org.robolectric:robolectric:4.15.1")
    testImplementation("org.json:json:20250517")
    androidTestImplementation("androidx.compose.ui:ui-test-junit4")
    debugImplementation("androidx.compose.ui:ui-tooling")
    debugImplementation("androidx.compose.ui:ui-test-manifest")
}
