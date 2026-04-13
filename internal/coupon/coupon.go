// Package coupon implements promo validation for the challenge: codes must be 8–10
// characters and appear in at least two of three gzipped corpora. See README for the
// full rules and scaling notes.
package coupon

import (
	"bufio"
	"compress/gzip"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"sync"
	"time"
)

// tokenKey is a fixed-size map key: first byte = length (8–10), next 10 = ASCII chars.
// A value type avoids a heap allocation per lookup.
type tokenKey [11]byte

func makeKey(b []byte) tokenKey {
	var k tokenKey
	k[0] = byte(len(b))
	copy(k[1:], b)
	return k
}

// Validator holds precomputed token sets from the three coupon files. It is safe
// for concurrent reads after Load returns.
type Validator struct {
	sets [3]map[tokenKey]struct{}
}

// File describes one remote gzipped corpus and the local filenames used for download
// and on-disk index cache.
type File struct {
	URL   string
	GZ    string
	Cache string
}

// Load downloads (if needed), indexes, and loads all three coupon corpora in parallel.
func Load(files [3]File) (*Validator, error) {
	var mu sync.Mutex
	var wg sync.WaitGroup
	var firstErr error
	v := &Validator{}

	for i, src := range files {
		wg.Add(1)
		go func(idx int, src File) {
			defer wg.Done()
			t0 := time.Now()
			set, err := loadOrBuild(src)
			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				if firstErr == nil {
					firstErr = fmt.Errorf("coupon file %d: %w", idx+1, err)
				}
				return
			}
			v.sets[idx] = set
			log.Printf("coupon: file %d ready (%d tokens, %s)", idx+1, len(set), time.Since(t0).Round(time.Millisecond))
		}(i, src)
	}
	wg.Wait()
	if firstErr != nil {
		return nil, firstErr
	}
	return v, nil
}

// Valid reports whether code satisfies assignment rules: length 8–10 and present
// in at least two of the three files.
func (v *Validator) Valid(code string) bool {
	if len(code) < 8 || len(code) > 10 {
		return false
	}
	key := makeKey([]byte(code))
	n := 0
	for _, set := range v.sets {
		if _, ok := set[key]; ok {
			n++
		}
	}
	return n >= 2
}

func loadOrBuild(src File) (map[tokenKey]struct{}, error) {
	if _, err := os.Stat(src.Cache); err == nil {
		log.Printf("coupon: loading cache %s", src.Cache)
		set, err := loadIndex(src.Cache)
		if err == nil {
			return set, nil
		}
		log.Printf("coupon: cache load failed (%v), rebuilding", err)
	}

	if _, err := os.Stat(src.GZ); os.IsNotExist(err) {
		log.Printf("coupon: downloading %s ...", src.GZ)
		if err := downloadFile(src.URL, src.GZ); err != nil {
			return nil, err
		}
	}

	log.Printf("coupon: scanning %s ...", src.GZ)
	f, err := os.Open(src.GZ)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	gr, err := gzip.NewReader(f)
	if err != nil {
		return nil, err
	}
	defer gr.Close()

	set, err := extractTokens(gr)
	if err != nil {
		return nil, err
	}

	go func() {
		if err := saveIndex(src.Cache, set); err != nil {
			log.Printf("coupon: warning: failed to save cache %s: %v", src.Cache, err)
		} else {
			log.Printf("coupon: saved cache %s", src.Cache)
		}
	}()

	return set, nil
}

// extractTokens scans r for maximal runs of [A-Z0-9] with length 8–10.
func extractTokens(r io.Reader) (map[tokenKey]struct{}, error) {
	set := make(map[tokenKey]struct{}, 1<<20)
	br := bufio.NewReaderSize(r, 1<<20)
	run := make([]byte, 0, 16)

	flush := func() {
		if l := len(run); l >= 8 && l <= 10 {
			set[makeKey(run)] = struct{}{}
		}
		run = run[:0]
	}

	for {
		c, err := br.ReadByte()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		if (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') {
			run = append(run, c)
		} else {
			flush()
		}
	}
	flush()
	return set, nil
}

const idxMagic = "CIDX"

func saveIndex(path string, set map[tokenKey]struct{}) error {
	tmp := path + ".tmp"
	f, err := os.Create(tmp)
	if err != nil {
		return err
	}
	bw := bufio.NewWriterSize(f, 1<<20)

	bw.WriteString(idxMagic)
	var countBuf [4]byte
	binary.LittleEndian.PutUint32(countBuf[:], uint32(len(set)))
	bw.Write(countBuf[:])

	for k := range set {
		bw.Write(k[:])
	}
	if err := bw.Flush(); err != nil {
		f.Close()
		os.Remove(tmp)
		return err
	}
	f.Close()
	return os.Rename(tmp, path)
}

func loadIndex(path string) (map[tokenKey]struct{}, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	br := bufio.NewReaderSize(f, 1<<20)
	magic := make([]byte, 4)
	if _, err := io.ReadFull(br, magic); err != nil || string(magic) != idxMagic {
		return nil, fmt.Errorf("invalid index magic")
	}
	var countBuf [4]byte
	if _, err := io.ReadFull(br, countBuf[:]); err != nil {
		return nil, err
	}
	count := binary.LittleEndian.Uint32(countBuf[:])

	set := make(map[tokenKey]struct{}, count)
	var k tokenKey
	for i := uint32(0); i < count; i++ {
		if _, err := io.ReadFull(br, k[:]); err != nil {
			return nil, err
		}
		set[k] = struct{}{}
	}
	return set, nil
}

func downloadFile(url, path string) error {
	resp, err := http.Get(url) //nolint:gosec // URL is fixed in sources
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected HTTP %d downloading %s", resp.StatusCode, url)
	}

	tmp := path + ".tmp"
	f, err := os.Create(tmp)
	if err != nil {
		return err
	}
	if _, err = io.Copy(f, resp.Body); err != nil {
		f.Close()
		os.Remove(tmp)
		return err
	}
	f.Close()
	return os.Rename(tmp, path)
}
