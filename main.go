package main

import (
	"bytes"
	"embed"
	"fmt"
	"html/template"
	"io/fs"
	"net/http"
	"os"
	"strings"

	goldmark "github.com/yuin/goldmark"
	meta "github.com/yuin/goldmark-meta"
	"github.com/yuin/goldmark/parser"
)

//go:embed all:content
var content embed.FS
var contentFS, _ = fs.Sub(content, "content")

const defaultPort = "8284"

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = defaultPort
	}

	run(port)
}

func parseRecipes() ([]Recipe, error) {
	recipes := []Recipe{}

	markdown := goldmark.New(
		goldmark.WithExtensions(
			meta.Meta,
		),
	)

	fs.WalkDir(contentFS, ".", func(path string, d fs.DirEntry, err error) error {
		if d.Type().IsRegular() && strings.HasSuffix(path, ".md") {

			source, err := content.ReadFile("content/" + path)
			if err != nil {
				return err
			}

			var buf bytes.Buffer

			context := parser.NewContext()
			if err := markdown.Convert(source, &buf, parser.WithContext(context)); err != nil {
				return err
			}

			recipes = append(recipes, Recipe{
				Path: path,
				Data: template.HTML(buf.Bytes()),
				Meta: meta.Get(context),
			})
		}

		return nil
	})

	return recipes, nil
}

func run(port string) error {
	recipes, err := parseRecipes()
	if err != nil {
		return err
	}

	mux := http.NewServeMux()

	mux.Handle("/", NewApp(recipes))

	fmt.Printf("Listening on http://0.0.0.0:%s\n", port)

	return http.ListenAndServe(":"+port, mux)
}

type Recipe struct {
	Path string
	Meta map[string]any
	Data template.HTML
}

func NewApp(recipes []Recipe) *App {
	return &App{
		recipes: recipes,
		content: http.StripPrefix("/content/", http.FileServer(http.FS(contentFS))),
	}
}

type App struct {
	recipes []Recipe
	content http.Handler
}

func (app *App) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/":
		app.index(w, r)
	case "/favicon.ico":
	default:
		if strings.HasSuffix(r.URL.Path, ".md") {
			app.recipe(w, r)
		} else {
			app.content.ServeHTTP(w, r)
		}
	}
}

func (app *App) index(w http.ResponseWriter, _ *http.Request) {
	index.ExecuteTemplate(w, "index", app.recipes)
}

func (app *App) recipe(w http.ResponseWriter, r *http.Request) {
	for i := range app.recipes {
		if strings.Contains(r.URL.Path, app.recipes[i].Path) {
			recept.ExecuteTemplate(w, "recept", app.recipes[i])
			return
		}
	}
}

var index = template.Must(template.New("index").Parse(`
<html>
	<head>
		<title>Recept</title>
		<style>	
			body {
				font-size: clamp(32px, 6dvw, 128px);
				font-family: sans-serif;
			}

			ul {
				list-style: none;
 				padding-left: 0;
			}

			h1 { margin-bottom: 0; }

			a:link { color: #7CAF3C; }
			a:visited { color: #7CAF3C; }
			a:hover { color: #000000; }
			a:active { color: #7CAF3C; }

			main {
					margin: auto;
    	    width: 95dvw;
    	    display: flex;
    	    flex-direction: column;
    	    align-items: center;
    	}
		</style>
	</head>
	<body>
		<main>
			<h1>Recept</h1>
			<ul>
				{{ range . }}
				<li><h2><a href="{{.Path}}">{{index .Meta "Titel"}}</a></h2></li>
				{{ end }}
			</ul>
		</main>
	</body>
</html>
`))

var recept = template.Must(template.New("recept").Parse(`
<html>
	<head>
		<title>{{ index .Meta "Titel" }}</title>
		<meta name="description" content="{{ index .Meta "Beskrivning"}}">
		<style>	
			h1 { margin-bottom: 0; }
			li { margin-bottom: 1em; }
			img { width: 100%; }
			
			a:link { color: #7CAF3C; }
			a:visited { color: #7CAF3C; }
			a:hover { color: #000000; }
			a:active { color: #7CAF3C; }
			
			body {
				font-size: clamp(32px, 4.5dvw, 48px);
				font-family: sans-serif;
			}

			main {
					margin: auto;
    	    width: 90dvw;
    	    display: flex;
    	    flex-direction: column;
    	    align-items: center;
    	}
		</style>
	</head>
	<body>
		{{ if index .Meta "Titel" }}
		<h1>{{ index .Meta "Titel" }}</h1>
		{{end}}
		{{ if index .Meta "Bild" }}
		<img src="{{index .Meta "Bild" }}">
		{{end}}
		<main>
		 {{.Data}}
		</main>
	</body>
</html>
`))