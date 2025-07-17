package main

import (
	"flag"
	"github.com/ja7ad/gocovmerge/internal"
	"golang.org/x/tools/cover"
	"log"
	"os"
)

func main() {
	flag.Parse()

	var merged []*cover.Profile

	for _, file := range flag.Args() {
		profiles, err := cover.ParseProfiles(file)
		if err != nil {
			log.Fatalf("failed to parse profiles: %v", err)
		}
		for _, p := range profiles {
			merged = internal.AddProfile(merged, p)
		}
	}

	internal.DumpProfiles(merged, os.Stdout)
}
