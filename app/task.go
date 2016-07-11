package main

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/go-humble/router"
	"github.com/steveoc64/formulate"
	"github.com/steveoc64/go-cmms/shared"
	"honnef.co/go/js/dom"
)

// Add in a pile of constants for lists and things
const (
	rfc3339DateLayout          = "2006-01-02"
	rfc3339DatetimeLocalLayout = "2006-01-02T15:04:05.999999999"
)

func taskMaint(context *router.Context) {
	print("TODO - taskmaint")
}

func calcAllDone(task shared.Task) bool {

	if task.CompletedDate != nil {
		return false
	}

	if task.LabourHrs == 0.0 {
		print("Needs some labour input")
		return false
	}
	for _, v := range task.Checks {
		if !v.Done {
			print("Task has incomplete checklist items")
			return false
		}
	}

	for _, v := range task.Parts {
		// check first if there is an expected Qty
		// if expected Qty == 0, then these are optional parts
		// as the part list was generated from a stoppage
		if v.Qty != 0.0 {
			print("part has qty of", v.Qty)
			if v.QtyUsed == 0 && v.Notes == "" {
				print("Task has an incomplete part record for part", v)
				return false
			}
		}
	}

	print("Task appears to be complete")
	return true
}

func taskEdit(context *router.Context) {
	id, err := strconv.Atoi(context.Params["id"])
	if err != nil {
		print(err.Error())
		return
	}
	Session.ID["task"] = id

	Session.Subscribe("task", _taskEdit)
	go _taskEdit("edit", id)
}

