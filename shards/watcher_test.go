// Copyright 2018 Google Inc. All rights reserved.
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

package shards

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

type loggingLoader struct {
	loads chan string
	drops chan string
}

func (l *loggingLoader) load(k string) {
	l.loads <- k
}

func (l *loggingLoader) drop(k string) {
	l.drops <- k
}

func advanceFS() {
	time.Sleep(10 * time.Millisecond)
}

func TestDirWatcherUnloadOnce(t *testing.T) {
	dir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	logger := &loggingLoader{
		loads: make(chan string, 10),
		drops: make(chan string, 10),
	}
	_, err = NewDirectoryWatcher(dir, logger)
	if err == nil || !strings.Contains(err.Error(), "empty") {
		t.Fatalf("got %v, want 'empty'", err)
	}

	shard := filepath.Join(dir, "foo.zoekt")
	if err := ioutil.WriteFile(shard, []byte("hello"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	dw, err := NewDirectoryWatcher(dir, logger)
	if err != nil {
		t.Fatalf("NewDirectoryWatcher: %v", err)
	}
	defer dw.Stop()

	if got := <-logger.loads; got != shard {
		t.Fatalf("got load event %v, want %v", got, shard)
	}

	// Must sleep because of FS timestamp resolution.
	advanceFS()
	if err := ioutil.WriteFile(shard, []byte("changed"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	if got := <-logger.loads; got != shard {
		t.Fatalf("got load event %v, want %v", got, shard)
	}

	advanceFS()
	if err := os.Remove(shard); err != nil {
		t.Fatalf("Remove: %v", err)
	}

	if got := <-logger.drops; got != shard {
		t.Fatalf("got drops event %v, want %v", got, shard)
	}

	advanceFS()
	if err := ioutil.WriteFile(shard+".bla", []byte("changed"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	dw.Stop()

	select {
	case k := <-logger.loads:
		t.Errorf("spurious load of %q", k)
	case k := <-logger.drops:
		t.Errorf("spurious drops of %q", k)
	default:
	}
}

func TestHumanTruncateList(t *testing.T) {
	paths := []string{
		"dir/1",
		"dir/2",
		"dir/3",
		"dir/4",
	}

	assert := func(max int, want string) {
		got := humanTruncateList(paths, max)
		if got != want {
			t.Errorf("unexpected humanTruncateList max=%d.\ngot:  %s\nwant: %s", max, got, want)
		}
	}

	assert(1, "1... 3 more")
	assert(2, "1, 2... 2 more")
	assert(3, "1, 2, 3... 1 more")
	assert(4, "1, 2, 3, 4")
	assert(5, "1, 2, 3, 4")
}
