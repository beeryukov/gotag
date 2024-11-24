// Copyright 2015, David Howden
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

/*
The tag tool reads metadata from media files (as supported by the tag library).
*/
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/dhowden/tag"
	"github.com/dhowden/tag/mbz"
)

var usage = func() {
	fmt.Fprintf(os.Stderr, "usage: %s [optional flags] filename\n", os.Args[0])
	flag.PrintDefaults()
}

var (
	raw        = flag.Bool("raw", false, "show raw tag data")
	extractMBZ = flag.Bool("mbz", false, "extract MusicBrainz tag data (if available)")
)

func main() {
	/*
		for i := 9; i >= 0; i-- {
			fmt.Println(i)
		}
		return*/

	/*	file1, _ := os.OpenFile("/home/maxim/go/src/gotaglib/cmd/tag/1.txt", os.O_RDWR, 664)
		file1.Seek(55, io.SeekCurrent)
		defer file1.Close()
		tag.ShiftFileRight(file1, 10)
		return
	*/
	flag.Usage = usage
	flag.Parse()

	if flag.NArg() != 1 {
		usage()
		return
	}

	metadata := make(map[string]string)
	metadata["Title"] = "The Wizard"
	metadata["Album"] = "The Forgotten Tales"
	metadata["Artist"] = "Blind Guardian"
	metadata["Genre"] = "Power Metal"
	metadata["Date"] = "1996"
	metadata["Tracknumber"] = "5"

	//tag.PrepareVorbisComment(metadata)
	//return

	file, err := os.Open(flag.Arg(0))
	if err != nil {
		fmt.Printf("error loading file: %v", err)
		return
	}

	m, err := tag.ReadFrom(file)
	file.Close()

	if err != nil {
		fmt.Printf("error reading file: %v\n", err)
		return
	}
	printMetadata(m)
	//	return

	file, err = os.OpenFile(flag.Arg(0), os.O_RDWR, 664)

	tag.SaveTo(file, metadata)

	if *raw {
		fmt.Println()
		fmt.Println()

		tags := m.Raw()
		for k, v := range tags {
			if _, ok := v.(*tag.Picture); ok {
				fmt.Printf("%#v: %v\n", k, v)
				continue
			}
			fmt.Printf("%#v: %#v\n", k, v)
		}
	}

	if *extractMBZ {
		b, err := json.MarshalIndent(mbz.Extract(m), "", "  ")
		if err != nil {
			fmt.Printf("error marshalling MusicBrainz info: %v\n", err)
			return
		}

		fmt.Printf("\nMusicBrainz Info:\n%v\n", string(b))
	}
}

func printMetadata(m tag.Metadata) {
	fmt.Printf("Metadata Format: %v\n", m.Format())
	fmt.Printf("File Type: %v\n", m.FileType())

	fmt.Printf(" Title: %v\n", m.Title())
	fmt.Printf(" Album: %v\n", m.Album())
	fmt.Printf(" Artist: %v\n", m.Artist())
	fmt.Printf(" Composer: %v\n", m.Composer())
	fmt.Printf(" Genre: %v\n", m.Genre())
	fmt.Printf(" Year: %v\n", m.Year())

	track, trackCount := m.Track()
	fmt.Printf(" Track: %v of %v\n", track, trackCount)

	disc, discCount := m.Disc()
	fmt.Printf(" Disc: %v of %v\n", disc, discCount)

	fmt.Printf(" Picture: %v\n", m.Picture())
	fmt.Printf(" Lyrics: %v\n", m.Lyrics())
	fmt.Printf(" Comment: %v\n", m.Comment())
}
