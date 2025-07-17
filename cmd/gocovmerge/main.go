package main

import (
	"flag"
	"github.com/ja7ad/gocovmerge/internal"
	"log"
	"os"
)

func main() {
	flag.Parse()

	var merged []*internal.Profile

	for _, file := range flag.Args() {
		profiles, err := internal.ParseProfiles(file)
		if err != nil {
			log.Fatalf("failed to parse profiles: %v", err)
		}
		for _, p := range profiles {
			merged = internal.AddProfile(merged, p)
		}
	}

	internal.DumpProfiles(merged, os.Stdout)
}
