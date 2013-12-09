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
	"sync"
)

type Options struct {
	root string // The root templates dir
	interval time.Duration // How often to refresh the templates
	extensions []string
}

type templates struct {
	Options
	template *template.Template
	reload <- chan time.Time
	sync.Mutex
}

var t *templates

// Load the templates in the root and re-parse the templates in dir every interval
func LoadTemplates(o Options) {
	t = &templates{Options:o}

	cwd, err := os.Getwd()
	if err != nil {
		// The situation where err is not nil is beyond me now
		panic(err)
	}
	t.root = path.Join(cwd, t.root)

	t.reload = time.Tick(t.interval)
	t.loadTemplates()
}

// Load our templates from root
func (t *templates) loadTemplates() {
	temp := template.New("templates")

	err := t.parseTemplates(t.root, temp)
	if err != nil {
		panic(err)
	}
	t.Lock()
	t.template = temp
	t.Unlock()
}

// Recursively get the template file paths
func (t *templates) parseTemplates(root string, temp *template.Template) error {
	// Ensure the root has a trailing "/"
	if root[len(root)-1:] != "/" {
		root = root + "/"
	}

	fis, err := ioutil.ReadDir(root)
	if err != nil {
		return err
	}
	
	for _, fi := range fis {
		fn := path.Join(root, fi.Name())
		ext := path.Ext(fi.Name())

		if fi.IsDir() {
			err = t.parseTemplates(fn, temp)
			if err != nil {
				return err
			}
		} else {
			// Make sure the extension is allowed
			allowed := false
			for _, e := range t.extensions {
				if e == ext {
					allowed = true
					break
				}
			}
			if allowed == false {
				continue
			}
		}
		
		b, err := ioutil.ReadFile(fn)
		if err != nil {
			return err
		}

		s := string(b)
		name := fi.Name()
		name = name[:len(name)-len(ext)]
		_, err = temp.New(name).Parse(s)
		if err != nil {
			return err
		}
	}

	return nil
}

var wikiPageName = regexp.MustCompile(`\[([a-zA-Z0-9]+)\]`)

func linkPageName(b []byte) []byte {
	// Exclude the open/close brackets from the page name
	pn := b[1:len(b)-1]

	// Write the link
	link := bytes.NewBufferString("<a href=\"/view/")
	link.Write(pn)
	link.WriteString("\">")
	link.Write(pn)
	link.WriteString("</a>")

	return link.Bytes()
}

type renderBuffer struct {
	bytes.Buffer
}

func (rb *renderBuffer) linkPageName() {
	// Link our wiki PageName
	b := wikiPageName.ReplaceAllFunc(rb.Bytes(), linkPageName)
	rb.Reset()
	rb.Write(b)
}

// Render our templates that were previously parsed
func renderTemplate(w http.ResponseWriter, tmpl string, data interface{}) {
	for {
		select {
			case <- t.reload:
				t.loadTemplates()
			default:
				rb := new(renderBuffer)
				t.Lock()
				err := t.template.ExecuteTemplate(rb, tmpl, data)
				t.Unlock()
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}

				rb.linkPageName()
				rb.WriteTo(w)

				return
		}
	}
}
