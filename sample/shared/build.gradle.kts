plugins {
    alias(libs.plugins.kotlin.multiplatform)
    alias(libs.plugins.android.kotlin.multiplatform.library)
}

val androidCompileSdk = libs.versions.androidCompileSdk.get().toInt()
val androidCompileSdkMinor = libs.versions.androidCompileSdkMinor.get().toInt()
val androidMinSdk = libs.versions.androidMinSdk.get().toInt()

kotlin {
    android {
        namespace = "com.example.ksrcsample"
        compileSdk {
            version = release(androidCompileSdk) {
                minorApiLevel = androidCompileSdkMinor
            }
        }
        minSdk = androidMinSdk
    }
    jvm("desktop")

    sourceSets {
        commonMain.dependencies {
            implementation(libs.kotlinx.datetime)
        }
        commonTest.dependencies {
            implementation(kotlin("test"))
        }
    }
}
