package emails

import (
	"bufio"
	"bytes"
	"fmt"
	"html/template"
	"io/ioutil"
	"net/textproto"
	"strings"
)

type Template string

type TemplateResult struct {
	Header textproto.MIMEHeader
	Text   []byte
	HTML   []byte
}

type TemplateTest struct {
	Data      map[string]interface{}
	Validator func(result *TemplateResult) error
}

func LoadTemplates(path string) (*Templater, []error) {
	entries, err := ioutil.ReadDir(path)
	if err != nil {
		return nil, []error{err}
	}

	templates := map[Template]*template.Template{}
	errors := []error{}
	for _, entry := range entries {
		filename := entry.Name()
		if strings.HasSuffix(filename, ".hdr") {
			tmplName := Template(filename[:len(filename)-4])
			templates[tmplName], err = template.ParseGlob(fmt.Sprintf("%s/%s.*", path, tmplName))
			if err != nil {
				errors = append(errors, err)
				continue
			}
		}
	}

	if len(errors) > 0 {
		return nil, errors
	}

	return &Templater{Templates: templates}, nil
}

type Templater struct {
	Templates map[Template]*template.Template
}

func (t *Templater) Validate(tmplName Template, tests ...TemplateTest) error {
	for i, test := range tests {
		result, err := t.Evaluate(tmplName, test.Data)
		if err != nil {
			return fmt.Errorf("test #%d: evaluation error: %s", i+1, err)
		}

		if err := test.Validator(result); err != nil {
			return fmt.Errorf("test #%d: validation error: %s", i+1, err)
		}
	}
	return nil
}

func (t *Templater) Evaluate(tmplName Template, data map[string]interface{}) (*TemplateResult, error) {
	tmpl, ok := t.Templates[tmplName]
	if !ok {
		return nil, fmt.Errorf("no templates found for %s", tmplName)
	}

	headerBytes, err := evaluate(tmpl, tmplName, "hdr", data)
	if err != nil {
		return nil, fmt.Errorf("%s.hdr: %s", tmplName, err)
	}

	result := &TemplateResult{}
	r := textproto.NewReader(bufio.NewReader(bytes.NewReader(headerBytes)))
	result.Header, err = r.ReadMIMEHeader()
	if err != nil {
		return nil, fmt.Errorf("%s.hdr: %s", tmplName, err)
	}

	if result.Text, err = evaluate(tmpl, tmplName, "txt", data); err != nil {
		return nil, fmt.Errorf("%s.txt: %s", tmplName, err)
	}

	if result.HTML, err = evaluate(tmpl, tmplName, "html", data); err != nil {
		return nil, fmt.Errorf("%s.html: %s", tmplName, err)
	}

	return result, nil
}

func evaluate(
	tmpl *template.Template, tmplName Template, ext string, data map[string]interface{}) ([]byte, error) {

	w := &bytes.Buffer{}
	name := fmt.Sprintf("%s.%s", tmplName, ext)
	if err := tmpl.ExecuteTemplate(w, name, data); err != nil {
		return nil, err
	}
	if ext == "hdr" {
		// Ensure content ends with a blank line.
		if !bytes.HasSuffix(w.Bytes(), []byte("\n\n")) {
			w.WriteString("\n\n")
		}
	}
	return w.Bytes(), nil
}
