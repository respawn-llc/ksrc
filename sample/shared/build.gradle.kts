plugins {
    alias(libs.plugins.kotlin.multiplatform)
    alias(libs.plugins.android.kotlin.multiplatform.library)
}

val androidCompileSdk = libs.versions.androidCompileSdk.get().toInt()
val androidMinSdk = libs.versions.androidMinSdk.get().toInt()

kotlin {
    android {
        namespace = "com.example.ksrcsample"
        compileSdk = androidCompileSdk
        minSdk = androidMinSdk
    }
    jvm("desktop")

    sourceSets {
        val commonMain by getting {
            dependencies {
                implementation(libs.kotlinx.datetime)
            }
        }
        val commonTest by getting {
            dependencies {
                implementation(kotlin("test"))
            }
        }
    }
}
