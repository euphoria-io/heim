package emails

import (
	"errors"
	"fmt"
	"html/template"
	"io/ioutil"
	"net/textproto"
	"os"
	"path/filepath"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestEvaluation(t *testing.T) {
	templater := &Templater{templates: map[Template]*template.Template{}}
	templateSet := func(name Template, hdr, txt, html string) {
		tmpl := template.New(string(name)).Funcs(
			template.FuncMap{"error": func(arg string) (string, error) { return "", errors.New(arg) }})

		hdrTmpl := tmpl.New(fmt.Sprintf("%s.hdr", name))
		template.Must(hdrTmpl.Parse(hdr))
		txtTmpl := tmpl.New(fmt.Sprintf("%s.txt", name))
		template.Must(txtTmpl.Parse(txt))
		htmlTmpl := tmpl.New(fmt.Sprintf("%s.html", name))
		template.Must(htmlTmpl.Parse(html))
		templater.templates[name] = tmpl
	}

	templateSet(
		Template("welcome"), "Subject: {{.Subject}}", "{{.Subject}}", "<blink>{{.Subject}}</blink>")

	data := map[string]interface{}{"Subject": "test"}

	Convey("evaluate", t, func() {
		tmpl := templater.templates[Template("welcome")]

		Convey("error", func() {
			content, err := evaluate(tmpl, Template("notfound"), "hdr", data)
			So(err, ShouldNotBeNil)
			So(content, ShouldBeNil)
		})

		Convey("text part", func() {
			content, err := evaluate(tmpl, Template("welcome"), "txt", data)
			So(err, ShouldBeNil)
			So(string(content), ShouldEqual, "test")
		})

		Convey("header", func() {
			content, err := evaluate(tmpl, Template("welcome"), "hdr", data)
			So(err, ShouldBeNil)
			So(string(content), ShouldEqual, "Subject: test\n\n")
		})
	})

	Convey("Templater evaluation", t, func() {
		Convey("Template not found", func() {
			result, err := templater.Evaluate(Template("notfound"), data)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldStartWith, "no templates found")
			So(result, ShouldBeNil)
		})

		Convey("Header evaluation error", func() {
			templateSet(
				Template("hdr-err"), `{{error "hdr err"}}`, `{{error "txt err"}}`, `{{error "html err"}}`)
			result, err := templater.Evaluate(Template("hdr-err"), data)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "hdr err")
			So(result, ShouldBeNil)
		})

		Convey("Header parse error", func() {
			templateSet(Template("hdr-err"), "error", "", "")
			result, err := templater.Evaluate(Template("hdr-err"), data)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "malformed MIME header")
			So(result, ShouldBeNil)
		})

		Convey("Text evaluation error", func() {
			templateSet(
				Template("hdr-err"), "", `{{error "txt err"}}`, `{{error "html err"}}`)
			result, err := templater.Evaluate(Template("hdr-err"), data)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "txt err")
			So(result, ShouldBeNil)
		})

		Convey("HTML evaluation error", func() {
			templateSet(
				Template("hdr-err"), "", "", `{{error "html err"}}`)
			result, err := templater.Evaluate(Template("hdr-err"), data)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "html err")
			So(result, ShouldBeNil)
		})

		Convey("Successful evaluation", func() {
			result, err := templater.Evaluate(Template("welcome"), data)
			So(err, ShouldBeNil)

			expected := &TemplateResult{
				Header: textproto.MIMEHeader{"Subject": []string{"test"}},
				Text:   []byte("test"),
				HTML:   []byte("<blink>test</blink>"),
			}
			So(expected, ShouldResemble, result)
		})
	})
}

