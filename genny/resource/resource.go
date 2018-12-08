package resource

import (
	"fmt"
	"os/exec"
	"text/template"

	"github.com/gobuffalo/flect"
	"github.com/gobuffalo/flect/name"
	"github.com/gobuffalo/genny"
	"github.com/gobuffalo/genny/movinglater/attrs"
	"github.com/gobuffalo/genny/movinglater/gotools"
	"github.com/gobuffalo/meta"
	"github.com/gobuffalo/packd"
	"github.com/gobuffalo/packr/v2"
	"github.com/pkg/errors"
)

var actions = []name.Ident{
	name.New("list"),
	name.New("show"),
	name.New("new"),
	name.New("create"),
	name.New("edit"),
	name.New("update"),
	name.New("destroy"),
}

func New(opts *Options) (*genny.Generator, error) {
	g := genny.New()

	if err := opts.Validate(); err != nil {
		return g, errors.WithStack(err)
	}

	core := packr.New("github.com/gobuffalo/buffalo/genny/resource/templates/core", "../resource/templates/core")

	if err := g.Box(core); err != nil {
		return g, errors.WithStack(err)
	}

	var abox packd.Box
	if opts.SkipModel {
		abox = packr.New("github.com/gobuffalo/buffalo/genny/resource/templates/standard", "../resource/templates/standard")
	} else {
		abox = packr.New("github.com/gobuffalo/buffalo/genny/resource/templates/use_model", "../resource/templates/use_model")
	}

	if err := g.Box(abox); err != nil {
		return g, errors.WithStack(err)
	}

	pres := presenter{
		App:   opts.App,
		Name:  name.New(opts.Name),
		Model: name.New(opts.Model),
		Attrs: opts.Attrs,
	}
	data := map[string]interface{}{
		"opts":    pres,
		"actions": actions,
	}
	helpers := template.FuncMap{
		"camelize": func(s string) string {
			return flect.Camelize(s)
		},
	}

	x := pres.Name.Resource().File().String()
	g.Transformer(gotools.TemplateTransformer(data, helpers))
	g.Transformer(genny.Replace("resource-name", x))
	g.Transformer(genny.Replace("resource-use_model", x))

	g.RunFn(func(r *genny.Runner) error {
		if !opts.SkipModel {
			if _, err := r.LookPath("buffalo-pop"); err != nil {
				if err := gotools.Get("github.com/gobuffalo/buffalo-pop")(r); err != nil {
					return errors.WithStack(err)
				}
			}

			return r.Exec(modelCommand(pres.Model, opts))
		}

		return nil
	})

	g.RunFn(func(r *genny.Runner) error {
		f, err := r.FindFile("actions/app.go")
		if err != nil {
			return errors.WithStack(err)
		}
		stmt := fmt.Sprintf("app.Resource(\"/%s\", %sResource{})", pres.Name.URL(), pres.Name.Resource())
		f, err = gotools.AddInsideBlock(f, "if app == nil {", stmt)
		if err != nil {
			return errors.WithStack(err)
		}
		return r.File(f)
	})
	return g, nil
}

func modelCommand(model name.Ident, opts *Options) *exec.Cmd {
	args := opts.Attrs.Slice()
	args = append(args[:0], args[0+1:]...)
	args = append([]string{"pop", "g", "model", model.Singularize().Underscore().String()}, args...)

	if opts.SkipMigration {
		args = append(args, "--skip-migration")
	}

	return exec.Command("buffalo-pop", args...)
}

type presenter struct {
	App   meta.App
	Name  name.Ident
	Model name.Ident
	Attrs attrs.Attrs
	// SkipMigration bool
	// SkipModel     bool
	// SkipTemplates bool
	// UseModel      bool
	// Args          []string
}