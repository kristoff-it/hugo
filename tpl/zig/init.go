package zig

import (
	"github.com/gohugoio/hugo/deps"
	"github.com/gohugoio/hugo/tpl/internal"
)

const name = "zig"

func init() {
	f := func(d *deps.Deps) *internal.TemplateFuncsNamespace {
		params := d.Cfg.GetStringMap("params")
		ctx := New(d, params)

		if run_tests, ok := params["disable_zig_rendering"]; ok && !(run_tests.(bool)) {
			if basepath, ok := params["zig_code_basepath"]; ok {
				// Start testing all snippets concurrently!
				go ctx.Warmup(basepath.(string))
			} else {
				println("[Zig Doctest] Warning, no `zig_code_basepath` specified in the config, so startup warmup was skipped!")
			}

		} else {
			println("[Zig Doctest] Test rendering is disabled!")
		}

		ns := &internal.TemplateFuncsNamespace{
			Name:    name,
			Context: func(args ...interface{}) interface{} { return ctx },
		}

		ns.AddMethodMapping(ctx.Doctest,
			[]string{"doctest"},
			[][2]string{},
		)

		return ns

	}

	internal.AddTemplateFuncsNamespace(f)
}