func _taskEdit(action string, id int) {

	BackURL := "/tasks"

	switch action {
	case "edit":
		print("manually edit")
	case "delete":
		if id != Session.ID["task"] {
			return
		}
		print("current record has been deleted")
		Session.Navigate(BackURL)
		return
	default:
		if id != Session.ID["task"] {
			return
		}
	}

	task := shared.Task{}

	rpcClient.Call("TaskRPC.Get", shared.TaskRPCData{
		Channel: Session.Channel,
		ID:      id,
	}, &task)

	task.AllDone = calcAllDone(task)
	// print("task with parts and checks attached =", task)

	RefreshURL := fmt.Sprintf("/task/%d", id)
	title := fmt.Sprintf("Task Details - %06d", id)
	form := formulate.EditForm{}
	form.New("fa-server", title)

	if task.StartDate != nil {
		task.DisplayStartDate = task.StartDate.Format("Mon, Jan 2 2006")
	}
	if task.DueDate != nil {
		task.DisplayDueDate = task.DueDate.Format("Mon, Jan 2 2006")
	}
	if task.Username == nil {
		task.DisplayUsername = "Unassigned"
	} else {
		task.DisplayUsername = *task.Username
	}

	// Define a function to add the actions and set all the callbacks
	setActions := func(actionID int) {

		switch Session.UserRole {
		case "Admin", "Site Manager":
			form.ActionGrid("task-admin-actions", "#action-grid", task, func(url string) {
				if strings.HasPrefix(url, "/sched") {
					if task.SchedID != 0 {
						c := &router.Context{
							Path:        url,
							InitialLoad: false,
							Params: map[string]string{
								"id":   fmt.Sprintf("%d", task.SchedID),
								"back": fmt.Sprintf("/task/%d", task.ID),
							},
						}
						schedEdit(c)
					}
				} else if strings.HasPrefix(url, "/stoppage") {
					if task.EventID != 0 {
						c := &router.Context{
							Path:        url,
							InitialLoad: false,
							Params: map[string]string{
								"id":   fmt.Sprintf("%d", task.EventID),
								"back": fmt.Sprintf("/task/%d", task.ID),
							},
						}
						stoppageEdit(c)
					}
				} else {
					go func() {
						done := false
						rpcClient.Call("TaskRPC.Complete", shared.TaskRPCData{
							Channel: Session.Channel,
							Task:    &task,
						}, &done)
						Session.Navigate(BackURL)
					}()
				}
			})
		case "Technician":
			form.ActionGrid("task-actions", "#action-grid", task, func(url string) {
				go func() {
					done := false
					print("calling task.complete ???")
					rpcClient.Call("TaskRPC.Complete", shared.TaskRPCData{
						Channel: Session.Channel,
						Task:    &task,
					}, &done)
					Session.Navigate(BackURL)
				}()
			})
		}
	}

	// Layout the fields
	partsTitle := ""
	if task.SchedID != 0 {
		partsTitle = "Parts Used - as specified on the maintenance schedule - all parts must have a Qty Used, or a note"
	} else {
		partsTitle = "Parts Used - record qty for each part used - or leave blank if part was not needed on this job"
	}

	useRole := Session.UserRole
	if Session.CanAllocate {
		useRole = "Admin"
	}

	switch useRole {
	case "Admin":

		techs := []shared.User{}
		rpcClient.Call("UserRPC.GetTechnicians", shared.SiteRPCData{
			Channel: Session.Channel,
			ID:      0,
		}, &techs)
		assignedTo := 0
		if task.AssignedTo != nil {
			assignedTo = *task.AssignedTo
		}

		form.Row(3).
			AddSelect(1, "User", "AssignedTo", techs, "ID", "Username", 0, assignedTo).
			AddDisplay(1, "Start Date", "DisplayStartDate").
			AddDisplay(1, "Due Date", "DisplayDueDate")

		form.Row(3).
			AddDisplay(1, "Site", "SiteName").
			AddDisplay(1, "Machine", "MachineName").
			AddDisplay(1, "Component", "Component")

		form.Row(1).
			AddTextarea(1, "Notes", "Log")

		form.Row(1).
			AddCustom(1, "Notes and CheckLists", "CheckList", "")

		rateStr := ""
		if task.LabourHrs != 0 {
			rate := int(task.LabourCost / task.LabourHrs)
			rateStr = fmt.Sprintf(" @ $%d/hr", rate)
		}
		form.Row(4).
			AddDisplay(2, "Labour Est $", "LabourEst").
			AddDecimal(1, fmt.Sprintf("Hours%s", rateStr), "LabourHrs", 2, "0.5").
			AddDecimal(1, "Actual Labour $", "LabourCost", 2, "1")

		form.Row(4).
			AddDisplay(2, "Material Est $", "MaterialEst").
			AddDecimal(2, "Actual Material $", "MaterialCost", 2, "1")

		form.Row(1).
			AddCustom(1, partsTitle, "PartList", "")

	case "Site Manager":
		form.Row(3).
			AddDisplay(1, "User", "DisplayUsername").
			AddDisplay(1, "Start Date", "DisplayStartDate").
			AddDisplay(1, "Due Date", "DisplayDueDate")

		form.Row(3).
			AddDisplay(1, "Site", "SiteName").
			AddDisplay(1, "Machine", "MachineName").
			AddDisplay(1, "Component", "Component")

		form.Row(1).
			AddDisplayArea(1, "Notes", "Log")

		form.Row(1).
			AddCustom(1, "Notes and CheckLists", "CheckList", "")

		form.Row(4).
			AddDisplay(2, "Labour Est $", "LabourEst").
			AddDecimal(1, "Hours", "LabourHrs", 2, "0.5").
			AddDecimal(1, "Actual Labour $", "LabourCost", 2, "1")

		form.Row(4).
			AddDisplay(2, "Material Est $", "MaterialEst").
			AddDecimal(2, "Actual Material $", "MaterialCost", 2, "1")

		form.Row(1).
			AddCustom(1, partsTitle, "PartList", "")

	case "Technician":
		row := form.Row(4).
			AddDisplay(1, "Start Date", "DisplayStartDate").
			AddDisplay(1, "Due Date", "DisplayDueDate").
			// AddDecimal(1, "Actual Material $", "MaterialCost", 2, "1").
			// AddDecimal(1, "Actual Labour $", "LabourCost", 2, "1").
			AddDisplay(1, "Actual Material $", "MaterialCost")
			// AddDisplay(1, "Actual Labour $", "LabourCost")

		if task.CompletedDate == nil {
			row.AddDecimal(1, "Hours", "LabourHrs", 2, "0.5")
		} else {
			row.AddDisplay(1, "Hours", "LabourHrs")
		}

		form.Row(3).
			AddDisplay(1, "Site", "SiteName").
			AddDisplay(1, "Machine", "MachineName").
			AddDisplay(1, "Component", "Component")

		if task.CompletedDate == nil {
			form.Row(1).
				AddTextarea(1, "Notes", "Log")
		} else {
			form.Row(1).
				AddDisplayArea(1, "Notes", "Log")
		}

		form.Row(1).
			AddCustom(1, "Notes and CheckLists", "CheckList", "")

		form.Row(1).
			AddCustom(1, partsTitle, "PartList", "")
	}

	// Add event handlers
	form.CancelEvent(func(evt dom.Event) {
		evt.PreventDefault()
		Session.Navigate(BackURL)
	})

	form.PrintEvent(func(evt dom.Event) {
		dom.GetWindow().Print()
	})

	if useRole == "Admin" {
		form.DeleteEvent(func(evt dom.Event) {
			evt.PreventDefault()
			task.ID = id
			go func() {
				done := false
				rpcClient.Call("TaskRPC.Delete", shared.TaskRPCData{
					Channel: Session.Channel,
					Task:    &task,
				}, &done)
				Session.Navigate(BackURL)
			}()
		})
	}

	print("useRole =", useRole)
	if useRole == "Admin" ||
		(Session.UserRole == "Technician" && task.CompletedDate == nil) {

		form.SaveEvent(func(evt dom.Event) {
			evt.PreventDefault()
			form.Bind(&task)

			// w := dom.GetWindow()
			// doc := w.Document()

			// // now get the parts array
			// for i, v := range task.Parts {
			// 	qtyUsed := doc.QuerySelector(fmt.Sprintf("[name=part-qty-used-%d]", v.PartID)).(*dom.HTMLInputElement)
			// 	notes := doc.QuerySelector(fmt.Sprintf("[name=part-notes-%d]", v.PartID)).(*dom.HTMLInputElement)
			// 	// print("Part ", v.PartID, "QtyUsed = ", qtyUsed.Value)
			// 	// print("Part ", v.PartID, "Notes = ", notes.Value)
			// 	task.Parts[i].QtyUsed, _ = strconv.ParseFloat(qtyUsed.Value, 64)
			// 	task.Parts[i].Notes = notes.Value
			// }

			go func() {
				updatedTask := shared.Task{}
				rpcClient.Call("TaskRPC.Update", shared.TaskRPCData{
					Channel: Session.Channel,
					Task:    &task,
				}, &updatedTask)
				Session.Navigate(RefreshURL)
			}()
		})

	}

	// All done, so render the form
	form.Render("edit-form", "main", &task)

	// Add the custom checklist
	loadTemplate("task-check-list", "[name=CheckList]", task)
	loadTemplate("task-parts-tree", "[name=PartList]", task)

	w := dom.GetWindow()
	doc := w.Document()

	// on change of the labour hrs, update the all done flag
	if useRole == "Admin" || task.CompletedDate == nil {
		lh := doc.QuerySelector("[name=LabourHrs]").(*dom.HTMLInputElement)
		lh.AddEventListener("change", false, func(evt dom.Event) {
			wasDone := task.AllDone
			task.LabourHrs, _ = strconv.ParseFloat(lh.Value, 64)
			// fire off the change to the backend
			form.Bind(&task)
			go func() {
				updatedTask := shared.Task{}
				rpcClient.Call("TaskRPC.Update", shared.TaskRPCData{
					Channel: Session.Channel,
					Task:    &task,
				}, &updatedTask)
				if Session.UserRole == "Admin" {
					w := dom.GetWindow()
					doc := w.Document()
					doc.QuerySelector("[name=LabourCost").(*dom.HTMLInputElement).Value = fmt.Sprintf("%.2f", updatedTask.LabourCost)
					print("updated =", updatedTask.LabourCost)
				}
			}()

			task.AllDone = calcAllDone(task)
			if wasDone != task.AllDone {
				setActions(2)
			}
		})
	}

	// click on the parts button, expand the div to show a tree
	if el := doc.QuerySelector("[name=parts-button]"); el != nil {
		el.AddEventListener("click", false, func(evt dom.Event) {
			evt.PreventDefault()
			print("show parts tree here")

			t := doc.QuerySelector(`[name="parts-tree-div"]`)
			t.SetInnerHTML("") // Init the tree panel

			// Create the Tree's UL element
			ul := doc.CreateElement("ul").(*dom.HTMLUListElement)
			ul.SetClass("css-treeview")

			// Fetch the complete parts tree from the backend
			go func() {
				tree := []shared.Category{}
				rpcClient.Call("PartRPC.GetTree", shared.PartTreeRPCData{
					Channel:    Session.Channel,
					CategoryID: 0,
				}, &tree)
				print("got tree", tree)

				// Recursively add elements to the tree
				addTaskPartsTree(tree, ul, 0)

				t.AppendChild(ul)
			}()
		})
	}

	if el := doc.QuerySelector("[name=CheckList]"); el != nil {

		el.AddEventListener("click", false, func(evt dom.Event) {
			evt.PreventDefault()
			clickedOn := evt.Target()
			switch clickedOn.TagName() {
			case "INPUT":
				ie := clickedOn.(*dom.HTMLInputElement)
				key, _ := strconv.Atoi(ie.GetAttribute("key"))
				// print("clicked on key", key)

				task.Checks[key-1].Done = true
				now := time.Now()
				task.Checks[key-1].DoneDate = &now
				task.AllDone = calcAllDone(task)

				// save to the backend
				go func() {
					done := false
					rpcClient.Call("TaskRPC.Check", shared.TaskCheckUpdate{
						Channel:   Session.Channel,
						TaskCheck: &task.Checks[key-1],
					}, &done)
				}()
				loadTemplate("task-check-list", "[name=CheckList]", task)
				setActions(3)
			}

		})
	}

	// And attach actions

	setActions(1)
}

