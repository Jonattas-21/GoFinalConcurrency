package main

import (
	"final-project/data"
	"fmt"
	"html/template"
	"net/http"
	"time"
)

var pathTemplate = "./cmd/web/templates"

type TemplateData struct {
	StringMap     map[string]string
	IntMap        map[string]int
	FloatMap      map[string]float64
	Data          map[string]interface{}
	Flash         string
	Warning       string
	Error         string
	Authenticated bool
	Now           time.Time
	User          *data.User
}

func (app *Config) render(w http.ResponseWriter, r *http.Request, t string, td *TemplateData) {
	partials := []string{
		fmt.Sprintf("%s/base.layout.gohtml", pathTemplate),
		fmt.Sprintf("%s/header.partial.gohtml", pathTemplate),
		fmt.Sprintf("%s/navbar.partial.gohtml", pathTemplate),
		fmt.Sprintf("%s/footer.partial.gohtml", pathTemplate),
		fmt.Sprintf("%s/alerts.partial.gohtml", pathTemplate),
	}

	var templateSlices []string

	templateSlices = append(templateSlices, fmt.Sprintf("%s/%s", pathTemplate, t))

	for _, p := range partials {
		templateSlices = append(templateSlices, p)
	}

	if td == nil {
		td = &TemplateData{}
	}

	tmpl, err := template.ParseFiles(templateSlices...)

	if err != nil {
		app.Errorlog.Println(err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := tmpl.Execute(w, app.AddDefaultData(td, r)); err != nil {
		app.Errorlog.Println(err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

}

func (app *Config) AddDefaultData(td *TemplateData, r *http.Request) *TemplateData {
	td.Flash = app.Session.PopString(r.Context(), "flash")
	td.Error = app.Session.PopString(r.Context(), "error")
	td.Warning = app.Session.PopString(r.Context(), "warning")
	if app.IsAuthenticated(r) {
		td.Authenticated = true
		user, ok := app.Session.Get(r.Context(), "user").(data.User)
		if !ok {
			app.Errorlog.Println("Can't convert to user")
		} else {
			td.User = &user
		}
	}
	td.Now = time.Now()

	return td
}

func (app *Config) IsAuthenticated(r *http.Request) bool {
	return app.Session.Exists(r.Context(), "userID")
}
