package main

import (
	"bytes"
	"embed"
	"fmt"
	"html/template"
	"io/fs"
	"net/http"
	"os"
	"slices"
	"strings"

	goldmark "github.com/yuin/goldmark"
	emoji "github.com/yuin/goldmark-emoji"
	meta "github.com/yuin/goldmark-meta"
	parser "github.com/yuin/goldmark/parser"
)

//go:embed all:content
var content embed.FS

const defaultPort = "8284"

func main() {
	if err := run(getPort(os.Getenv)); err != nil {
		fmt.Println("Error", err)
		os.Exit(1)
	}
}

func getPort(getenv func(string) string) string {
	if port := getenv("PORT"); port != "" {
		return port
	}

	return defaultPort
}

func run(port string) error {
	app, err := parseApp()
	if err != nil {
		return err
	}

	fmt.Printf("Listening on http://0.0.0.0:%s\n", port)

	return http.ListenAndServe(":"+port, app)
}

func parseRecipes(contentFS fs.FS) ([]Recipe, error) {
	recipes := []Recipe{}

	markdown := goldmark.New(
		goldmark.WithParserOptions(
			parser.WithAutoHeadingID(),
		),
		goldmark.WithExtensions(
			meta.Meta,
			emoji.Emoji,
		),
	)

	return recipes, fs.WalkDir(contentFS, ".",
		func(path string, d fs.DirEntry, err error) error {
			if d.Type().IsRegular() && strings.HasSuffix(path, ".md") {
				source, err := content.ReadFile("content/" + path)
				if err != nil {
					return err
				}

				var buf bytes.Buffer

				ctx := parser.NewContext()
				pwc := parser.WithContext(ctx)

				if err := markdown.Convert(source, &buf, pwc); err != nil {
					return err
				}

				recipes = append(recipes, Recipe{
					Path: path,
					Data: template.HTML(buf.Bytes()),
					Meta: meta.Get(ctx),
				})
			}

			return nil
		},
	)
}

type Recipe struct {
	Path string
	Meta map[string]any
	Data template.HTML
	List bool
}

func parseApp() (*App, error) {
	contentFS, err := fs.Sub(content, "content")
	if err != nil {
		return nil, err
	}

	recipes, err := parseRecipes(contentFS)
	if err != nil {
		return nil, err
	}

	return NewApp(recipes, contentFS), nil
}

func NewApp(recipes []Recipe, contentFS fs.FS) *App {
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
	listed := slices.Collect(func(yield func(Recipe) bool) {
		for _, r := range app.recipes {
			if unlisted, ok := r.Meta["Olistad"].(bool); ok && unlisted {
				return
			}

			if !yield(r) {
				return
			}
		}
	})

	index.ExecuteTemplate(w, "index", listed)
}

func (app *App) recipe(w http.ResponseWriter, r *http.Request) {
	for i := range app.recipes {
		if strings.Contains(r.URL.Path, app.recipes[i].Path) {
			recept.ExecuteTemplate(w, "recept", app.recipes[i])
			return
		}
	}
}

var index = template.Must(template.New("index").Parse(`<!DOCTYPE html>
<html>
	<head>
		<title>Recept</title>
		<style>	
			h1 { margin: 0.6em 0; line-height: 1; }
			h2 { margin-top: 1em; margin-bottom: 0; }
			a:link { color: #7CAF3C; }
			a:visited { color: #7CAF3C; }
			a:hover { color: #000000; }
			a:active { color: #7CAF3C; }
			
			ul {
				list-style: none;
				padding-left: 0;
				margin-top: 0;
			}

			body {
				font-size: clamp(32px, 4.6dvw, 52px);
				font-family: sans-serif;
			}

			main {
				max-width: clamp(100px, 87dvw, 1140px);
				margin: auto;
				width: 95dvw;
				display: flex;
				flex-direction: column;
			}
		</style>
	</head>
	<body>
		<main>
			<h2>Recept</h2>
			<ul>
				{{- range . }}
				<li><h1><a href="{{.Path}}">{{index .Meta "Titel"}}</a></h1></li>
				{{- end }}
			</ul>
		</main>
	</body>
</html>`))

var recept = template.Must(template.New("recept").Parse(`<!DOCTYPE html>
<html>
	<head>
		<title>Recept: {{ index .Meta "Titel" }}</title>
		<meta name="description" content="{{ index .Meta "Beskrivning"}}">
		<style>	
			h1 { margin: 0.6em 0; line-height: 1; }
			h2 { margin-top: 1em; margin-bottom: 0; }
			h3 { margin-top: 0.5em; margin-bottom: -0.5em; }
			a:link { color: #7CAF3C; }
			a:visited { color: #7CAF3C; }
			a:hover { color: #000000; }
			a:active { color: #7CAF3C; }
			ul { list-style-type: none; padding-inline-start: 0; }
			ul, ol { margin-top: 1em; }
			li { margin-bottom: 1em; }
			li p { margin-top: 0; }
			li::marker { font-weight: 900; }
			img { width: 100%; }
			em { font-weight: 300; }			
			
			body {
				font-size: clamp(32px, 4.6dvw, 52px);
				font-family: sans-serif;
			}

			main {
				max-width: clamp(100px, 87dvw, 1140px);
 				margin: auto;
 				width: 95dvw;
 				display: flex;
 				flex-direction: column;
			}
		</style>
	</head>
	<body>
		<main>
			{{- if index .Meta "Titel" }}
			<h2><a href="/">Recept</a></h2>
			<h1>{{ index .Meta "Titel" }}</h1>
			{{- end}}
			{{- if index .Meta "Bild" }}
			<img src="{{index .Meta "Bild" }}">
			{{- end}}
			{{ .Data }}
		</main>
	</body>
</html>`))
