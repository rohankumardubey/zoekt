// Copyright 2016 Google Inc. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package zoekt

import (
	"bytes"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path"
	"reflect"
	"testing"
)

var update = flag.Bool("update", false, "update the golden files of this test")

func TestReadWrite(t *testing.T) {
	b, err := NewIndexBuilder(nil)
	if err != nil {
		t.Fatalf("NewIndexBuilder: %v", err)
	}

	if err := b.AddFile("filename", []byte("abcde")); err != nil {
		t.Fatalf("AddFile: %v", err)
	}

	var buf bytes.Buffer
	b.Write(&buf)
	f := &memSeeker{buf.Bytes()}

	r := reader{r: f}

	var toc indexTOC
	err = r.readTOC(&toc)

	if err != nil {
		t.Errorf("got read error %v", err)
	}
	if toc.fileContents.data.sz != 5 {
		t.Errorf("got contents size %d, want 5", toc.fileContents.data.sz)
	}

	data, err := r.readIndexData(&toc)
	if err != nil {
		t.Fatalf("readIndexData: %v", err)
	}
	if got := data.fileName(0); string(got) != "filename" {
		t.Errorf("got filename %q, want %q", got, "filename")
	}

	if len(data.ngrams) != 3 {
		t.Fatalf("got ngrams %v, want 3 ngrams", data.ngrams)
	}

	if _, ok := data.ngrams[stringToNGram("bcq")]; ok {
		t.Errorf("found ngram bcd in %v", data.ngrams)
	}
}

func TestReadWriteNames(t *testing.T) {
	b, err := NewIndexBuilder(nil)
	if err != nil {
		t.Fatalf("NewIndexBuilder: %v", err)
	}

	if err := b.AddFile("abCd", []byte("")); err != nil {
		t.Fatalf("AddFile: %v", err)
	}

	var buf bytes.Buffer
	b.Write(&buf)
	f := &memSeeker{buf.Bytes()}

	r := reader{r: f}

	var toc indexTOC
	if err := r.readTOC(&toc); err != nil {
		t.Errorf("got read error %v", err)
	}
	if toc.fileNames.data.sz != 4 {
		t.Errorf("got contents size %d, want 4", toc.fileNames.data.sz)
	}

	data, err := r.readIndexData(&toc)
	if err != nil {
		t.Fatalf("readIndexData: %v", err)
	}
	if !reflect.DeepEqual([]uint32{0, 4}, data.fileNameIndex) {
		t.Errorf("got index %v, want {0,4}", data.fileNameIndex)
	}
	if got := data.fileNameNgrams[stringToNGram("bCd")]; !reflect.DeepEqual(got, []uint32{1}) {
		t.Errorf("got trigram bcd at bits %v, want sz 2", data.fileNameNgrams)
	}
}

func TestBackwardsCompat(t *testing.T) {
	if *update {
		b, err := NewIndexBuilder(nil)
		if err != nil {
			t.Fatalf("NewIndexBuilder: %v", err)
		}

		if err := b.AddFile("filename", []byte("abcde")); err != nil {
			t.Fatalf("AddFile: %v", err)
		}

		var buf bytes.Buffer
		b.Write(&buf)

		outname := fmt.Sprintf("testdata/backcompat/new_v%d.%05d.zoekt", IndexFormatVersion, 0)
		t.Log("writing new file", outname)

		err = os.WriteFile(outname, buf.Bytes(), 0644)
		if err != nil {
			t.Fatalf("Creating output file: %v", err)
		}
	}

	compatibleFiles, err := fs.Glob(os.DirFS("."), "testdata/backcompat/*.zoekt")
	if err != nil {
		t.Fatalf("fs.Glob: %v", err)
	}

	for _, fname := range compatibleFiles {
		t.Run(path.Base(fname),
			func(t *testing.T) {
				f, err := os.Open(fname)
				if err != nil {
					t.Fatal("os.Open", err)
				}
				idx, err := NewIndexFile(f)
				if err != nil {
					t.Fatal("NewIndexFile", err)
				}
				r := reader{r: idx}

				var toc indexTOC
				err = r.readTOC(&toc)

				if err != nil {
					t.Errorf("got read error %v", err)
				}
				if toc.fileContents.data.sz != 5 {
					t.Errorf("got contents size %d, want 5", toc.fileContents.data.sz)
				}

				data, err := r.readIndexData(&toc)
				if err != nil {
					t.Fatalf("readIndexData: %v", err)
				}
				if got := data.fileName(0); string(got) != "filename" {
					t.Errorf("got filename %q, want %q", got, "filename")
				}

				if len(data.ngrams) != 3 {
					t.Fatalf("got ngrams %v, want 3 ngrams", data.ngrams)
				}

				if _, ok := data.ngrams[stringToNGram("bcq")]; ok {
					t.Errorf("found ngram bcd in %v", data.ngrams)
				}
			},
		)
	}
}
