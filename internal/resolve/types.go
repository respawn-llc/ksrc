package resolve

import (
	"fmt"
	"strings"
)

type Coord struct {
	Group    string
	Artifact string
	Version  string
}

func (c Coord) String() string {
	if c.Version == "" {
		return c.Group + ":" + c.Artifact
	}
	return c.Group + ":" + c.Artifact + ":" + c.Version
}

func (c Coord) IsZero() bool {
	return c.Group == "" && c.Artifact == "" && c.Version == ""
}

func ParseCoord(s string) (Coord, error) {
	parts := strings.Split(s, ":")
	if len(parts) < 2 || len(parts) > 3 {
		return Coord{}, fmt.Errorf("invalid coord: %q", s)
	}
	c := Coord{Group: parts[0], Artifact: parts[1]}
	if len(parts) == 3 {
		c.Version = parts[2]
	}
	if c.Group == "" || c.Artifact == "" {
		return Coord{}, fmt.Errorf("invalid coord: %q", s)
	}
	return c, nil
}

type SourceJar struct {
	Coord      Coord
	Path       string
	SelectedBy []Coord
}