func taskList(context *router.Context) {
	Session.Subscribe("task", _taskList)
	go _taskList("list", 0)
}

// Show a list of all tasks
func _taskList(action string, id int) {

	tasks := []shared.Task{}
	rpcClient.Call("TaskRPC.List", Session.Channel, &tasks)

	form := formulate.ListForm{}
	form.New("fa-server", "Task List - All Active Tasks")

	// Define the layout
	switch Session.UserRole {
	case "Admin", "Site Manager":
		form.Column("User", "Username")
		form.BoolColumn("Read", "IsRead")
	}
	form.Column("TaskID", "GetID")
	form.Column("Date", "GetStartDate")
	// form.Column("Due", "GetDueDate")
	form.Column("Site", "SiteName")
	form.Column("Machine", "MachineName")
	form.Column("Component", "Component")
	form.Column("Description", "Descr")
	form.Column("Duration", "DurationDays")
	form.Column("Hrs", "LabourHrs")

	// Add event handlers
	form.CancelEvent(func(evt dom.Event) {
		evt.PreventDefault()
		Session.Navigate("/")
	})

	form.RowEvent(func(key string) {
		Session.Navigate("/task/" + key)
	})

	form.PrintEvent(func(evt dom.Event) {
		dom.GetWindow().Print()
	})

	form.Render("task-list", "main", tasks)

	ctasks := []shared.Task{}
	rpcClient.Call("TaskRPC.ListCompleted", Session.Channel, &ctasks)

	cform := formulate.ListForm{}
	cform.New("fa-server", "Completed Tasks")

	// Define the layout
	switch Session.UserRole {
	case "Admin", "Site Manager":
		cform.Column("User", "Username")
	}
	cform.Column("TaskID", "GetID")
	cform.Column("Date", "GetStartDate")
	// form.Column("Due", "GetDueDate")
	cform.Column("Site", "SiteName")
	cform.Column("Machine", "MachineName")
	cform.Column("Component", "Component")
	cform.Column("Description", "Descr")
	cform.Column("Duration", "DurationDays")

	if Session.UserRole == "Admin" {
		cform.Column("Hrs", "GetLabour")

	} else {
		cform.Column("Hrs", "LabourHrs")

	}
	// cform.Column("Completed", "CompletedDate")
	cform.Column("Completed", "GetCompletedDate")

	cform.RowEvent(func(key string) {
		Session.Navigate("/task/" + key)
	})

	w := dom.GetWindow()
	doc := w.Document()

	// force a page break for printing
	div := doc.CreateElement("div")
	div.Class().Add("page-break")
	doc.QuerySelector("main").AppendChild(div)

	div = doc.CreateElement("div").(*dom.HTMLDivElement)
	div.SetID("ctasks")
	doc.QuerySelector("main").AppendChild(div)

	cform.Render("task-clist", "#ctasks", ctasks)

}

type MachineSchedListData struct {
	Machine shared.Machine
	Tasks   []shared.SchedTask
}

