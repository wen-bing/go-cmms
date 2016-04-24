package main

import (
	"fmt"
	"strconv"

	"github.com/go-humble/router"
	"github.com/steveoc64/formulate"
	"github.com/steveoc64/go-cmms/shared"
	"honnef.co/go/js/dom"
)

// Show a list of machine classes, select one to show the parts for that class
func classSelect(context *router.Context) {

	go func() {
		data := []shared.PartClass{}
		rpcClient.Call("PartRPC.ClassList", Session.Channel, &data)
		BackURL := "/"

		form := formulate.ListForm{}
		form.New("fa-puzzle-piece", "Select Machine Type for Parts List")

		// Define the layout
		form.Column("Name", "Name")
		form.Column("Description", "Descr")

		// Add event handlers
		form.CancelEvent(func(evt dom.Event) {
			evt.PreventDefault()
			Session.Router.Navigate(BackURL)
		})

		if Session.UserRole == "Admin" {
			form.NewRowEvent(func(evt dom.Event) {
				evt.PreventDefault()
				Session.Router.Navigate("/class/add")
			})
		}

		form.RowEvent(func(key string) {
			Session.Router.Navigate("/parts/" + key)
		})

		form.Render("class-select", "main", data)

	}()
}

func classAdd(context *router.Context) {

	go func() {
		partClass := shared.PartClass{}
		BackURL := "/class/select"
		title := "Add Machine Type for Parts List"
		form := formulate.EditForm{}
		form.New("fa-puzzle-piece", title)

		// Layout the fields

		form.Row(2).
			AddInput(1, "Name", "Name")

		form.Row(1).
			AddInput(1, "Description", "Descr")

		// Add event handlers
		form.CancelEvent(func(evt dom.Event) {
			evt.PreventDefault()
			Session.Router.Navigate(BackURL)
		})

		form.SaveEvent(func(evt dom.Event) {
			evt.PreventDefault()
			form.Bind(&partClass)
			go func() {
				data := shared.PartClassUpdateData{
					Channel:   Session.Channel,
					PartClass: &partClass,
				}
				newID := 0
				rpcClient.Call("PartRPC.InsertClass", data, &newID)
				print("added class ID", newID)
				Session.Router.Navigate(BackURL)
			}()
		})

		// All done, so render the form
		form.Render("edit-form", "main", &partClass)

	}()

}

// Show a list of all parts for the given class
func partList(context *router.Context) {

	partClass, _ := strconv.Atoi(context.Params["id"])
	print("show parts of class", partClass)

	go func() {
		data := []shared.Part{}
		req := shared.PartListReq{
			Channel: Session.Channel,
			Class:   partClass,
		}
		class := shared.PartClass{}
		rpcClient.Call("PartRPC.List", req, &data)
		rpcClient.Call("PartRPC.GetClass", partClass, &class)

		BackURL := "/class/select"
		Title := fmt.Sprintf("Parts of type - %s", class.Name)

		// load a form for the class
		if partClass == 0 {
			loadTemplate("class-display", "main", class)
		} else {
			loadTemplate("class-edit", "main", class)
			w := dom.GetWindow()
			doc := w.Document()

			if el := doc.QuerySelector(".data-del-btn"); el != nil {

				if el := doc.QuerySelector(".md-confirm-del"); el != nil {
					el.AddEventListener("click", false, func(evt dom.Event) {
						go func() {
							data := shared.PartClassUpdateData{
								Channel:   Session.Channel,
								PartClass: &class,
							}
							done := false
							rpcClient.Call("PartRPC.DeleteClass", data, &done)
						}()
						Session.Router.Navigate(BackURL)
					})
				}

				el.AddEventListener("click", false, func(evt dom.Event) {
					doc.QuerySelector("#confirm-delete").Class().Add("md-show")
				})

				if el := doc.QuerySelector(".md-close-del"); el != nil {
					el.AddEventListener("click", false, func(evt dom.Event) {
						doc.QuerySelector("#confirm-delete").Class().Remove("md-show")
					})
				}

				if el := doc.QuerySelector("#confirm-delete"); el != nil {
					el.AddEventListener("keyup", false, func(evt dom.Event) {
						if evt.(*dom.KeyboardEvent).KeyCode == 27 {
							evt.PreventDefault()
							doc.QuerySelector("#confirm-delete").Class().Remove("md-show")
						}
					})
				}
			}
		}

		form := formulate.ListForm{}
		form.New("fa-puzzle-piece", Title)

		// Define the layout
		form.Column("Name", "Name")
		form.Column("Description", "Descr")
		form.Column("Stock Code", "StockCode")
		form.Column("Reorder Lvl/Qty", "ReorderDetails")
		form.Column("Qty", "QtyType")
		form.Column("Latest Price", "DisplayPrice")

		// Add event handlers
		form.CancelEvent(func(evt dom.Event) {
			evt.PreventDefault()
			Session.Router.Navigate(BackURL)
		})

		if Session.UserRole == "Admin" {
			form.NewRowEvent(func(evt dom.Event) {
				evt.PreventDefault()
				Session.Router.Navigate(fmt.Sprintf("/part/add/%d", class.ID))
			})
		}

		form.RowEvent(func(key string) {
			Session.Router.Navigate("/part/" + key)
		})

		form.Render("parts-list", "#parts-list-goes-here", data)

		// Add an onChange callback to the class edit fields
		w := dom.GetWindow()
		doc := w.Document()

		doc.QuerySelector("#class-name").AddEventListener("change", false, func(evt dom.Event) {
			print("TODO - Name has changed")
			go func() {
				class.Name = doc.QuerySelector("#class-name").(*dom.HTMLInputElement).Value
				data := shared.PartClassUpdateData{
					Channel:   Session.Channel,
					PartClass: &class,
				}
				done := false
				rpcClient.Call("PartRPC.UpdateClass", data, &done)
			}()
		})
		doc.QuerySelector("#class-descr").AddEventListener("change", false, func(evt dom.Event) {
			print("TODO - Description has changed")
			go func() {
				class.Descr = doc.QuerySelector("#class-descr").(*dom.HTMLInputElement).Value
				data := shared.PartClassUpdateData{
					Channel:   Session.Channel,
					PartClass: &class,
				}
				done := false
				rpcClient.Call("PartRPC.UpdateClass", data, &done)
			}()
		})

	}()
}

