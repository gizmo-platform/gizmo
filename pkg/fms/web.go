package fms

import (
	"net/http"
)

func (f *FMS) uiViewLogin(w http.ResponseWriter, r *http.Request) {
	f.doTemplate(w, r, "login.p2", nil)
}

func (f *FMS) uiViewAdminLanding(w http.ResponseWriter, r *http.Request) {
	f.doTemplate(w, r, "views/admin/landing.p2", nil)
}
