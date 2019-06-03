package test

import (
	"bytes"
	"fmt"
	"scrigo/template"
	"testing"
)

var templateCases = []struct {
	src string
	out string
}{
	{
		src: `Only text`,
		out: `Only text`,
	},
}

func TestTemplate(t *testing.T) {
	i := 0
	for _, cas := range templateCases {
		t.Run(fmt.Sprintf("Template test #%d", i), func(t *testing.T) {
			r := template.MapReader{"/main": []byte(cas.src)}
			templ, err := template.Load("/main", r, nil, template.ContextText, template.LoadOption(0))
			if err != nil {
				t.Fatalf("loading error: %s", err)
			}
			w := &bytes.Buffer{}
			err = templ.Render(w, nil, template.RenderOptions{})
			if err != nil {
				t.Fatalf("rendering error: %s", err)
			}
			if cas.out != w.String() {
				t.Fatalf("expecting %q, got %q", cas.out, w.String())
			}
		})
	}
}
