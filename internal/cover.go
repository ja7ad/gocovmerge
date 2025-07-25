package internal

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"sort"
	"strconv"
	"strings"
)

// Profile represents the profiling data for a specific file.
type Profile struct {
	FileName string
	Mode     string
	Blocks   []ProfileBlock
}

// ProfileBlock represents a single block of profiling data.
type ProfileBlock struct {
	StartLine, StartCol int
	EndLine, EndCol     int
	NumStmt, Count      int
}

type byFileName []*Profile

func (p byFileName) Len() int           { return len(p) }
func (p byFileName) Less(i, j int) bool { return p[i].FileName < p[j].FileName }
func (p byFileName) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }

// ParseProfiles parses profile data in the specified file and returns a
// Profile for each source file described therein.
func ParseProfiles(fileName string) ([]*Profile, error) {
	pf, err := os.Open(fileName)
	if err != nil {
		return nil, err
	}
	defer pf.Close()
	return ParseProfilesFromReader(pf)
}

// ParseProfilesFromReader parses profile data from the Reader and
// returns a Profile for each source file described therein.
func ParseProfilesFromReader(rd io.Reader) ([]*Profile, error) {
	// First line is "mode: foo", where foo is "set", "count", or "atomic".
	// Rest of file is in the format
	//	encoding/base64/base64.go:34.44,37.40 3 1
	// where the fields are: name.go:line.column,line.column numberOfStatements count
	files := make(map[string]*Profile)
	s := bufio.NewScanner(rd)
	mode := ""
	for s.Scan() {
		line := s.Text()
		if mode == "" {
			const p = "mode: "
			if !strings.HasPrefix(line, p) || line == p {
				return nil, fmt.Errorf("bad mode line: %v", line)
			}
			mode = line[len(p):]
			continue
		}
		fn, b, err := parseLine(line)
		if err != nil {
			return nil, fmt.Errorf("line %q doesn't match expected format: %v", line, err)
		}
		p := files[fn]
		if p == nil {
			p = &Profile{
				FileName: fn,
				Mode:     mode,
			}
			files[fn] = p
		}
		p.Blocks = append(p.Blocks, b)
	}
	if err := s.Err(); err != nil {
		return nil, err
	}
	for _, p := range files {
		sort.Sort(blocksByStart(p.Blocks))
		// Merge samples from the same location.
		j := 1
		for i := 1; i < len(p.Blocks); i++ {
			b := p.Blocks[i]
			last := p.Blocks[j-1]
			if b.StartLine == last.StartLine &&
				b.StartCol == last.StartCol &&
				b.EndLine == last.EndLine &&
				b.EndCol == last.EndCol {
				if b.NumStmt != last.NumStmt {
					return nil, fmt.Errorf("inconsistent NumStmt: changed from %d to %d", last.NumStmt, b.NumStmt)
				}
				if mode == "set" {
					p.Blocks[j-1].Count |= b.Count
				} else {
					p.Blocks[j-1].Count += b.Count
				}
				continue
			}
			p.Blocks[j] = b
			j++
		}
		p.Blocks = p.Blocks[:j]
	}
	// Generate a sorted slice.
	profiles := make([]*Profile, 0, len(files))
	for _, profile := range files {
		profiles = append(profiles, profile)
	}
	sort.Sort(byFileName(profiles))
	return profiles, nil
}

// parseLine parses a line from a coverage file.
// It is equivalent to the regex
// ^(.+):([0-9]+)\.([0-9]+),([0-9]+)\.([0-9]+) ([0-9]+) ([0-9]+)$
//
// However, it is much faster: https://golang.org/cl/179377
func parseLine(l string) (fileName string, block ProfileBlock, err error) {
	end := len(l)

	b := ProfileBlock{}
	b.Count, end, err = seekBack(l, ' ', end, "Count")
	if err != nil {
		return "", b, err
	}
	b.NumStmt, end, err = seekBack(l, ' ', end, "NumStmt")
	if err != nil {
		return "", b, err
	}
	b.EndCol, end, err = seekBack(l, '.', end, "EndCol")
	if err != nil {
		return "", b, err
	}
	b.EndLine, end, err = seekBack(l, ',', end, "EndLine")
	if err != nil {
		return "", b, err
	}
	b.StartCol, end, err = seekBack(l, '.', end, "StartCol")
	if err != nil {
		return "", b, err
	}
	b.StartLine, end, err = seekBack(l, ':', end, "StartLine")
	if err != nil {
		return "", b, err
	}
	fn := l[0:end]
	if fn == "" {
		return "", b, errors.New("a FileName cannot be blank")
	}
	return fn, b, nil
}