func TestTemplater(t *testing.T) {
	tempdir := func() string {
		td, err := ioutil.TempDir("", "")
		if err != nil {
			t.Fatal(err)
		}

		if err := os.Mkdir(filepath.Join(td, "static"), 0755); err != nil {
			t.Fatal(err)
		}

		return td
	}

	write := func(tmpDir, name, content string) {
		path := filepath.Join(tmpDir, name)
		err := ioutil.WriteFile(path, []byte(content), 0644)
		So(err, ShouldBeNil)
	}

	Convey("Pair of templates", t, func() {
		td := tempdir()
		defer os.RemoveAll(td)

		write(td, "welcome.hdr", "Subject: {{.Subject}}\nReply-To: max@euphoria.io")
		write(td, "welcome.txt", "Welcome!")
		write(td, "welcome.html", "<blink>{{.Subject}}</blink>")

		write(td, "alert.hdr", "Subject: alert!")
		write(td, "alert.txt", "alert!")
		write(td, "alert.html", "")

		templater, errs := LoadTemplates(td)
		So(errs, ShouldBeNil)

		result, err := templater.Evaluate(Template("welcome"), map[string]interface{}{"Subject": "test"})
		So(err, ShouldBeNil)
		So(len(result.Header), ShouldEqual, 2)
		So(string(result.HTML), ShouldEqual, "<blink>test</blink>")

		result, err = templater.Evaluate(Template("alert"), nil)
		So(err, ShouldBeNil)
		So(string(result.Text), ShouldEqual, "alert!")
	})

	Convey("Attachments", t, func() {
		td := tempdir()
		defer os.RemoveAll(td)

		write(filepath.Join(td, "static"), "test.png", "test")
		write(td, "test.hdr", "")
		write(td, "test.txt", "")
		write(td, "test.html", `<img src="{{call .file "test.png"}}">`)

		templater, errs := LoadTemplates(td)
		So(errs, ShouldBeNil)

		result, err := templater.Evaluate(Template("test"), nil)
		So(err, ShouldBeNil)
		So(string(result.HTML), ShouldEqual, `<img src="cid:test.png@localhost">`)
		So(result.Attachments, ShouldResemble, map[string]string{"test.png": "test.png@localhost"})
	})

	Convey("Path not found", t, func() {
		td := tempdir()
		defer os.RemoveAll(td)

		templater, errs := LoadTemplates(td + "/notfound")
		So(len(errs), ShouldEqual, 1)
		So(errs[0].Error(), ShouldEndWith, "no such file or directory")
		So(templater, ShouldBeNil)
	})

	Convey("Static directory not found", t, func() {
		td := tempdir()
		defer os.RemoveAll(td)

		So(os.Remove(filepath.Join(td, "static")), ShouldBeNil)
		templater, errs := LoadTemplates(td)
		So(len(errs), ShouldEqual, 1)
		So(errs[0].Error(), ShouldEndWith, "/static: no such file or directory")
		So(templater, ShouldBeNil)
	})

	Convey("Error opening static files", t, func() {
		td := tempdir()
		defer os.RemoveAll(td)

		write(filepath.Join(td, "static"), "test1.png", "test")
		write(filepath.Join(td, "static"), "test2.png", "test")
		So(os.Chmod(filepath.Join(td, "static", "test1.png"), 0), ShouldBeNil)
		So(os.Chmod(filepath.Join(td, "static", "test2.png"), 0), ShouldBeNil)

		templater, errs := LoadTemplates(td)
		So(len(errs), ShouldEqual, 2)
		So(errs[0].Error(), ShouldEndWith, "/static/test1.png: permission denied")
		So(errs[1].Error(), ShouldEndWith, "/static/test2.png: permission denied")
		So(templater, ShouldBeNil)
	})

	Convey("Attachment not found", t, func() {
		td := tempdir()
		defer os.RemoveAll(td)

		write(td, "test.hdr", "")
		write(td, "test.txt", "")
		write(td, "test.html", "{{call .file `test.png`}}")

		templater, errs := LoadTemplates(td)
		So(errs, ShouldBeNil)

		result, err := templater.Evaluate(Template("test"), nil)
		So(err, ShouldNotBeNil)
		So(err.Error(), ShouldEndWith, "test.png: file not available")
		So(result, ShouldBeNil)
	})

	Convey("Multiple parse errors", t, func() {
		td := tempdir()
		defer os.RemoveAll(td)

		write(td, "a.hdr", "{{")
		write(td, "a.txt", "")
		write(td, "a.html", "")

		write(td, "b.hdr", "")
		write(td, "b.txt", "")
		write(td, "b.html", "{{")

		templater, errs := LoadTemplates(td)
		So(len(errs), ShouldEqual, 2)
		So(templater, ShouldBeNil)
	})

	Convey("Validators", t, func() {
		td := tempdir()
		defer os.RemoveAll(td)

		write(td, "a.hdr", "Subject: {{.Subject}}")
		write(td, "a.txt", "welcome, {{.Name}}")
		write(td, "a.html", "welcome, <b>{{.Name}}</b>")

		write(td, "b.hdr", "Subject: hi")
		write(td, "b.txt", "another test")
		write(td, "b.html", "another test")

		templater, errs := LoadTemplates(td)
		So(errs, ShouldBeNil)

		headerTest := TemplateTest{
			Data: map[string]interface{}{"Subject": "test"},
			Validator: func(result *TemplateResult) error {
				if len(result.Header["Subject"]) != 1 || result.Header["Subject"][0] != "test" {
					return fmt.Errorf("invalid subject")
				}
				return nil
			},
		}

		textTest := TemplateTest{
			Data: map[string]interface{}{"Name": "name"},
			Validator: func(result *TemplateResult) error {
				if string(result.Text) != "welcome, name" {
					return fmt.Errorf("invalid text part")
				}
				return nil
			},
		}

		nilTest := TemplateTest{
			Validator: func(*TemplateResult) error { return nil },
		}

		Convey("Successful validation", func() {
			So(templater.Validate(Template("a"), headerTest, textTest), ShouldBeNil)
		})

		Convey("Validation error", func() {
			err := templater.Validate(Template("b"), nilTest, headerTest)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldEqual, "test #2: validation error: invalid subject")
		})
	})
}
