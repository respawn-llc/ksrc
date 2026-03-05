package resolve

import "testing"

func TestMatchModule(t *testing.T) {
	coord := Coord{Group: "org.jetbrains.kotlinx", Artifact: "kotlinx-datetime", Version: "0.7.1"}
	if !MatchModule("org.jetbrains.kotlinx:kotlinx-datetime", coord) {
		t.Fatal("expected exact module match")
	}
	if !MatchModule("kotlinx.datetime", coord) {
		t.Fatal("expected dot-normalized match")
	}
	if MatchModule("other.group", coord) {
		t.Fatal("expected no match")
	}
}

func TestMatchAny(t *testing.T) {
	if !MatchAny("org.*", "org.jetbrains.kotlinx") {
		t.Fatal("expected glob match")
	}
	if MatchAny("com.*", "org.jetbrains.kotlinx") {
		t.Fatal("expected no match")
	}
}
