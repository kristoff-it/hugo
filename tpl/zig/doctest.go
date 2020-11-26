package zig

import (
	"bytes"
	"fmt"
	"html/template"
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

func (ns *Namespace) Doctest(s interface{}) (template.HTML, error) {
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
		fmt.Printf("[Zig Doctest] TESTING <%s>...\n", ss)
		res, err := ns.run(ss)
		return []byte(res), err
	})

	return template.HTML(result), err
}

func (ns *Namespace) run(file string) (string, error) {
	var out bytes.Buffer
	var stderr bytes.Buffer

	doctest_exe := os.Getenv("ZIG_DOCTEST")
	if doctest_exe == "" {
		return "", _errors.Errorf("missing doctest env variable, set `ZIG_DOCTEST` to the path where the doctest executable lives")
	}

	abs_file, err := filepath.Abs(file)
	if err != nil {
		return "", _errors.Errorf("failed to grab an absolute path to the script")
	}

	cmd := exec.Command(doctest_exe, "inline", "--in_file", abs_file)
	cmd.Stdout = &out
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return stderr.String(), _errors.Wrapf(err, "Error executing zigdoctest for [%s]: %s", file, stderr.String())
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
				_, err := ns.Doctest(p)
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
			if filepath.Ext(info.Name()) == ".zig" {
				jobQ <- path
			}

			return nil
		})
	close(jobQ)
	if err != nil {
		panic(err)
	}
}