// seekBack searches backwards from end to find sep in l, then returns the
// value between sep and end as an integer.
// If seekBack fails, the returned error will reference what.
func seekBack(l string, sep byte, end int, what string) (value int, nextSep int, err error) {
	// Since we're seeking backwards and we know only ASCII is legal for these values,
	// we can ignore the possibility of non-ASCII characters.
	for start := end - 1; start >= 0; start-- {
		if l[start] == sep {
			i, err := strconv.Atoi(l[start+1 : end])
			if err != nil {
				return 0, 0, fmt.Errorf("couldn't parse %q: %v", what, err)
			}
			if i < 0 {
				return 0, 0, fmt.Errorf("negative values are not allowed for %s, found %d", what, i)
			}
			return i, start, nil
		}
	}
	return 0, 0, fmt.Errorf("couldn't find a %s before %s", string(sep), what)
}

type blocksByStart []ProfileBlock

func (b blocksByStart) Len() int      { return len(b) }
func (b blocksByStart) Swap(i, j int) { b[i], b[j] = b[j], b[i] }
func (b blocksByStart) Less(i, j int) bool {
	bi, bj := b[i], b[j]
	return bi.StartLine < bj.StartLine || bi.StartLine == bj.StartLine && bi.StartCol < bj.StartCol
}

// Boundary represents the position in a source file of the beginning or end of a
// block as reported by the coverage profile. In HTML mode, it will correspond to
// the opening or closing of a <span> tag and will be used to colorize the source
type Boundary struct {
	Offset int     // Location as a byte offset in the source file.
	Start  bool    // Is this the start of a block?
	Count  int     // Event count from the cover profile.
	Norm   float64 // Count normalized to [0..1].
	Index  int     // Order in input file.
}

// Boundaries returns a Profile as a set of Boundary objects within the provided src.
func (p *Profile) Boundaries(src []byte) (boundaries []Boundary) {
	// Find maximum count.
	max := 0
	for _, b := range p.Blocks {
		if b.Count > max {
			max = b.Count
		}
	}
	// Divisor for normalization.
	divisor := math.Log(float64(max))

	// boundary returns a Boundary, populating the Norm field with a normalized Count.
	index := 0
	boundary := func(offset int, start bool, count int) Boundary {
		b := Boundary{Offset: offset, Start: start, Count: count, Index: index}
		index++
		if !start || count == 0 {
			return b
		}
		if max <= 1 {
			b.Norm = 0.8 // Profile is in"set" mode; we want a heat map. Use cov8 in the CSS.
		} else if count > 0 {
			b.Norm = math.Log(float64(count)) / divisor
		}
		return b
	}

	line, col := 1, 2 // TODO: Why is this 2?
	for si, bi := 0, 0; si < len(src) && bi < len(p.Blocks); {
		b := p.Blocks[bi]
		if b.StartLine == line && b.StartCol == col {
			boundaries = append(boundaries, boundary(si, true, b.Count))
		}
		if b.EndLine == line && b.EndCol == col || line > b.EndLine {
			boundaries = append(boundaries, boundary(si, false, 0))
			bi++
			continue // Don't advance through src; maybe the next block starts here.
		}
		if src[si] == '\n' {
			line++
			col = 0
		}
		col++
		si++
	}
	sort.Sort(boundariesByPos(boundaries))
	return
}

type boundariesByPos []Boundary

func (b boundariesByPos) Len() int      { return len(b) }
func (b boundariesByPos) Swap(i, j int) { b[i], b[j] = b[j], b[i] }
func (b boundariesByPos) Less(i, j int) bool {
	if b[i].Offset == b[j].Offset {
		// Boundaries at the same offset should be ordered according to
		// their original position.
		return b[i].Index < b[j].Index
	}
	return b[i].Offset < b[j].Offset
}