func partEdit(context *router.Context) {
	id, err := strconv.Atoi(context.Params["id"])
	if err != nil {
		print(err.Error())
		return
	}

	go func() {
		part := shared.Part{}
		rpcClient.Call("PartRPC.Get", id, &part)

		BackURL := fmt.Sprintf("/parts/%d", part.Class)
		title := fmt.Sprintf("Part Details - %s - %s", part.Name, part.StockCode)
		form := formulate.EditForm{}
		form.New("fa-puzzle-piece", title)

		// Layout the fields

		form.Row(2).
			AddInput(1, "Name", "Name").
			AddInput(1, "Stock Code", "StockCode")

		form.Row(1).
			AddInput(1, "Description", "Descr")

		form.Row(3).
			AddNumber(1, "ReOrder Level", "ReorderStocklevel", "1").
			AddNumber(1, "ReOrder Qty", "ReorderQty", "1").
			AddInput(1, "Qty Type", "QtyType")

		form.Row(1).
			AddTextarea(1, "Notes", "Notes")

		// Add event handlers
		form.CancelEvent(func(evt dom.Event) {
			evt.PreventDefault()
			Session.Router.Navigate(BackURL)
		})

		form.DeleteEvent(func(evt dom.Event) {
			evt.PreventDefault()
			print("TODO - delete part")
			return

			// machine.ID = id
			// go func() {
			// 	data := shared.MachineUpdateData{
			// 		Channel: Session.Channel,
			// 		Machine: &machine,
			// 	}
			// 	done := false
			// 	rpcClient.Call("MachineRPC.Delete", data, &done)
			// 	Session.Router.Navigate(BackURL)
			// }()
		})

		form.SaveEvent(func(evt dom.Event) {
			evt.PreventDefault()
			print("TODO - part save")
			return
			// form.Bind(&machine)
			// go func() {
			// 	data := shared.MachineUpdateData{
			// 		Channel: Session.Channel,
			// 		Machine: &machine,
			// 	}
			// 	done := false
			// 	rpcClient.Call("MachineRPC.Update", data, &done)
			// 	Session.Router.Navigate(BackURL)
			// }()
		})

		// All done, so render the form
		form.Render("edit-form", "main", &part)

		// And attach actions
		form.ActionGrid("part-actions", "#action-grid", part.ID, func(url string) {
			Session.Router.Navigate(url)
		})

	}()
}

func partAdd(context *router.Context) {
	print("TODO partAdd")
}
