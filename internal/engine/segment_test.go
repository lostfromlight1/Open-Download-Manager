package engine

import (
	"testing"
)

func TestPlanSegmentsSplitsEvenly(t *testing.T) {
	segs, err := PlanSegments(1000, 4, 100)
	if err != nil {
		t.Fatal(err)
	}
	if len(segs) != 4 {
		t.Fatalf("got %d segments, want 4", len(segs))
	}
	if segs[0].Start != 0 || segs[len(segs)-1].End != 999 {
		t.Fatalf("bad coverage: %#v", segs)
	}
	var sum int64
	for i := 1; i < len(segs); i++ {
		if segs[i].Start != segs[i-1].End+1 {
			t.Fatalf("gap/overlap at %d: %#v", i, segs)
		}
	}
	for _, s := range segs {
		sum += s.Length()
	}
	if sum != 1000 {
		t.Fatalf("covered %d, want 1000", sum)
	}
}

func TestPlanSegmentsFallsBackWhenTiny(t *testing.T) {
	segs, err := PlanSegments(500, 8, 256)
	if err != nil {
		t.Fatal(err)
	}
	// 500/256 = 1 → single segment
	if len(segs) != 1 || segs[0].Start != 0 || segs[0].End != 499 {
		t.Fatalf("expected single segment, got %#v", segs)
	}
}

func TestPlanSegmentsConnectionsOne(t *testing.T) {
	segs, err := PlanSegments(999, 1, 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(segs) != 1 {
		t.Fatalf("got %#v", segs)
	}
}

func TestParseContentRange(t *testing.T) {
	start, end, total, err := parseContentRange("bytes 10-19/100")
	if err != nil {
		t.Fatal(err)
	}
	if start != 10 || end != 19 || total != 100 {
		t.Fatalf("got %d-%d/%d", start, end, total)
	}

	_, _, _, err = parseContentRange("bytes abc-def/xyz")
	if err == nil {
		t.Fatal("expected error for malformed Content-Range")
	}
}
