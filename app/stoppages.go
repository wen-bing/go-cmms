package main

import (
	"fmt"
	"strconv"

	"github.com/go-humble/router"
	"github.com/steveoc64/formulate"
	"github.com/steveoc64/go-cmms/shared"
	"honnef.co/go/js/dom"
)

func stoppageList(context *router.Context) {

	go func() {
		events := []shared.Event{}
		rpcClient.Call("EventRPC.List", Session.Channel, &events)

		form := formulate.ListForm{}
		form.New("fa-pause-circle-o", "Current Stoppages")

		// Define the layout
		form.Column("Raised By", "Username")
		form.Column("Date", "GetStartDate")
		form.Column("Site", "SiteName")
		form.Column("Machine", "MachineName")
		form.Column("Component", "ToolType")
		form.Column("Notes", "Notes")

		// Add event handlers
		form.CancelEvent(func(evt dom.Event) {
			evt.PreventDefault()
			Session.Router.Navigate("/")
		})

		form.RowEvent(func(key string) {
			Session.Router.Navigate("/stoppage/" + key)
		})

		form.Render("stoppage-list", "main", events)

	}()
}

// Show an edit form for the given stoppage event
func stoppageEdit(context *router.Context) {
	id, err := strconv.Atoi(context.Params["id"])
	if err != nil {
		print(err.Error())
		return
	}

	go func() {
		event := shared.Event{}

		rpcClient.Call("EventRPC.Get", id, &event)

		BackURL := "/stoppages"
		title := fmt.Sprintf("Stoppage Details - %d", id)
		form := formulate.EditForm{}
		form.New("fa-pause-circle-o", title)

		print("and the startdate is ", event.StartDate)
		print("and the startdate is ", event.StartDate.String())
		event.DisplayDate = event.StartDate.String()
		event.DisplayDate = event.StartDate.Format("Mon, Jan 2 2006 15:04:05")

		// Layout the fields
		form.Row(2).
			AddDisplay(1, "Site", "SiteName").
			AddDisplay(1, "Machine", "MachineName")

		form.Row(3).
			AddDisplay(1, "Component", "ToolType").
			AddDisplay(1, "StartDate", "DisplayDate").
			AddDisplay(1, "Raised By", "Username")

		form.Row(1).
			AddTextarea(1, "Notes", "Notes")

		// Add event handlers
		form.CancelEvent(func(evt dom.Event) {
			evt.PreventDefault()
			Session.Router.Navigate(BackURL)
		})

		form.DeleteEvent(func(evt dom.Event) {
			evt.PreventDefault()
			event.ID = id
			go func() {
				data := shared.EventUpdateData{
					Channel: Session.Channel,
					Event:   &event,
				}
				done := false
				rpcClient.Call("EventRPC.Delete", data, &done)
				Session.Router.Navigate(BackURL)
			}()
		})

		form.SaveEvent(func(evt dom.Event) {
			evt.PreventDefault()
			form.Bind(&event)
			data := shared.EventUpdateData{
				Channel: Session.Channel,
				Event:   &event,
			}
			go func() {
				done := false
				rpcClient.Call("EventRPC.Update", data, &done)
				Session.Router.Navigate(BackURL)
			}()
		})

		// All done, so render the form
		form.Render("edit-form", "main", &event)

		// And attach actions
		form.ActionGrid("event-actions", "#action-grid", event.ID, func(url string) {
			Session.Router.Navigate(url)
		})

	}()

}
