package main

import (
	"final-project/data"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

var pageTest = []struct {
	name               string
	url                string
	expectedStatusCode int
	handler            http.HandlerFunc
	SessionData        map[string]any
	expectedHtml       string
}{
	{
		name:               "home",
		url:                "/",
		expectedStatusCode: http.StatusOK,
		handler:            testApp.HomePage,
	},
	{
		name:               "login",
		url:                "/login",
		expectedStatusCode: http.StatusOK,
		handler:            testApp.LoginPage,
		expectedHtml:       `<h1 class="mt-5">Login</h1>`,
	},
	{
		name:               "logout",
		url:                "/logout",
		expectedStatusCode: http.StatusSeeOther,
		handler:            testApp.LoginPage,
		SessionData: map[string]any{
			"userID": 1,
			"user":   data.User{},
		},
	},
}

func Test_Pages(t *testing.T) {
	pathTemplate = "./templates"

	for _, e := range pageTest {

		rr := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", e.url, nil)

		ctx := getCtx(req)
		req = req.WithContext(ctx)

		if len(e.SessionData) > 0 {
			for k, v := range e.SessionData {
				testApp.Session.Put(ctx, k, v)
			}
		}

		e.handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("%s page, failed, expected %d but got %d", e.name, e.expectedStatusCode, rr.Code)
		}

		if len(e.expectedHtml) > 0 {
			html := rr.Body.String()
			if !strings.Contains(html, e.expectedHtml) {
				t.Errorf("failed, expected %v but got %v page name %s", e.expectedHtml, html, e.name)
			}
		}
	}
}

func TestConfig_PostLoginPage(t *testing.T) {
	pathTemplate = "./templates"

	postData := url.Values{
		"email":    {"admin@example.com"},
		"password": {"password"},
	}

	rr := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/login", strings.NewReader(postData.Encode()))

	ctx := getCtx(req)
	req = req.WithContext(ctx)

	handler := http.HandlerFunc(testApp.PostLoginPage)
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusSeeOther {
		t.Errorf("failed, expected %d but got %d", http.StatusSeeOther, rr.Code)
	}

	if !testApp.Session.Exists(ctx, "userID") {
		t.Errorf("failed, expected %v but got %v", false, true)
	}
}

func TestConfig_SubscribeToPlan(t *testing.T) {
	pathTemplate = "./templates"

	rr := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/subscribe?id=1", nil)
	ctx := getCtx(req)
	req = req.WithContext(ctx)

	testApp.Session.Put(ctx, "user", data.User{
		ID:        1,
		Email:     "admin@example.com",
		FirstName: "admin",
		LastName:  "user",
		Active:    1,
	})

	handler := http.HandlerFunc(testApp.SubscribeToPlan)
	handler.ServeHTTP(rr, req)

	testApp.Wait.Wait()

	if rr.Code != http.StatusSeeOther {
		t.Errorf("failed, expected %d but got %d", http.StatusSeeOther, rr.Code)
	}
}
