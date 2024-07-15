package main

import (
	"errors"
	"final-project/data"
	"fmt"
	"html/template"
	"net/http"
	"strconv"
	"time"

	"github.com/phpdave11/gofpdf"
	"github.com/phpdave11/gofpdf/contrib/gofpdi"
)

var pathToManual = "./pdf"
var tempPath = "./tmp"

func (app *Config) HomePage(w http.ResponseWriter, r *http.Request) {

	app.render(w, r, "home.page.gohtml", nil)
}

func (app *Config) LoginPage(w http.ResponseWriter, r *http.Request) {

	app.render(w, r, "login.page.gohtml", nil)
}

func (app *Config) PostLoginPage(w http.ResponseWriter, r *http.Request) {

	fmt.Println("PostLoginPage")

	_ = app.Session.RenewToken(r.Context())
	err := r.ParseForm()
	if err != nil {
		app.Errorlog.Println(err)
	}

	//get email and password from the post
	email := r.Form.Get("email")
	password := r.Form.Get("password")

	//get user from the database
	user, err := app.Models.User.GetByEmail(email)
	if err != nil {
		app.Session.Put(r.Context(), "error", "Invalid credentials")
		app.Errorlog.Println(err)
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	//check password
	validPassword, err := app.Models.User.PasswordMatches(password)

	if err != nil {
		app.Session.Put(r.Context(), "error", "Invalid credentials")
		app.Errorlog.Println(err)
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	if !validPassword {
		msg := Message{
			To:      user.Email,
			Subject: "Failed Login Attempt",
			Data:    "Someone has tried to login to your account with the wrong password",
		}
		app.sendemail(msg)

		app.Session.Put(r.Context(), "error", "Invalid credentials")
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	//user login
	app.Session.Put(r.Context(), "userID", user.ID)
	app.Session.Put(r.Context(), "user", user)

	app.Session.Put(r.Context(), "flash", "You have been logged in successfully")

	//redirect the user
	http.Redirect(w, r, "/", http.StatusSeeOther)

	http.Redirect(w, r, "/home", http.StatusSeeOther)
}

func (app *Config) LogoutPage(w http.ResponseWriter, r *http.Request) {
	// clean up the session

	app.Session.Destroy(r.Context())
	app.Session.RenewToken(r.Context())
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

func (app *Config) RegisterPage(w http.ResponseWriter, r *http.Request) {

	app.render(w, r, "register.page.gohtml", nil)
}

func (app *Config) PostRegisterPage(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		app.Errorlog.Println(err)
	}

	//TODO vailidate data

	//create user
	user := data.User{
		Email:     r.Form.Get("email"),
		FirstName: r.Form.Get("first-name"),
		LastName:  r.Form.Get("last-name"),
		Password:  r.Form.Get("password"),
		Active:    0,
		IsAdmin:   0,
	}

	_, err = app.Models.User.Insert(user)
	if err != nil {
		app.Session.Put(r.Context(), "error", "Could not create user")
		app.Errorlog.Println(err)
		http.Redirect(w, r, "/register", http.StatusSeeOther)
	}

	//send email

	url := fmt.Sprintf("http://localhost/activate?email=%s", user.Email)
	signedUrl := GenerateTokenFromString(url)
	app.Infolog.Println(signedUrl)

	msg := Message{
		To:       user.Email,
		Subject:  "Activate your account",
		Template: "confirmation-email",
		Data:     template.HTML(signedUrl),
	}

	app.sendemail(msg)
	app.Session.Put(r.Context(), "flash", "Your account has been created, please check your email to activate it")
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

func (app *Config) ActivetedAccount(w http.ResponseWriter, r *http.Request) {

	//validate the url
	url := r.RequestURI
	testeUrl := fmt.Sprintf("http://localhost%s", url)

	okay := VerifyToken(testeUrl)
	if !okay {
		app.Errorlog.Println("Invalid token")
		app.Session.Put(r.Context(), "error", "Invalid token")
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	//activate account
	u, err := app.Models.User.GetByEmail(r.URL.Query().Get("email"))
	if err != nil {
		app.Errorlog.Println(err)
		app.Session.Put(r.Context(), "error", "No User found")
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	u.Active = 1
	err = app.Models.User.Update(u)
	if err != nil {
		app.Errorlog.Println(err)
		app.Session.Put(r.Context(), "error", "Could not update account")
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	app.Session.Put(r.Context(), "flash", "Your account has been activated, please login")
	http.Redirect(w, r, "/login", http.StatusSeeOther)

}

func (app *Config) ChooseSubscription(w http.ResponseWriter, r *http.Request) {
	plans, err := app.Models.Plan.GetAll()
	if err != nil {
		app.Errorlog.Println(err)
		return
	}
	dataMap := make(map[string]any)
	dataMap["plans"] = plans

	app.render(w, r, "plans.page.gohtml", &TemplateData{Data: dataMap})
}

func (app *Config) SubscribeToPlan(w http.ResponseWriter, r *http.Request) {
	//get id from the plan
	id := r.URL.Query().Get("id")

	planId, err := strconv.Atoi(id)
	if err != nil {
		app.Errorlog.Println(err)
		app.Session.Put(r.Context(), "error", "Invalid Plan")
	}

	//get the plan from db

	plan, err := app.Models.Plan.GetOne(planId)

	if err != nil {
		app.Session.Put(r.Context(), "error", "Could not get plan")
		app.Errorlog.Println(err)
		return
	}

	//get user from session
	user, ok := app.Session.Get(r.Context(), "user").(data.User)

	if !ok {
		app.Session.Put(r.Context(), "error", "You must login first")
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	//gerate an invoice and email it
	app.Wait.Add(1)
	go func() {
		defer app.Wait.Done()

		invoice, err := app.getInvoice(user, plan)
		if err != nil {
			app.Errorlog.Println(err)
			app.ErrorChan <- err
		}

		message := Message{
			To:       user.Email,
			Subject:  "Your Invoice",
			Data:     invoice,
			Template: "invoice",
		}
		app.sendemail(message)
	}()

	//generate a manual and email it
	app.Wait.Add(1)
	go func() {
		defer app.Wait.Done()
		pdf := app.generateManual(user, plan)
		err := pdf.OutputFileAndClose(fmt.Sprintf("%s/%d_manual.pdf", tempPath, user.ID))
		if err != nil {
			app.Errorlog.Println(err)
			app.ErrorChan <- err
			return
		}

		msg := Message{
			To:      user.Email,
			Subject: "Your Manual",
			Data:    "Your user manual is attacted",
			AttachmentsMap: map[string]string{
				"manual.pdf": fmt.Sprintf("%s/%d_manual.pdf", tempPath, user.ID),
			},
		}
		app.sendemail(msg)

		//test app error chan
		app.ErrorChan <- errors.New("Test Error")
	}()

	//subscribe user to an plam

	err = app.Models.Plan.SubscribeUserToPlan(user, *plan)
	if err != nil {
		app.Errorlog.Println(err)
		app.Session.Put(r.Context(), "error", "Error subscriben to plan")
		http.Redirect(w, r, "/members/plans", http.StatusSeeOther)
		return
	}

	u, err := app.Models.User.GetOne(user.ID)
	if err != nil {
		app.Errorlog.Println(err)
		app.Session.Put(r.Context(), "error", "Error getting user from db")
		http.Redirect(w, r, "/members/plans", http.StatusSeeOther)
		return
	}

	app.Session.Put(r.Context(), "user", u)

	//redirect to the home page

	app.Session.Put(r.Context(), "flash", "Subscribed!")
	http.Redirect(w, r, "/members/plans", http.StatusSeeOther)
}

func (app *Config) generateManual(user data.User, plan *data.Plan) *gofpdf.Fpdf {
	pdf := gofpdf.New("P", "mm", "letter", "")

	pdf.SetMargins(10, 13, 10)
	importer := gofpdi.NewImporter()

	time.Sleep(5 * time.Second)
	t := importer.ImportPage(pdf, fmt.Sprintf("%s/manual.pdf", pathToManual), 1, "/MediaBox")
	pdf.AddPage()
	importer.UseImportedTemplate(pdf, t, 0, 0, 215.9, 0)
	pdf.SetX(75)
	pdf.SetY(150)
	pdf.SetFont("Arial", "", 12)
	pdf.MultiCell(0, 4, fmt.Sprintf("%s %s", user.FirstName, user.LastName), "", "C", false)
	pdf.Ln(5)
	pdf.MultiCell(0, 4, fmt.Sprintf("%s user guide", plan.PlanName), "", "C", false)

	return pdf
}

func (app *Config) getInvoice(user data.User, plan *data.Plan) (string, error) {
	return plan.PlanAmountFormatted, nil
}
