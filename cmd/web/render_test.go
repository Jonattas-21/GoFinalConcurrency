package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestConfig_AddDefaultData(t *testing.T) {
	req, _ := http.NewRequest("GET", "/", nil)
	ctx := getCtx(req)
	req = req.WithContext(ctx)

	testApp.Session.Put(ctx, "flash", "flash")
	testApp.Session.Put(ctx, "warning", "warning")
	testApp.Session.Put(ctx, "error", "error")

	td := testApp.AddDefaultData(&TemplateData{}, req)

	if td.Flash != "flash" {
		t.Error("flash value of 'flash' not found in session")
	}

	if td.Warning != "warning" {
		t.Error("flash value of 'warning' not found in session")
	}

	if td.Error != "error" {
		t.Error("flash value of 'error' not found in session")
	}
}

func TestConfig_IfAuth(t *testing.T) {
	req, _ := http.NewRequest("GET", "/", nil)
	ctx := getCtx(req)
	req = req.WithContext(ctx)

	auth := testApp.IsAuthenticated(req)

	if auth {
		t.Error("user is authenticated when shoud be false")
	}

	testApp.Session.Put(ctx, "userID", 1)
	auth = testApp.IsAuthenticated(req)

	if !auth {
		t.Error("user is not authenticated when shoud be true")
	}
}

func TestConfig_Render(t *testing.T) {
	pathTemplate = "./templates"

	rr := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/", nil)
	ctx := getCtx(req)
	req = req.WithContext(ctx)

	testApp.render(rr, req, "home.page.gohtml", &TemplateData{})

	if rr.Code != http.StatusOK {
		t.Errorf("fail to render page return code %d", rr.Code)
	}
}
