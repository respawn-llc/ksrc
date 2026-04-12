package resolve

import "testing"

func TestCompareVersion(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		a    string
		b    string
		want int
	}{
		{name: "numeric ordering", a: "1.0.1", b: "1.0.0", want: 1},
		{name: "trailing zero equivalent", a: "1.2", b: "1.2.0", want: 0},
		{name: "alpha before beta", a: "1.0-alpha1", b: "1.0-beta1", want: -1},
		{name: "milestone before rc", a: "1.0-m1", b: "1.0-rc1", want: -1},
		{name: "rc before snapshot", a: "1.0-rc1", b: "1.0-snapshot", want: -1},
		{name: "snapshot before release", a: "1.0-snapshot", b: "1.0", want: -1},
		{name: "underscore separator keeps rc before release", a: "1.0_rc1", b: "1.0", want: -1},
		{name: "underscore separator matches dot numeric segment", a: "1.0_1", b: "1.0.1", want: 0},
		{name: "plus qualifier stays below numeric segment", a: "1.0+1", b: "1.0.1", want: -1},
		{name: "service pack after release", a: "1.0-sp", b: "1.0", want: 1},
		{name: "release alias ga", a: "1.0-ga", b: "1.0", want: 0},
		{name: "release alias final", a: "1.0-final", b: "1.0", want: 0},
		{name: "release alias release", a: "1.0-release", b: "1.0", want: 0},
		{name: "rc alias cr", a: "1.0-cr1", b: "1.0-rc1", want: 0},
		{name: "unknown qualifier after release", a: "1.0-foo", b: "1.0", want: 1},
		{name: "unknown qualifier lexical ordering", a: "1.0-foo", b: "1.0-zzz", want: -1},
		{name: "single letter alpha alias", a: "1.0-a1", b: "1.0-alpha1", want: 0},
		{name: "single letter beta alias", a: "1.0-b2", b: "1.0-beta2", want: 0},
		{name: "single letter milestone alias", a: "1.0-m3", b: "1.0-milestone3", want: 0},
		{name: "dot qualifier before hyphen qualifier", a: "1.0.RC2", b: "1.0-RC3", want: -1},
		{name: "dot unknown qualifier before hyphen unknown qualifier", a: "1.0.X1", b: "1.0-X2", want: -1},
		{name: "hyphen qualifier before next numeric", a: "1.0-RC3", b: "1.0.1", want: -1},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			if got := compareVersionSign(CompareVersion(testCase.a, testCase.b)); got != testCase.want {
				t.Fatalf("CompareVersion(%q, %q) = %d, want %d", testCase.a, testCase.b, got, testCase.want)
			}
		})
	}
}

func compareVersionSign(value int) int {
	switch {
	case value < 0:
		return -1
	case value > 0:
		return 1
	default:
		return 0
	}
}
