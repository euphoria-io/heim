package templates

import (
	"bytes"
	"fmt"
	"html/template"
	"io/ioutil"
	"net/url"
	"path/filepath"
	"strings"
)

type TemplateTest struct {
	Data interface{}
}

type Templater struct {
	Templates map[string]*template.Template

	staticFiles map[string][]byte
}

func (t *Templater) Load(path string) []error {
	// Initialize if necessary.
	if t.staticFiles == nil {
		t.staticFiles = map[string][]byte{}
	}
	if t.Templates == nil {
		t.Templates = map[string]*template.Template{}
	}

	// Scan the top-level directory of the given path.
	entries, err := ioutil.ReadDir(path)
	if err != nil {
		return []error{err}
	}

	// Find and parse templates.
	errors := []error{}
	for _, entry := range entries {
		filename := entry.Name()
		if strings.HasSuffix(filename, ".html") {
			tmplName := strings.TrimSuffix(filename, ".html")
			t.Templates[tmplName], err = template.ParseGlob(filepath.Join(path, tmplName+".*"))
			if err != nil {
				errors = append(errors, err)
				continue
			}
		}
	}
	if len(errors) > 0 {
		return errors
	}

	// Scan the static directory under the given path for possible attachments, and load into
	// memory.
	entries, err = ioutil.ReadDir(filepath.Join(path, "static"))
	if err != nil {
		return []error{err}
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			fpath := filepath.Join(path, "static", entry.Name())
			t.staticFiles[entry.Name()], err = ioutil.ReadFile(fpath)
			if err != nil {
				errors = append(errors, err)
				continue
			}
		}
	}
	if len(errors) > 0 {
		return errors
	}

	return nil
}

func (t *Templater) Evaluate(name string, context interface{}) ([]byte, error) {
	baseName := filepath.Base(name)
	ext := filepath.Ext(baseName)
	tmplName := baseName[:len(baseName)-len(ext)]
	tmpl, ok := t.Templates[tmplName]
	if !ok {
		return nil, fmt.Errorf("no template found for %s", name)
	}

	if sf, ok := context.(staticFiles); ok {
		sf.setStaticFiles(t.staticFiles)
	}

	w := &bytes.Buffer{}
	if err := tmpl.ExecuteTemplate(w, name, context); err != nil {
		return nil, err
	}

	if strings.HasSuffix(name, ".hdr") && !bytes.HasSuffix(w.Bytes(), []byte("\n\n")) {
		w.WriteString("\n\n")
	}

	return w.Bytes(), nil
}

func (t *Templater) Validate(name string, testCase ...TemplateTest) []error {
	return nil
}

type Attachment struct {
	Name      string
	ContentID string
	Content   []byte
}

type attachmentList []Attachment

func (as attachmentList) Len() int           { return len(as) }
func (as attachmentList) Less(i, j int) bool { return as[i].Name < as[j].Name }
func (as attachmentList) Swap(i, j int)      { as[i], as[j] = as[j], as[i] }

type staticFiles interface {
	setStaticFiles(map[string][]byte)
	Attachments() map[string]Attachment
}

type StaticFiles struct {
	domain    string
	available map[string][]byte
	attached  map[string]Attachment
}

func (sf *StaticFiles) setStaticFiles(files map[string][]byte) { sf.available = files }
func (sf *StaticFiles) Attachments() map[string]Attachment     { return sf.attached }

func (sf *StaticFiles) File(path string) (template.URL, error) {
	// Verify file is available.
	content, ok := sf.available[path]
	if !ok {
		return "", fmt.Errorf("%s: file not available", path)
	}

	// Derive Content-ID from path and domain.
	domain := sf.domain
	if domain == "" {
		domain = "localhost"
	}
	if sf.attached == nil {
		sf.attached = map[string]Attachment{}
	}
	sf.attached[path] = Attachment{
		Name:      path,
		ContentID: fmt.Sprintf("%s@%s", url.QueryEscape(path), domain),
		Content:   content,
	}
	return template.URL("cid:" + sf.attached[path].ContentID), nil
}
