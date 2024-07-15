package main

import (
	"net/http"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/v5/middleware"
)

func (app *Config) routes() http.Handler {
	//create a route
	mux := chi.NewRouter()

	//set Middleware
	mux.Use(middleware.Recoverer)
	mux.Use(app.SessionLoad)

	//define applications route
	mux.Get("/", app.HomePage)
	mux.Get("/login", app.LoginPage)
	mux.Post("/login", app.PostLoginPage)
	mux.Get("/logout", app.LogoutPage)

	mux.Get("/register", app.RegisterPage)
	mux.Post("/register", app.PostRegisterPage)
	mux.Get("/activate", app.ActivetedAccount)

	mux.Mount("/members", app.authRouter())

	// mux.Get("/test-email", func(w http.ResponseWriter, r *http.Request) {
	// 	m := Mail{
	// 		Domain:     "localhost",
	// 		Host:       "localhost",
	// 		Port:       1025,
	// 		Encryption: "none",
	// 		FromAdress: "info@myCompany.com",
	// 		FromName:   "info",
	// 		ErrorChan:  make(chan error),
	// 	}
	// 	msg := Message{
	// 		To:      "me@here.com",
	// 		Subject: "Test Email",
	// 		Data:    "This is a test email",
	// 	}

	// 	m.SendEMail(msg, make(chan error))

	// })

	return mux
}

func (app *Config) authRouter() http.Handler {
	mux := chi.NewRouter()
	mux.Use(app.Auth)

	mux.Get("/plans", app.ChooseSubscription)
	mux.Get("/subscribe", app.SubscribeToPlan)
	return mux
}
