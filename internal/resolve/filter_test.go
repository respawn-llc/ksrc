package resolve

import (
	"reflect"
	"testing"
)

func TestMatchAny(t *testing.T) {
	t.Parallel()

	if !MatchAny("org.*", "org.jetbrains.kotlinx") {
		t.Fatal("expected glob match")
	}
	if !MatchAny("KOTLINX-DATETIME", "kotlinx-datetime") {
		t.Fatal("expected case-insensitive exact match")
	}
	if MatchAny("com.*", "org.jetbrains.kotlinx") {
		t.Fatal("expected no match")
	}
}

func TestMatchModule(t *testing.T) {
	t.Parallel()

	coord := Coord{Group: "org.jetbrains.kotlinx", Artifact: "kotlinx-datetime", Version: "0.7.1"}

	cases := []struct {
		name     string
		selector string
		want     bool
	}{
		{name: "exact", selector: "org.jetbrains.kotlinx:kotlinx-datetime", want: true},
		{name: "loose normalized", selector: "kotlinx.datetime", want: true},
		{name: "qualified trailing colon", selector: "org.jetbrains.kotlinx:kotlinx-datetime:", want: true},
		{name: "qualified version", selector: "org.jetbrains.kotlinx:kotlinx-datetime:0.7.1", want: true},
		{name: "miss", selector: "other.group", want: false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := MatchModule(tc.selector, coord)
			if got != tc.want {
				t.Fatalf("MatchModule(%q) = %v, want %v", tc.selector, got, tc.want)
			}
		})
	}
}

func TestSelectorToCoord(t *testing.T) {
	t.Parallel()

	coord, ok := SelectorToCoord("org.jetbrains.kotlinx:kotlinx-datetime:0.7.1", "", "", "")
	if !ok {
		t.Fatal("expected module selector coord")
	}
	if coord != (Coord{Group: "org.jetbrains.kotlinx", Artifact: "kotlinx-datetime", Version: "0.7.1"}) {
		t.Fatalf("unexpected coord: %#v", coord)
	}

	coord, ok = SelectorToCoord("org.jetbrains.kotlinx:kotlinx-datetime:0.7.1", "override.group", "override-artifact", "1.2.3")
	if !ok {
		t.Fatal("expected explicit group/artifact coord")
	}
	if coord != (Coord{Group: "override.group", Artifact: "override-artifact", Version: "1.2.3"}) {
		t.Fatalf("unexpected override coord: %#v", coord)
	}
}

func TestFilterSourcesAndCoordsShareSelectorSemantics(t *testing.T) {
	t.Parallel()

	sources := []SourceJar{
		{Coord: Coord{Group: "org.jetbrains.kotlinx", Artifact: "kotlinx-datetime", Version: "0.7.1"}, Path: "/tmp/datetime.jar"},
		{Coord: Coord{Group: "org.jetbrains.kotlinx", Artifact: "kotlinx-coroutines-core", Version: "1.10.2"}, Path: "/tmp/coroutines.jar"},
		{Coord: Coord{Group: "com.squareup.okio", Artifact: "okio", Version: "3.10.2"}, Path: "/tmp/okio.jar"},
	}
	coords := []Coord{sources[0].Coord, sources[1].Coord, sources[2].Coord}

	filteredSources := FilterSources(sources, "org.jetbrains.kotlinx:*datetime*:", "org.jetbrains.*", "*datetime", "0.*")
	filteredCoords := FilterCoords(coords, "org.jetbrains.kotlinx:*datetime*:", "org.jetbrains.*", "*datetime", "0.*")

	wantCoords := []Coord{{Group: "org.jetbrains.kotlinx", Artifact: "kotlinx-datetime", Version: "0.7.1"}}
	if !reflect.DeepEqual(filteredCoords, wantCoords) {
		t.Fatalf("unexpected coords: %#v", filteredCoords)
	}
	if len(filteredSources) != 1 || filteredSources[0].Coord != wantCoords[0] {
		t.Fatalf("unexpected sources: %#v", filteredSources)
	}
	if filteredSources[0].Path != "/tmp/datetime.jar" {
		t.Fatalf("unexpected source path: %q", filteredSources[0].Path)
	}
}

func TestFilterSourcesKeepsExternalVariantsSelectedByMatchingCoord(t *testing.T) {
	t.Parallel()

	sources := []SourceJar{
		{Coord: Coord{Group: "org.jetbrains.kotlinx", Artifact: "kotlinx-datetime", Version: "0.7.1"}, Path: "/tmp/datetime.jar"},
		{
			Coord: Coord{Group: "org.jetbrains.kotlinx", Artifact: "kotlinx-datetime-jvm", Version: "0.7.1"},
			Path:  "/tmp/datetime-jvm.jar",
			SelectedBy: []Coord{
				{Group: "org.jetbrains.kotlinx", Artifact: "kotlinx-datetime", Version: "0.7.1"},
			},
		},
		{Coord: Coord{Group: "org.jetbrains.kotlinx", Artifact: "kotlinx-coroutines-core", Version: "1.10.2"}, Path: "/tmp/coroutines.jar"},
	}

	filtered := FilterSources(sources, "org.jetbrains.kotlinx:kotlinx-datetime", "", "", "")
	if len(filtered) != 2 {
		t.Fatalf("expected base and selected external variant sources, got %#v", filtered)
	}
	if filtered[0].Path != "/tmp/datetime.jar" || filtered[1].Path != "/tmp/datetime-jvm.jar" {
		t.Fatalf("unexpected filtered sources: %#v", filtered)
	}
}
