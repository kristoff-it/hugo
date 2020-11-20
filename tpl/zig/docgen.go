package zig

import (
	"bytes"
	"fmt"
	"html/template"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sync"

	"github.com/gohugoio/hugo/cache/filecache"
	"github.com/gohugoio/hugo/deps"
	_errors "github.com/pkg/errors"
	"github.com/spf13/cast"
)

func New(deps *deps.Deps) *Namespace {
	return &Namespace{
		deps:      deps,
		notifLock: &sync.Mutex{},
		cache:     deps.FileCaches.AssetsCache(),
	}
}

type Namespace struct {
	deps      *deps.Deps
	notifLock *sync.Mutex
	cache     *filecache.Cache
}

func (ns *Namespace) Docgen(s interface{}) (template.HTML, error) {
	ss, err := cast.ToStringE(s)
	if err != nil {
		return "", err
	}

	file_stats, err := os.Stat(ss)
	if err != nil {
		return "", err
	}

	key := fmt.Sprintf("%s:%s:%s", ss, file_stats.ModTime(), file_stats.Size())

	_, result, err := ns.cache.GetOrCreateBytes(key, func() ([]byte, error) {
		fmt.Printf("[Zig Docgen] TESTING <%s>...\n", ss)
		res, err := ns.run(ss)
		return []byte(res), err
	})

	return template.HTML(result), err
}

func (ns *Namespace) run(file string) (string, error) {
	var out bytes.Buffer
	var stderr bytes.Buffer

	docgen_exe := os.Getenv("ZIG_DOCGEN")
	if docgen_exe == "" {
		return "", _errors.Errorf("missing docgen env variable, set `ZIG_DOCGEN` to the path where the docgen executable lives")
	}

	zig_exe := os.Getenv("ZIG_COMPILER")
	if zig_exe == "" {
		return "", _errors.Errorf("missing docgen env variable, set `ZIG_COMPILER` to the path where the zig executable lives")
	}

	dir, err := ioutil.TempDir("", "zig_docgen")
	if err != nil {
		return "", _errors.Errorf("failed to make temp dir")
	}
	defer os.RemoveAll(dir)

	abs_file, err := filepath.Abs(file)
	if err != nil {
		return "", _errors.Errorf("failed to grab an absolute path to the script")
	}

	cmd := exec.Command(docgen_exe, zig_exe, abs_file, "/dev/stdout")
	cmd.Dir = dir
	cmd.Stdout = &out
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return stderr.String(), _errors.Wrapf(err, "Error executing zigdocgen for [%s]: %s", file, stderr.String())
	}
	return out.String(), nil
}

func (ns *Namespace) Warmup(path string) {
	jobQ := make(chan string, 20)

	for i := 0; i < runtime.GOMAXPROCS(0); i++ {
		go func() {
			select {
			case p, ok := <-jobQ:
				if !ok {
					return
				}
				_, err := ns.Docgen(p)
				if err != nil {
					panic(err)
				}
			}
		}()
	}

	err := filepath.Walk(path,
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				panic(err)
			}

			// if it's an .md file, let's run it through the doctest tool
			if filepath.Ext(info.Name()) == ".md" {
				jobQ <- path
			}

			return nil
		})
	close(jobQ)
	if err != nil {
		panic(err)
	}
}
