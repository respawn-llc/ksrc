# Issue 25 Follow-ups

## Centralize ksrc cache roots
- Current state: file-id cache, source extraction cache, and Gradle init-script cache each compute their own user-cache paths.
- Desired state: shared cache-root helper/package with stable subdirectories, env override policy documented per cache, and tests.
- Context: stable Gradle init script cache was added under user cache dir for Configuration Cache reuse. Keep behavior, but avoid growing separate cache-root implementations.

## KMP Variant Source Inclusion
- Status: implemented for Gradle builds that expose `ResolvedVariantResult.externalVariant`.
- Product behavior: `ksrc search <pattern> --module group:artifact` and `ksrc resolve --module group:artifact` include common/base sources plus Gradle-selected platform variant source jars, unless `--targets` or `--config` narrows the inspected configurations.
- Constraint kept: no artifact-name suffix heuristics/string matching for variant expansion.
- Design: Gradle module metadata `available-at` is surfaced through `ResolvedVariantResult.externalVariant`; ksrc records external variant sources with internal `selectedBy` metadata so base selectors keep those source jars after Go-side filtering.
- Search duplicate handling: if a selected variant source jar repeats the same matching line from the selected base source jar, search emits the base hit and suppresses the duplicate variant hit.

Sample proof (Gradle 9.4.0, Kotlin 2.3.10, AGP 9.1.0):
- `desktopMainCompileClasspath` root dependency selects `org.jetbrains.kotlinx:kotlinx-datetime:0.7.1` variant `jvmApiElements-published`.
- That selected base component has a direct resolved dependency edge to `org.jetbrains.kotlinx:kotlinx-datetime-jvm:0.7.1`.
- Edge details from `ResolvedDependencyResult` probe:
  - root edge: requested `org.jetbrains.kotlinx:kotlinx-datetime:0.7.1`, selected variant `jvmApiElements-published`, attrs include `org.jetbrains.kotlin.platform.type=jvm`, capability `org.jetbrains.kotlinx:kotlinx-datetime:0.7.1`.
  - base-to-platform edge: requested `org.jetbrains.kotlinx:kotlinx-datetime-jvm:0.7.1`, selected variant `jvmApiElements-published`, attrs include `org.jetbrains.kotlin.platform.type=jvm`, capability `org.jetbrains.kotlinx:kotlinx-datetime-jvm:0.7.1`, `selectionReason=requested`.
  - normal dependency edges from metadata config also have `selectionReason=requested`, so `selectionReason` alone cannot identify platform-variant edges.
- `artifactView { withVariantReselection(); category=documentation; docsType=sources }` on JVM configs returns `kotlinx-datetime-jvm-0.7.1-sources.jar`.
- Metadata/common configs select base component variant `metadataApiElements`; artifact view did not return common/base sources in this probe, while legacy detached `classifier: sources` does.
- Resolved distinction: use `ResolvedVariantResult.externalVariant`; do not infer from dependency edges, selection reasons, names, groups, or suffixes.

Covered regressions:
- Base selector returns common/base plus selected JVM variant sources in the sample KMP fixture.
- `--targets` narrows selected external variants using a lightweight local Gradle module-metadata fixture with distinct JVM and JS variants under Configuration Cache and Isolated Projects.
- Search suppresses duplicate base/variant matches without suppressing same-text matches from different source paths.