// List all scheduled maint tasks that include this hashtag
func hashtagUsed(context *router.Context) {
	id, err := strconv.Atoi(context.Params["id"])
	if err != nil {
		print(err.Error())
		return
	}

	go func() {
		hashtag := shared.Hashtag{}
		tasks := []shared.SchedTask{}
		data := shared.HashtagRPCData{
			Channel: Session.Channel,
			ID:      id,
		}
		rpcClient.Call("TaskRPC.HashtagGet", data, &hashtag)
		rpcClient.Call("TaskRPC.ListHashSched", data, &tasks)

		BackURL := fmt.Sprintf("/hashtag/%d", id)

		form := formulate.ListForm{}
		form.New("fa-wrench", "Sched Maint that includes #"+hashtag.Name)

		// Define the layout
		form.Column("Tool / Component", "Component")
		form.Column("Frequency", "ShowFrequency")
		form.Column("Description", "Descr")
		form.Column("$ Labour", "LabourCost")
		form.Column("$ Materials", "MaterialCost")
		form.Column("Duration", "DurationDays")
		form.Column("", "ShowPaused")

		// Add event handlers
		form.CancelEvent(func(evt dom.Event) {
			evt.PreventDefault()
			Session.Navigate(BackURL)
		})

		form.RowEvent(func(key string) {
			Session.Navigate("/sched/" + key)
		})

		form.Render("hash-sched-list", "main", tasks)
	}()
}

// Show a list of all Scheduled Maint items for this machine
func machineSchedList(context *router.Context) {

	id, err := strconv.Atoi(context.Params["machine"])
	if err != nil {
		print(err.Error())
		return
	}

	go func() {
		machine := shared.Machine{}
		tasks := []shared.SchedTask{}
		data := shared.MachineRPCData{
			Channel: Session.Channel,
			ID:      id,
		}
		rpcClient.Call("MachineRPC.Get", data, &machine)
		rpcClient.Call("TaskRPC.ListMachineSched", data, &tasks)

		BackURL := fmt.Sprintf("/machine/%d", id)

		form := formulate.ListForm{}
		form.New("fa-wrench", "Sched Maint for - "+machine.Name+" - "+*machine.SiteName)

		// Define the layout
		form.Column("Tool / Component", "Component")
		form.Column("Frequency", "ShowFrequency")
		form.Column("Description", "Descr")
		form.Column("$ Labour", "LabourCost")
		form.Column("$ Materials", "MaterialCost")
		form.Column("Duration", "DurationDays")
		form.Column("", "ShowPaused")

		// Add event handlers
		form.CancelEvent(func(evt dom.Event) {
			evt.PreventDefault()
			Session.Navigate(BackURL)
		})

		form.NewRowEvent(func(evt dom.Event) {
			evt.PreventDefault()
			Session.Navigate(fmt.Sprintf("/machine/sched/add/%d", id))
		})

		form.PrintEvent(func(evt dom.Event) {
			dom.GetWindow().Print()
		})

		form.RowEvent(func(key string) {
			Session.Navigate("/sched/" + key)
		})

		form.Render("machine-sched-list", "main", tasks)
	}()
}

