package main

import "testing"

func TestShouldHideOnClose(t *testing.T) {
	t.Parallel()
	cases := []struct {
		closeToTray bool
		forceQuit   bool
		want        bool
	}{
		{true, false, true},
		{true, true, false},
		{false, false, false},
		{false, true, false},
	}
	for _, tc := range cases {
		if got := ShouldHideOnClose(tc.closeToTray, tc.forceQuit); got != tc.want {
			t.Fatalf("closeToTray=%v forceQuit=%v got %v want %v", tc.closeToTray, tc.forceQuit, got, tc.want)
		}
	}
}
