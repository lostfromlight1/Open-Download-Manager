package engine

import (
	// fmt builds overlap / planner error messages.
	"fmt"
)

// ByteRange is an inclusive byte interval [Start, End].
//
// Inclusive ends match HTTP Content-Range ("bytes 0-99/100" covers 100 bytes).
type ByteRange struct {
	Start int64
	End   int64
}

// Length returns how many bytes this range covers.
func (r ByteRange) Length() int64 {
	if r.End < r.Start {
		return 0
	}
	return r.End - r.Start + 1
}

// PlanSegments splits a known-size file into up to `connections` non-overlapping ranges.
//
// Rules:
//   - connections <= 1 or total too small → one segment covering the whole file
//   - segments never overlap and leave no gaps
//   - each segment is at least minBytes when possible (last segment may be larger)
//
// Learning note: we plan from the TOTAL size (from a verified Content-Range),
// not from Accept-Ranges or Content-Length alone.
func PlanSegments(total int64, connections int, minBytes int64) ([]ByteRange, error) {
	if total < 0 {
		return nil, fmt.Errorf("total size must be >= 0")
	}
	if total == 0 {
		return nil, nil // empty file: no segments needed
	}
	if connections <= 1 {
		return []ByteRange{{Start: 0, End: total - 1}}, nil
	}
	if minBytes <= 0 {
		minBytes = 1
	}

	// Cap worker count by how many minBytes chunks fit in the file.
	maxBySize := int(total / minBytes)
	if maxBySize < 1 {
		maxBySize = 1
	}
	n := connections
	if n > maxBySize {
		n = maxBySize
	}
	if n < 1 {
		n = 1
	}

	segs := make([]ByteRange, 0, n)
	// Even split with remainder bytes tacked onto the final segment.
	base := total / int64(n)
	rem := total % int64(n)

	var offset int64
	for i := 0; i < n; i++ {
		length := base
		if i == n-1 {
			length += rem
		}
		if length <= 0 {
			continue
		}
		segs = append(segs, ByteRange{
			Start: offset,
			End:   offset + length - 1,
		})
		offset += length
	}

	if err := validateSegments(segs, total); err != nil {
		return nil, err
	}
	return segs, nil
}

// validateSegments checks coverage, ordering, and non-overlap for a full-file plan.
func validateSegments(segs []ByteRange, total int64) error {
	if len(segs) == 0 {
		if total == 0 {
			return nil
		}
		return fmt.Errorf("no segments for total=%d", total)
	}
	if segs[0].Start != 0 {
		return fmt.Errorf("first segment must start at 0, got %d", segs[0].Start)
	}
	var covered int64
	var prevEnd int64 = -1
	for i, s := range segs {
		if s.End < s.Start {
			return fmt.Errorf("segment %d has end < start", i)
		}
		if s.Start <= prevEnd {
			return fmt.Errorf("%w: segment %d starts at %d but previous ended at %d", ErrSegmentOverlap, i, s.Start, prevEnd)
		}
		if s.Start != prevEnd+1 {
			return fmt.Errorf("gap before segment %d: previous ended %d, next starts %d", i, prevEnd, s.Start)
		}
		covered += s.Length()
		prevEnd = s.End
	}
	if prevEnd != total-1 || covered != total {
		return fmt.Errorf("segments cover %d..%d (%d bytes), want 0..%d (%d bytes)", segs[0].Start, prevEnd, covered, total-1, total)
	}
	return nil
}