func schedEdit(context *router.Context) {
	id, err := strconv.Atoi(context.Params["id"])
	if err != nil {
		print(err.Error())
		return
	}

	freqs := []formulate.SelectOption{
		{1, "Monthly"},
		{2, "Yearly"},
		{3, "Every N Days"},
		{4, "One Off"},
		{5, "Job Count"},
	}

	weeks := []formulate.SelectOption{
		{1, "1st Week"},
		{2, "2nd Week"},
		{3, "3rd Week"},
		{4, "4th Week"},
	}

	weekdays := []formulate.SelectOption{
		{1, "Mon"},
		{2, "Tue"},
		{3, "Wed"},
		{4, "Thur"},
		{5, "Fri"},
	}

	go func() {
		machine := shared.Machine{}
		task := shared.SchedTask{}
		technicians := []shared.User{}

		rpcClient.Call("TaskRPC.GetSched", shared.TaskRPCData{
			Channel: Session.Channel,
			ID:      id,
		}, &task)

		rpcClient.Call("MachineRPC.Get", shared.MachineRPCData{
			Channel: Session.Channel,
			ID:      task.MachineID,
		}, &machine)

		rpcClient.Call("UserRPC.GetTechnicians", shared.UserRPCData{
			Channel: Session.Channel,
			ID:      machine.SiteID,
		}, &technicians)

		BackURL := context.Params["back"]
		if BackURL == "" {
			BackURL = fmt.Sprintf("/machine/sched/%d", machine.ID)
		}
		plainTitle := fmt.Sprintf("Sched Maint Task for - %s - %s", machine.Name, *machine.SiteName)
		title := plainTitle
		if task.Paused {
			title += " (PAUSED)"
		}

		form := formulate.EditForm{}
		form.New("fa-wrench", title)

		// create the swapper panels
		swapper := formulate.Swapper{
			Name:     "freq",
			Selected: 1,
		}

		// Add a set of swappable panels for freq options
		theWeek := 1
		if task.Week != nil {
			theWeek = *task.Week
		}
		theWeekDay := 1
		if task.WeekDay != nil {
			theWeekDay = *task.WeekDay
		}
		swapper.AddPanel("week").AddRow(2).
			AddRadio(1, "Week of the Month", "Week", weeks, "ID", "Name", theWeek).
			AddRadio(1, "Weekday", "WeekDay", weekdays, "ID", "Name", theWeekDay)

		swapper.AddPanel("year").AddRow(1).AddDate(1, "Day of the Year", "StartDate")
		swapper.AddPanel("days").AddRow(1).AddNumber(1, "Number of Days", "Days", "1")
		swapper.AddPanel("oneoff").AddRow(1).AddDate(1, "One Off Date", "OneOffDate")
		swapper.AddPanel("job").AddRow(1).AddNumber(1, "Job Count", "Count", "1")

		// Layout the fields
		currentFreq := 0
		for _, f := range freqs {
			if f.Name == task.Freq {
				currentFreq = f.ID
				break
			}
		}
		form.Row(3).
			AddRadio(1, "Frequency", "Freq", freqs, "ID", "Name", currentFreq).
			AddSwapper(2, "Frequency Options:", &swapper)

		compGen := []formulate.SelectOption{
			{0, "General Maintenance"},
		}

		compTools := []formulate.SelectOption{}
		for i, comp := range machine.Components {
			newOpt := formulate.SelectOption{ID: i + 1, Name: comp.Name}
			compTools = append(compTools, newOpt)
		}

		compOther := []formulate.SelectOption{
			{100, "RollBed"},
			{101, "Uncoiler"},
			{102, "Electrical"},
			{103, "Hydraulic"},
			{104, "Lube"},
			{105, "Printer"},
			{106, "Console"},
		}

		currentComp := 0
		switch task.CompType {
		case "T":
			for i, tool := range machine.Components {
				if tool.ID == task.ToolID {
					currentComp = i + 1
					break
				}
			}
		case "C":
			for i, c := range compOther {
				if c.Name == task.Component {
					currentComp = i + 100
				}
			}
		}
		form.Row(2).
			AddGroupedSelect(1,
				"Component", "Component",
				[]formulate.SelectGroup{
					{"", compGen},
					{"Tools", compTools},
					{"Other Components", compOther},
				},
				currentComp).
			AddSelect(1, "Assign To Technician", "UserID", technicians, "ID", "Username", 0, task.UserID)

		form.Row(1).
			AddTextarea(1, "Task Description", "Descr")

		form.Row(3).
			AddDecimal(1, "Labour Cost", "LabourCost", 2, "1").
			AddDecimal(1, "Material Cost", "MaterialCost", 2, "1").
			AddNumber(1, "Duration (days)", "DurationDays", "1")

		// Add a DIV that we can attach panels to
		form.Row(1).
			AddCustom(1, "Parts Required", "PartsPicker", "")

		// Add event handlers
		form.CancelEvent(func(evt dom.Event) {
			evt.PreventDefault()
			Session.Navigate(BackURL)
		})

		form.DeleteEvent(func(evt dom.Event) {
			evt.PreventDefault()
			go func() {
				done := false
				rpcClient.Call("TaskRPC.DeleteSched", shared.SchedTaskRPCData{
					Channel:   Session.Channel,
					SchedTask: &task,
				}, &done)
				Session.Navigate(BackURL)
			}()
		})

		form.PrintEvent(func(evt dom.Event) {
			dom.GetWindow().Print()
		})

		form.SaveEvent(func(evt dom.Event) {
			evt.PreventDefault()
			form.Bind(&task)
			task.MachineID = machine.ID

			// interpret the Component from the grouped options
			comp, _ := strconv.Atoi(task.Component)
			// print("comp = ", comp)
			if comp == 0 {
				task.CompType = "A"
				task.Component = compGen[0].Name
			} else if comp < len(machine.Components) {
				task.CompType = "T"
				task.ToolID = machine.Components[comp-1].ID
				task.Component = compTools[comp-1].Name
			} else {
				task.CompType = "C"
				offset := comp - len(machine.Components)
				task.Component = compOther[offset-1].Name
			}

			// convert the selected freq into a meaningful string
			targetFreq, _ := strconv.Atoi(task.Freq)
			for _, f := range freqs {
				if f.ID == targetFreq {
					task.Freq = f.Name
					break
				}
			}
			switch targetFreq {
			case 1:
				task.Days = nil
				task.Count = nil
				task.StartDate = nil
				task.OneOffDate = nil
			case 2:
				task.Days = nil
				task.Count = nil
				task.Week = nil
				task.OneOffDate = nil
			case 3:
				task.Week = nil
				task.Count = nil
				task.StartDate = nil
				task.OneOffDate = nil
			case 4:
				task.Days = nil
				task.Count = nil
				task.StartDate = nil
				task.Week = nil
			case 5:
				task.Days = nil
				task.Week = nil
				task.StartDate = nil
				task.OneOffDate = nil
			}

			go func() {
				done := false
				rpcClient.Call("TaskRPC.UpdateSched", shared.SchedTaskRPCData{
					Channel:   Session.Channel,
					SchedTask: &task,
				}, &done)
				Session.Navigate(BackURL)
			}()
		})

		// All done, so render the form
		form.Render("edit-form", "main", &task)

		// and set the swap panel to match the data
		for i, f := range freqs {
			if f.Name == task.Freq {
				swapper.Select(i)
				break
			}
		}

		// Setup a callback on the freq selector
		w := dom.GetWindow()
		doc := w.Document()

		doc.QuerySelector("[name=radio-Freq]").AddEventListener("click", false, func(evt dom.Event) {
			clickedOn := evt.Target()
			switch clickedOn.TagName() {
			case "INPUT":
				ie := clickedOn.(*dom.HTMLInputElement)
				key, _ := strconv.Atoi(ie.Value)
				switch key {
				case 1:
					swapper.SelectByName("week")
				case 2:
					swapper.SelectByName("year")
				case 3:
					swapper.SelectByName("days")
				case 4:
					swapper.SelectByName("oneoff")
				case 5:
					swapper.SelectByName("job")
				}
			}
		})

		// Plug in the PartsPicker widget
		loadTemplate("parts-picker", "[name=PartsPicker]", task)

		if el := doc.QuerySelector("[name=PartsPicker]"); el != nil {

			el.AddEventListener("click", false, func(evt dom.Event) {
				evt.PreventDefault()
				clickedOn := evt.Target()
				switch clickedOn.TagName() {
				case "INPUT":
					ie := clickedOn.(*dom.HTMLInputElement)
					key, _ := strconv.Atoi(ie.GetAttribute("key"))
					// print("clicked on key", key)

					// Get the selected partreq from the task
					for i, v := range task.PartsRequired {
						if v.PartID == key {
							// popup a dialog to edit the relationship

							req := shared.PartReqEdit{
								Channel: Session.Channel,
								Task:    task,
								Part:    &task.PartsRequired[i],
							}

							// print("which has a part of ", v)
							loadTemplate("edit-part-req", "#edit-part-req", req)
							doc.QuerySelector("#edit-part-req").Class().Add("md-show")
							doc.QuerySelector("#partreq-qty").(*dom.HTMLInputElement).Focus()

							// Cancel the popup
							doc.QuerySelector(".md-close").AddEventListener("click", false, func(evt dom.Event) {
								// print("Cancel Editing the part req")
								doc.QuerySelector("#edit-part-req").Class().Remove("md-show")
							})

							el.AddEventListener("keyup", false, func(evt dom.Event) {
								if evt.(*dom.KeyboardEvent).KeyCode == 27 {
									evt.PreventDefault()
									// print("Esc out of dialog")
									doc.QuerySelector("#edit-part-req").Class().Remove("md-show")
								}
							})

							// Save the popup
							doc.QuerySelector(".md-save").AddEventListener("click", false, func(evt dom.Event) {
								evt.PreventDefault()
								doc.QuerySelector("#edit-part-req").Class().Remove("md-show")
								// print("Save the part req")

								qty, _ := strconv.ParseFloat(doc.QuerySelector("#partreq-qty").(*dom.HTMLInputElement).Value, 64)
								notes := doc.QuerySelector("#partreq-notes").(*dom.HTMLTextAreaElement).Value

								// print("in save, req still =", req)
								req.Part.Qty = qty
								req.Part.Notes = notes

								// print("qty =", qty, "notes =", notes)

								if qty > 0.0 {
									ie.Checked = true
								} else {
									ie.Checked = false
								}

								go func() {
									done := false
									rpcClient.Call("TaskRPC.SchedPart", req, &done)
									// print("updated at backend")
								}()
							})
							break // dont need to look at the rest
						}
					}

				}

			})
		}

		// Add some action buttons for this schedule
		form.ActionGrid("sched-actions", "#action-grid", task, func(url string) {
			done := false
			switch url {
			case "play":
				go rpcClient.Call("TaskRPC.SchedPlay", shared.SchedTaskRPCData{
					Channel: Session.Channel,
					ID:      task.ID,
				}, &done)
				task.Paused = false
				doc.QuerySelector("#playtask").Class().Add("action-hidden")
				doc.QuerySelector("#pausetask").Class().Remove("action-hidden")
				task.Paused = false
				title := plainTitle
				if task.Paused {
					title += " (PAUSED)"
				}
				form.SetTitle(title)

			case "pause":
				go rpcClient.Call("TaskRPC.SchedPause", shared.SchedTaskRPCData{
					Channel: Session.Channel,
					ID:      task.ID,
				}, &done)
				task.Paused = true
				doc.QuerySelector("#pausetask").Class().Add("action-hidden")
				doc.QuerySelector("#playtask").Class().Remove("action-hidden")
				title := plainTitle
				if task.Paused {
					title += " (PAUSED)"
				}
				form.SetTitle(title)
			default:
				Session.Navigate(url)
			}
		})

		// Set the initial vis of the action items
		// print("paused =", task.Paused)
		if task.Paused {
			doc.QuerySelector("#playtask").Class().Remove("action-hidden")
		} else {
			doc.QuerySelector("#pausetask").Class().Remove("action-hidden")
		}

	}()

}

