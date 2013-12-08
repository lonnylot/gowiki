package main

import(
	"html/template"
	"net/http"
	"time"
	"io/ioutil"
	"os"
	"path"
	"bytes"
	"regexp"
)

type templates struct {
	root string // The root templates dir
	interval time.Duration // How often to refresh the templates
	template *template.Template
	reload <- chan time.Time
}

var t *templates

// Load the templates in the root and re-parse the templates in dir every interval
func LoadTemplates(root string, interval time.Duration) {
	cwd, err := os.Getwd()
	if err != nil {
		// The situation where err is not nil is beyond me now
		panic(err)
	}

	absRoot := path.Join(cwd, root)
	t = &templates{root: absRoot, interval: interval}
	t.reload = time.Tick(t.interval)
	t.loadTemplates()
}

// Load our templates from root
func (t *templates) loadTemplates() {
	s, err := t.templateFilePaths(t.root)
	if err != nil {
		panic(err)
	}
	temp := template.Must(template.ParseFiles(s...))
	t.template = temp
}

// Recursively get the template file paths
func (t *templates) templateFilePaths(root string) ([]string, error) {
	// Ensure the root has a trailing "/"
	if root[len(root)-1:] != "/" {
		root = root + "/"
	}

	fis, err := ioutil.ReadDir(root)
	if err != nil {
		return nil, err
	}
	
	s := make([]string, 0)
	for _, fi := range fis {
		if fi.IsDir() == false && path.Ext(fi.Name()) != ".html" {
			continue
		}

		fn := path.Join(root, fi.Name())
		temps := []string{fn}

		if fi.IsDir() {
			temps, err = t.templateFilePaths(fn)
			if err != nil {
				return nil, err
			}
		}

		s = append(s, temps...)
	}

	return s, nil
}

var linkPageName = regexp.MustCompile(`\[([a-zA-Z0-9]+)\]`)

// Render our templates that were previously parsed
func renderTemplate(w http.ResponseWriter, tmpl string, p *Page) {
	for {
		select {
			case <- t.reload:
				t.loadTemplates()
			default:
				tb := new(bytes.Buffer)
				err := t.template.ExecuteTemplate(tb, tmpl+".html", p)

				b := linkPageName.ReplaceAllFunc(tb.Bytes(), func(pn []byte) []byte {
					b := bytes.NewBufferString("<a href=\"/view/")
					b.Write(pn[1:len(pn)-1])
					b.WriteString("\">")
					b.Write(pn[1:len(pn)-1])
					b.WriteString("</a>")

					return b.Bytes()
				})

				tb.Reset()
				tb.Write(b)

				tb.WriteTo(w)
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
				}
				return
		}
	}
}
