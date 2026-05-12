import nl.littlerobots.vcu.plugin.versionCatalogUpdate

plugins {
    alias(libs.plugins.version.catalog.update)
}

versionCatalogUpdate {
    sortByKey = true

    keep {
        keepUnusedVersions = true
    }
}