func machineSchedAdd(context *router.Context) {
	id, err := strconv.Atoi(context.Params["machine"])
	if err != nil {
		print(err.Error())
		return
	}

	freqs := []formulate.SelectOption{
		{1, "Monthly"},
		{2, "Yearly"},
		{3, "Every N Days"},
		{4, "One Off"},
		{5, "Job Count"},
	}

	weeks := []formulate.SelectOption{
		{1, "1st Week"},
		{2, "2nd Week"},
		{3, "3rd Week"},
		{4, "4th Week"},
	}

	weekdays := []formulate.SelectOption{
		{1, "Mon"},
		{2, "Tue"},
		{3, "Wed"},
		{4, "Thur"},
		{5, "Fri"},
	}

	go func() {
		machine := shared.Machine{}
		task := shared.SchedTask{}
		technicians := []shared.User{}
		rpcClient.Call("MachineRPC.Get", shared.MachineRPCData{
			Channel: Session.Channel,
			ID:      id,
		}, &machine)
		rpcClient.Call("UserRPC.GetTechnicians", shared.SiteRPCData{
			Channel: Session.Channel,
			ID:      machine.SiteID,
		}, &technicians)

		BackURL := fmt.Sprintf("/machine/sched/%d", machine.ID)
		title := fmt.Sprintf("Add Sched Maint Task for - %s - %s", machine.Name, *machine.SiteName)

		form := formulate.EditForm{}
		form.New("fa-wrench", title)

		// create the swapper panels
		swapper := formulate.Swapper{
			Name:     "freq",
			Selected: 1,
		}

		// Add a set of swappable panels for freq options
		swapper.AddPanel("week").AddRow(2).
			AddRadio(1, "Week of the Month", "Week", weeks, "ID", "Name", 1).
			AddRadio(1, "Weekday", "WeekDay", weekdays, "ID", "Name", 1)
		swapper.AddPanel("year").AddRow(1).AddDate(1, "Day of the Year", "StartDate")
		swapper.AddPanel("days").AddRow(1).AddNumber(1, "Number of Days", "Days", "1")
		swapper.AddPanel("oneoff").AddRow(1).AddDate(1, "One Off Date", "OneOffDate")
		swapper.AddPanel("job").AddRow(1).AddNumber(1, "Job Count", "Count", "1")

		// Layout the fields
		form.Row(3).
			AddRadio(1, "Frequency", "Freq", freqs, "ID", "Name", 1).
			AddSwapper(2, "Frequency Options:", &swapper)

		compGen := []formulate.SelectOption{
			{0, "General Maintenance"},
		}

		compTools := []formulate.SelectOption{}
		for i, comp := range machine.Components {
			newOpt := formulate.SelectOption{ID: i + 1, Name: comp.Name}
			compTools = append(compTools, newOpt)
		}

		compOther := []formulate.SelectOption{
			{100, "RollBed"},
			{101, "Uncoiler"},
			{102, "Electrical"},
			{103, "Hydraulic"},
			{104, "Lube"},
			{105, "Printer"},
			{106, "Console"},
		}

		form.Row(2).
			AddGroupedSelect(1,
				"Component", "Component",
				[]formulate.SelectGroup{
					{"", compGen},
					{"Tools", compTools},
					{"Other Components", compOther},
				},
				0).
			AddSelect(1, "Assign To Technician", "UserID", technicians, "ID", "Username", 0, 0)

		form.Row(1).
			AddTextarea(1, "Task Description", "Descr")

		form.Row(3).
			AddDecimal(1, "Labour Cost", "LabourCost", 2, "1").
			AddDecimal(1, "Material Cost", "MaterialCost", 2, "1").
			AddNumber(1, "Duration (days)", "DurationDays", "1")

		// Add event handlers
		form.CancelEvent(func(evt dom.Event) {
			evt.PreventDefault()
			Session.Navigate(BackURL)
		})

		form.SaveEvent(func(evt dom.Event) {
			evt.PreventDefault()
			form.Bind(&task)
			task.MachineID = machine.ID

			// interpret the Component from the grouped options
			comp, _ := strconv.Atoi(task.Component)
			// print("comp = ", comp)
			if comp == 0 {
				task.CompType = "A"
				task.Component = compGen[0].Name
			} else if comp < len(machine.Components) {
				task.CompType = "T"
				task.ToolID = machine.Components[comp-1].ID
				task.Component = compTools[comp-1].Name
			} else {
				task.CompType = "C"
				offset := comp - len(machine.Components)
				task.Component = compOther[offset-1].Name
			}

			// convert the selected freq into a meaningful string
			targetFreq, _ := strconv.Atoi(task.Freq)
			for _, f := range freqs {
				if f.ID == targetFreq {
					task.Freq = f.Name
					break
				}
			}
			switch targetFreq {
			case 1:
				task.Days = nil
				task.Count = nil
				task.StartDate = nil
				task.OneOffDate = nil
			case 2:
				task.Days = nil
				task.Count = nil
				task.Week = nil
				task.OneOffDate = nil
			case 3:
				task.Week = nil
				task.Count = nil
				task.StartDate = nil
				task.OneOffDate = nil
			case 4:
				task.Days = nil
				task.Count = nil
				task.StartDate = nil
				task.Week = nil
			case 5:
				task.Days = nil
				task.Week = nil
				task.StartDate = nil
				task.OneOffDate = nil
			}

			go func() {
				newID := 0
				rpcClient.Call("TaskRPC.InsertSched", shared.SchedTaskRPCData{
					Channel:   Session.Channel,
					SchedTask: &task,
				}, &newID)
				// print("added task ID", newID)
				Session.Navigate(fmt.Sprintf("/sched/%d", newID))
			}()
		})

		// All done, so render the form
		form.Render("edit-form", "main", &task)
		swapper.SelectByName("week")

		// Setup a callback on the freq selector
		w := dom.GetWindow()
		doc := w.Document()

		doc.QuerySelector("[name=radio-Freq]").AddEventListener("click", false, func(evt dom.Event) {
			clickedOn := evt.Target()
			switch clickedOn.TagName() {
			case "INPUT":
				ie := clickedOn.(*dom.HTMLInputElement)
				key, _ := strconv.Atoi(ie.Value)
				switch key {
				case 1:
					swapper.SelectByName("week")
				case 2:
					swapper.SelectByName("year")
				case 3:
					swapper.SelectByName("days")
				case 4:
					swapper.SelectByName("oneoff")
				case 5:
					swapper.SelectByName("job")
				}
			}
		})

	}()

}

