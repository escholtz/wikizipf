package main

import (
	"fmt"
	"io"
	"os"
	"runtime"
	"sort"
	"strings"
	"unicode"

	"github.com/dustin/go-wikiparse"
	"github.com/escholtz/segment"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

const indexName = "enwiki-20180701-pages-articles-multistream-index.txt.bz2"
const dataName = "enwiki-20180701-pages-articles-multistream.xml.bz2"

func isMn(r rune) bool {
	// Mn: nonspacing marks
	return unicode.Is(unicode.Mn, r)
}

// Replace accents and convert string to standard form.
// http://blog.golang.org/normalization#TOC_10.
func removeAccents(s string) string {
	// Don't think this is thread safe - can't be global
	t := transform.Chain(norm.NFD, transform.RemoveFunc(isMn), norm.NFC)
	c, _, err := transform.String(t, s)
	if err == nil {
		return c
	}
	return s
}

func normalize(s string) string {
	s = strings.ToLower(s)
	s = removeAccents(s)
	return s
}

type Pair struct {
	Key   string
	Value int
}

func main() {
	p, err := wikiparse.NewIndexedParser(
		indexName,
		dataName,
		runtime.GOMAXPROCS(0))

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error setting up parser: %v", err)
		os.Exit(1)
	}

	tokenCount := map[string]int{}

	for {
		var page *wikiparse.Page
		page, err := p.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			fmt.Fprintf(os.Stderr, "%v", err)
			break
		}

		// Skip pages that aren't articles
		if page.Ns != 0 {
			continue
		}

		// Skip redirects
		if len(page.Redir.Title) > 0 {
			continue
		}

		r := strings.NewReader(page.Title)
		seg := segment.NewWordSegmenter(r)
		for seg.Segment() {
			if seg.Type() == segment.None {
				continue
			}
			text := normalize(seg.Text())
			tokenCount[text]++
		}
	}

	pairs := make([]Pair, len(tokenCount))
	total := 0
	i := 0
	for k, v := range tokenCount {
		pairs[i] = Pair{k, v}
		total += v
		i++
	}

	sort.SliceStable(pairs, func(i, j int) bool {
		if pairs[i].Value != pairs[j].Value {
			return pairs[i].Value > pairs[j].Value
		}
		return strings.Compare(pairs[i].Key, pairs[j].Key) < 0
	})

	for _, p := range pairs {
		f := 100.0 * (float64(p.Value) / float64(total))
		fmt.Printf("%s\t%d\t%.6f\n", p.Key, p.Value, f)
	}
}
