package main

import (
	"errors"

	"github.com/go-humble/router"
	"github.com/steveoc64/go-cmms/shared"
	"honnef.co/go/js/dom"
)

func fixLinks() {
	Session.Router.InterceptLinks()
}

// Load a template and attach it to the specified element in the doc
func loadTemplate(template string, selector string, data interface{}) error {
	w := dom.GetWindow()
	doc := w.Document()

	t, err := GetTemplate(template)
	if t == nil {
		print("Failed to load template", template)
		return errors.New("Invalid template")
	}
	if err != nil {
		print(err.Error())
		return err
	}

	el := doc.QuerySelector(selector)
	if el == nil {
		print("Could not find selector", selector)
		return errors.New("Invalid selector")
	}
	if err := t.ExecuteEl(el, data); err != nil {
		print(err.Error())
		return err
	}
	Session.Router.InterceptLinks()
	return nil
}

func enableRoutes(Role string) {

	print("enabling routes for role", Role)

	switch Role {
	case "Admin":
		Session.AppFn = map[string]router.Handler{
			"sitemap":       siteMap,
			"sitemachines":  siteMachines,
			"sites":         siteList,
			"site":          siteEdit,
			"site-machines": siteMachineList,
			"tasks":         taskMaint,
			"stoppages":     stoppagesList,
			"parts":         partsList,
			"users":         usersList,
			"reports":       adminReports,
		}
	case "Site Manager":
		Session.AppFn = map[string]router.Handler{
			"sitemap":       siteMap,
			"sitemachines":  siteMachines,
			"sites":         siteList,
			"site":          siteEdit,
			"site-machines": siteMachineList,
			"tasks":         taskList,
			"stoppages":     stoppagesList,
			"parts":         partsList,
			"users":         usersList,
			"reports":       adminReports,
		}
	case "Worker":
		Session.AppFn = map[string]router.Handler{
			"sitemap":      siteMap,
			"sitemachines": siteMachines,
			"tasks":        taskList,
			"stoppages":    stoppagesList,
			"parts":        partsList,
			"reports":      workerReports,
		}
	case "Floor":
		Session.AppFn = map[string]router.Handler{
			"sitemap":      siteMap,
			"sitemachines": siteMachines,
		}
	}
}

func initRouter() {
	Session.Router = router.New()
	Session.Router.ShouldInterceptLinks = true
	Session.Router.HandleFunc("/", defaultRoute)
	Session.Router.Start()
}

func defaultRoute(context *router.Context) {
	// print("default route")
}

func loadRoutes(Role string, Routes []shared.UserRoute) {

	// print("Loading new routing table")
	enableRoutes(Role)
	if Session.Router != nil {
		Session.Router.Stop()
	}
	Session.Router = router.New()
	Session.Router.ShouldInterceptLinks = true

	for _, v := range Routes {
		if f, ok := Session.AppFn[v.Func]; ok {
			// print("found a function called", v.Func)
			// print("adding route", v.Route, v.Func)
			Session.Router.HandleFunc(v.Route, f)
		}
	}
	Session.Router.Start()
}