func siteTaskList(context *router.Context) {
	id, err := strconv.Atoi(context.Params["id"])
	if err != nil {
		print(err.Error())
		return
	}

	go func() {
		site := shared.Site{}
		tasks := []shared.Task{}
		rpcClient.Call("SiteRPC.Get", shared.SiteRPCData{
			Channel: Session.Channel,
			ID:      id,
		}, &site)
		rpcClient.Call("TaskRPC.SiteList", shared.TaskRPCData{
			Channel: Session.Channel,
			ID:      id,
		}, &tasks)

		BackURL := fmt.Sprintf("/site/%d", id)
		form := formulate.ListForm{}
		form.New("fa-server", "Active Tasks for "+site.Name)

		// Define the layout

		switch Session.UserRole {
		case "Admin", "Site Manager":
			form.Column("User", "Username")
		}

		form.Column("Date", "GetStartDate")
		// form.Column("Due", "GetDueDate")
		// form.Column("Site", "SiteName")
		form.Column("Machine", "MachineName")
		form.Column("Component", "Component")
		form.Column("Description", "Descr")
		form.Column("Duration", "DurationDays")
		form.Column("Completed", "CompletedDate")

		// Add event handlers
		form.CancelEvent(func(evt dom.Event) {
			evt.PreventDefault()
			Session.Navigate(BackURL)
		})

		form.RowEvent(func(key string) {
			Session.Navigate("/task/" + key)
		})

		form.PrintEvent(func(evt dom.Event) {
			dom.GetWindow().Print()
		})

		form.Render("site-task-list", "main", tasks)

	}()
}

func schedTaskList(context *router.Context) {
	print("TODO - schedTaskList")
}

func taskPartList(context *router.Context) {
	print("TODO - taksPartList")
}

