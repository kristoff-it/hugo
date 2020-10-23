package zig

import (
	"bytes"
	"fmt"
	"html/template"
	"os"
	"os/exec"

	"github.com/gohugoio/hugo/cache/namedmemcache"

	"github.com/gohugoio/hugo/deps"
	_errors "github.com/pkg/errors"
	"github.com/spf13/cast"
)

func New(deps *deps.Deps) *Namespace {
	cache := namedmemcache.New()

	return &Namespace{
		cache: cache,
		deps:  deps,
	}
}

type Namespace struct {
	cache *namedmemcache.Cache
	deps  *deps.Deps
}

func (ns *Namespace) Docgen(s interface{}) (template.HTML, error) {
	ss, err := cast.ToStringE(s)
	if err != nil {
		return "", err
	}

	fmt.Printf("[Zig Docgen] <%s>...", ss)
	generated := false

	file_stats, err := os.Stat(ss)
	if err != nil {
		return "", err
	}

	key := fmt.Sprintf("%s:%s:%s", ss, file_stats.ModTime(), file_stats.Size())
	v, err := ns.cache.GetOrCreate(key, func() (interface{}, error) {
		println(" GENERATING")
		generated = true
		res, err := ns.run(ss)
		if err != nil {
			return "", err
		}
		return res, nil
	})

	if err != nil {
		return "", err
	}

	if !generated {
		println(" CACHED")
	}

	return template.HTML(v.(string)), nil
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
	cmd := exec.Command(docgen_exe, zig_exe, file, "/dev/stdout")
	cmd.Stdout = &out
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return stderr.String(), _errors.Wrapf(err, "Error executing zigdocgen for [%s]: %s", file, stderr.String())
	}
	return out.String(), nil
}
