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

func New(deps *deps.Deps, params map[string]interface{}) *Namespace {
	return &Namespace{
		deps:      deps,
		notifLock: &sync.Mutex{},
		cache:     deps.FileCaches.AssetsCache(),
		params:    params,
	}
}

type Namespace struct {
	deps      *deps.Deps
	notifLock *sync.Mutex
	cache     *filecache.Cache
	params    map[string]interface{}
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

	var doctest_exe string = "doctest" // assume its in path
	if dc_path, ok := ns.params["zig_doctest_path"]; ok {
		doctest_exe = dc_path.(string)
	}

	var zig_exe string // if missing we don't specify the setting
	if zig_path, ok := ns.params["zig_path"]; ok {
		zig_exe = zig_path.(string)
	}

	abs_file, err := filepath.Abs(file)
	if err != nil {
		return "", _errors.Errorf("failed to grab an absolute path to the script")
	}
	command_args := []string {
		"inline", 
		"--in_file", 
		abs_file,
	}
	if (zig_exe != "") {
		command_args = append(command_args, []string{
			"--zig_exe",
			zig_exe,
		}...)
	}
	cmd := exec.Command(doctest_exe, command_args...)
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