// Show a list of all tasks
func stoppageTaskList(context *router.Context) {
	id, err := strconv.Atoi(context.Params["id"])
	if err != nil {
		print(err.Error())
		return
	}

	go func() {
		tasks := []shared.Task{}
		event := shared.Event{}
		rpcClient.Call("EventRPC.Get", shared.EventRPCData{
			Channel: Session.Channel,
			ID:      id,
		}, &event)
		rpcClient.Call("TaskRPC.StoppageList", shared.TaskRPCData{
			Channel: Session.Channel,
			ID:      id,
		}, &tasks)

		BackURL := fmt.Sprintf("/stoppage/%d", id)

		form := formulate.ListForm{}
		form.New("fa-server", fmt.Sprintf("Task List for Stoppage - %06d", id))

		form.Column("User", "Username")
		form.Column("TaskID", "ID")
		form.Column("Date", "GetStartDate")
		// form.Column("Due", "GetDueDate")
		form.Column("Site", "SiteName")
		form.Column("Machine", "MachineName")
		form.Column("Component", "Component")
		form.Column("Description", "Descr")
		form.Column("Duration", "DurationDays")
		form.Column("Completed", "CompletedDate")

		// Add event handlers
		form.CancelEvent(func(evt dom.Event) {
			evt.PreventDefault()
			Session.Navigate(BackURL)
		})

		form.RowEvent(func(key string) {
			Session.Navigate("/task/" + key)
		})

		form.PrintEvent(func(evt dom.Event) {
			dom.GetWindow().Print()
		})

		form.Render("event-task-list", "main", tasks)

	}()
}

func technicianDiary(context *router.Context) {
	go func() {
		done := false
		rpcClient.Call("TaskRPC.Diary", Session.Channel, &done)
	}()
}

func addTaskPartsTree(tree []shared.Category, ul *dom.HTMLUListElement, depth int) {

	w := dom.GetWindow()
	doc := w.Document()
	// print("adding from ", tree, " to ", ul)
	// Add a LI for each category
	for _, tv := range tree {
		// print("Tree Value", i, tv)
		widgetID := fmt.Sprintf("category-%d", tv.ID)
		li := doc.CreateElement("li")
		li.SetID(widgetID)
		chek := doc.CreateElement("input").(*dom.HTMLInputElement)
		chek.Type = "checkbox"
		li.AppendChild(chek)
		label := doc.CreateElement("label")
		label.SetAttribute("for", widgetID)
		label.SetInnerHTML(tv.Name)
		label.SetAttribute("data-type", "category")
		label.SetAttribute("data-id", fmt.Sprintf("%d", tv.ID))
		label.SetID(widgetID + "-label")
		chek.SetAttribute("data-type", "category")
		chek.SetAttribute("data-id", fmt.Sprintf("%d", tv.ID))
		chek.SetID(widgetID + "-chek")
		li.AppendChild(label)
		ul.AppendChild(li)

		if len(tv.Subcats) > 0 {
			ul2 := doc.CreateElement("ul").(*dom.HTMLUListElement)
			li.AppendChild(ul2)
			addTaskPartsTree(tv.Subcats, ul2, depth+1)
		} else {
			if depth == 0 {
				ulempty := doc.CreateElement("ul")
				li.AppendChild(ulempty)
				liempty := doc.CreateElement("li")
				liempty.SetInnerHTML("(no sub-categories)")
				ulempty.AppendChild(liempty)
			}
		}

		ul3 := doc.CreateElement("ul")
		li.AppendChild(ul3)
		if len(tv.Parts) > 0 {
			for _, part := range tv.Parts {

				partID := fmt.Sprintf("part-%d", part.ID)

				li2 := doc.CreateElement("li")
				li2.Class().Add("part-editor")

				s1 := doc.CreateElement("span")
				s1.Class().Add("partlabel")
				partLabel := doc.CreateElement("label").(*dom.HTMLLabelElement)
				partLabel.SetInnerHTML(part.StockCode + " : " + part.Name)
				partLabel.Class().Add("partlabel")
				s1.AppendChild(partLabel)

				s2 := doc.CreateElement("span")
				s2.Class().Add("part-qty-input")
				partInput := doc.CreateElement("input").(*dom.HTMLInputElement)
				partInput.Type = "number"
				partInput.Name = "part-qty-" + partID
				partInput.Value = ""
				partInput.Class().Add("part-qty-input")
				s2.AppendChild(partInput)

				li2.AppendChild(s2)
				li2.AppendChild(s1)

				ul3.AppendChild(li2)

				if false {

					////
					partID := fmt.Sprintf("part-%d", part.ID)
					li2 := doc.CreateElement("li")
					li2.Class().Add("file")
					li2.SetID(partID)

					span := doc.CreateElement("span")
					span.Class().Add("partspan")
					span.SetInnerHTML(fmt.Sprintf(`%s : %s`, part.StockCode, part.Name))
					// li2.AppendChild(span)

					/*
						<div data-field-span=1>
							<label>Stock code and part name</label>
							<input type=number name=part-qty-id value=thevalue step=1>
						</div>

					*/
					// div := doc.CreateElement("div")
					// div.SetAttribute("data-field-span", "1")
					// li2.AppendChild(div)

					// label := doc.CreateElement("label").(*dom.HTMLLabelElement)
					// label.SetInnerHTML(part.StockCode + " : " + part.Name)
					// li2.AppendChild(label)

					span2 := doc.CreateElement("span")
					span2.Class().Add("qtyspan")
					input := doc.CreateElement("input").(*dom.HTMLInputElement)
					input.Type = "number"
					input.Name = "part-qty-" + partID
					input.Value = "123"
					span2.AppendChild(input)
					li2.AppendChild(span2)
					// li2.AppendChild(input)

					li2.AppendChild(span)

					// li2.Class().Add("stock-item")
					li2.SetAttribute("data-type", "part")
					li2.SetAttribute("data-id", fmt.Sprintf("%d", part.ID))
					ul3.AppendChild(li2)
				}
			}
		} else {
			if depth > 0 {
				li3 := doc.CreateElement("li")
				li3.SetInnerHTML("(no parts)")
				ul3.AppendChild(li3)
			}
		}
	}
}
