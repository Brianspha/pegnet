package grader

import (
	"encoding/binary"
	"sort"

	"github.com/pegnet/pegnet/modules/lxr30"
)

// baseGrader provides common functionality that is deemed useful in all versions
type baseGrader struct {
	oprs    []*GradingOPR
	winners []*GradingOPR // this can be an empty slice if there are no winners
	graded  bool

	height int32

	prevWinners []string
}

// NewGrader instantiates a Block Grader for a specific version.
// Once set, the height and list of previous winners can't be changed.
func NewGrader(version int, height int32, previousWinners []string) Block {
	switch version {
	case 1:
		v1 := new(V1Block)
		v1.height = height
		v1.prevWinners = previousWinners
		return v1
	case 2:
		v2 := new(V2Block)
		v2.height = height
		v2.prevWinners = previousWinners
		return v2
	default:
		// most likely developer error or outdated package
		panic("invalid grader version")
	}
}

// Count will return the total number of OPRs stored in the block.
// If the set has been graded, this number may be less than the amount of OPRs added
// due to duplicate filter and self reported difficulty checks
func (bg *baseGrader) Count() int {
	return len(bg.oprs)
}

// GetPreviousWinners returns the set of previous winners
func (bg *baseGrader) GetPreviousWinners() []string {
	return bg.prevWinners
}

// Height returns the height the block grader is set to
func (bg *baseGrader) Height() int32 {
	return bg.height
}

// filter out duplicate gradingOPRs. an OPR is a duplicate when both
// nonce and oprhash are the same
func (bg *baseGrader) filterDuplicates() {
	filtered := make([]*GradingOPR, 0)

	added := make(map[string]bool)
	for _, v := range bg.oprs {
		id := string(append(v.Nonce, v.OPRHash...))
		if !added[id] {
			filtered = append(filtered, v)
			added[id] = true
		}
	}

	bg.oprs = filtered
}

// sortByDifficulty uses an efficient algorithm based on self-reported difficulty
// to avoid having to LXRhash the entire set.
// calculates at most `limit + misreported difficulties` hashes
func (bg *baseGrader) sortByDifficulty(limit int) {
	sort.SliceStable(bg.oprs, func(i, j int) bool {
		return bg.oprs[i].SelfReportedDifficulty > bg.oprs[i].SelfReportedDifficulty
	})

	lx := lxr30.Init()

	topX := make([]*GradingOPR, 0)
	for _, o := range bg.oprs {
		hash := lx.Hash(append(o.OPRHash, o.Nonce...))
		diff := binary.BigEndian.Uint64(hash)

		if diff != o.SelfReportedDifficulty {
			continue
		}

		topX = append(topX, o)

		if len(topX) >= limit {
			break
		}
	}

	bg.oprs = topX
}