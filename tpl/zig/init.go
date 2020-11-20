package zig

import (
	"github.com/gohugoio/hugo/deps"
	"github.com/gohugoio/hugo/tpl/internal"
)

const name = "zig"

func init() {
	f := func(d *deps.Deps) *internal.TemplateFuncsNamespace {
		ctx := New(d)

		if run_tests, _ := d.Cfg.GetStringMap("params")["render_zig_tests"]; run_tests.(bool) {
			// Start testing all snippets in a concurrent fashion!
			go ctx.Warmup("docgen-samples/")
		} else {
			println("[Zig] Test rendering is disabled!")
		}

		ns := &internal.TemplateFuncsNamespace{
			Name:    name,
			Context: func(args ...interface{}) interface{} { return ctx },
		}

		ns.AddMethodMapping(ctx.Docgen,
			[]string{"Docgen"},
			[][2]string{},
		)

		return ns

	}

	internal.AddTemplateFuncsNamespace(f)
}
