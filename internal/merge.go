package internal

import (
	"fmt"
	"io"
	"log"
	"sort"
)

func MergeProfiles(p *Profile, merge *Profile) {
	if p.Mode != merge.Mode {
		log.Fatalf("cannot merge profiles with different modes")
	}
	// Since the blocks are sorted, we can keep track of where the last block
	// was inserted and only look at the blocks after that as targets for merge
	startIndex := 0
	for _, b := range merge.Blocks {
		startIndex = MergeProfileBlock(p, b, startIndex)
	}
}

func MergeProfileBlock(p *Profile, pb ProfileBlock, startIndex int) int {
	sortFunc := func(i int) bool {
		pi := p.Blocks[i+startIndex]
		return pi.StartLine >= pb.StartLine && (pi.StartLine != pb.StartLine || pi.StartCol >= pb.StartCol)
	}

	i := 0
	if sortFunc(i) != true {
		i = sort.Search(len(p.Blocks)-startIndex, sortFunc)
	}
	i += startIndex
	if i < len(p.Blocks) && p.Blocks[i].StartLine == pb.StartLine && p.Blocks[i].StartCol == pb.StartCol {
		if p.Blocks[i].EndLine != pb.EndLine || p.Blocks[i].EndCol != pb.EndCol {
			log.Fatalf("OVERLAP MERGE: %v %v %v", p.FileName, p.Blocks[i], pb)
		}
		switch p.Mode {
		case "set":
			p.Blocks[i].Count |= pb.Count
		case "count", "atomic":
			p.Blocks[i].Count += pb.Count
		default:
			log.Fatalf("unsupported covermode: '%s'", p.Mode)
		}
	} else {
		if i > 0 {
			pa := p.Blocks[i-1]
			if pa.EndLine >= pb.EndLine && (pa.EndLine != pb.EndLine || pa.EndCol > pb.EndCol) {
				log.Fatalf("OVERLAP BEFORE: %v %v %v", p.FileName, pa, pb)
			}
		}
		if i < len(p.Blocks)-1 {
			pa := p.Blocks[i+1]
			if pa.StartLine <= pb.StartLine && (pa.StartLine != pb.StartLine || pa.StartCol < pb.StartCol) {
				log.Fatalf("OVERLAP AFTER: %v %v %v", p.FileName, pa, pb)
			}
		}
		p.Blocks = append(p.Blocks, ProfileBlock{})
		copy(p.Blocks[i+1:], p.Blocks[i:])
		p.Blocks[i] = pb
	}
	return i + 1
}

func AddProfile(profiles []*Profile, p *Profile) []*Profile {
	i := sort.Search(len(profiles), func(i int) bool { return profiles[i].FileName >= p.FileName })
	if i < len(profiles) && profiles[i].FileName == p.FileName {
		MergeProfiles(profiles[i], p)
	} else {
		profiles = append(profiles, nil)
		copy(profiles[i+1:], profiles[i:])
		profiles[i] = p
	}
	return profiles
}

func DumpProfiles(profiles []*Profile, out io.Writer) {
	if len(profiles) == 0 {
		return
	}
	n, err := fmt.Fprintf(out, "mode: %s\n", profiles[0].Mode)
	if err != nil {
		log.Printf("error writing mode: %v", err)
	}
	_ = n
	for _, p := range profiles {
		for _, b := range p.Blocks {
			n, err := fmt.Fprintf(out, "%s:%d.%d,%d.%d %d %d\n", p.FileName, b.StartLine, b.StartCol, b.EndLine, b.EndCol, b.NumStmt, b.Count)
			if err != nil {
				log.Printf("error writing profile block: %v", err)
			}
			_ = n
		}
	}
}
