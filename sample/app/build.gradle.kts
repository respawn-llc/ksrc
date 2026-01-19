plugins {
    alias(libs.plugins.android.application)
}

val androidCompileSdk = libs.versions.androidCompileSdk.get().toInt()
val androidMinSdk = libs.versions.androidMinSdk.get().toInt()
val androidTargetSdk = libs.versions.androidTargetSdk.get().toInt()

android {
    namespace = "com.example.ksrcsample.app"
    compileSdk = androidCompileSdk

    defaultConfig {
        applicationId = "com.example.ksrcsample.app"
        minSdk = androidMinSdk
        targetSdk = androidTargetSdk
        versionCode = 1
        versionName = "1.0"
    }
}

dependencies {
    implementation(project(":shared"))
}
