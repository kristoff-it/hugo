package zig

import (
	"github.com/gohugoio/hugo/deps"
	"github.com/gohugoio/hugo/tpl/internal"
)

const name = "zignew"

func init() {
	f := func(d *deps.Deps) *internal.TemplateFuncsNamespace {
		ctx := New(d)

		if run_tests, _ := d.Cfg.GetStringMap("params")["render_zig_tests"]; run_tests.(bool) {
			// Start testing all snippets in a concurrent fashion!
			go ctx.Warmup("docgen-samples/")
		} else {
			println("[Zig Doctest] Test rendering is disabled!")
		}

		ns := &internal.TemplateFuncsNamespace{
			Name:    name,
			Context: func(args ...interface{}) interface{} { return ctx },
		}

		ns.AddMethodMapping(ctx.Doctest,
			[]string{"Doctest"},
			[][2]string{},
		)

		return ns

	}

	internal.AddTemplateFuncsNamespace(f)
}
