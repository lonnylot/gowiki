package main

import(
	"html/template"
	"net/http"
	"time"
	"io/ioutil"
	"os"
	"path"
)

type templates struct {
	root string // The root templates dir
	interval time.Duration // How often to refresh the templates
	template *template.Template
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
	t.loadTemplates()
	go func(){
		tick := time.Tick(t.interval)
		for{
			select{
			case <-tick:
				t.loadTemplates()
			}
		}
	}()
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

// Render our templates that were previously parsed
func renderTemplate(w http.ResponseWriter, tmpl string, p *Page) {
	err := t.template.ExecuteTemplate(w, tmpl+".html", p)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
