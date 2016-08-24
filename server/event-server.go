package main

import (

	// "encoding/base64"
	"fmt"
	"log"
	"strings"
	"time"

	"itrak-cmms/shared"
)

type EventRPC struct{}

// Raise a new event
func (t *EventRPC) Raise(issue shared.RaiseIssue, id *int) error {
	start := time.Now()

	conn := Connections.Get(issue.Channel)
	*id = 0

	ToolName := issue.NonTool
	if issue.IsTool {
		ToolName = issue.Component.Name
	}

	// Create 1 event record - which includes details of both tool and machine
	evt := &shared.Event{
		SiteID:    issue.Machine.SiteID,
		Type:      "Alert",
		MachineID: issue.Machine.ID,
		ToolID:    issue.CompID,
		ToolType:  ToolName,
		CreatedBy: conn.UserID,
		Notes:     issue.Descr,
		Priority:  1,
		Status:    "Pending",
		Photo:     issue.Photo,
	}

	// Create the event record and get its ID
	DB.InsertInto("event").
		Whitelist("site_id", "type", "machine_id", "tool_id", "tool_type", "created_by",
			"notes", "priority", "status").
		Record(evt).
		Returning("id").
		QueryScalar(id)

	// Process the photo if present
	if issue.Photo.Data != "" {
		issue.Photo.Entity = "event"
		issue.Photo.EntityID = *id
		decodePhoto(&issue.Photo)
		DB.InsertInto("photo").
			Columns("entity", "entity_id", "photo", "thumb", "preview").
			Record(issue.Photo).
			Exec()
	}
	conn.Broadcast("event", "insert", *id)

	DB.SQL(`update machine 
			set alert_at=localtimestamp, status=$2 
			where id=$1`,
		issue.Machine.ID,
		`Needs Attention`).
		Exec()

	conn.Broadcast("machine", "update", issue.Machine.ID)
	conn.Broadcast("sitestatus", "update", 1)

	// if its a tool, then update the tool record, otherwise update the non-tool field on the machine record
	if evt.ToolID == 0 {
		// is a non-tool.
		fieldName := ""
		switch evt.ToolType {
		case "Electrical":
			fieldName = "electrical"
		case "Hydraulic":
			fieldName = "hydraulic"
		case "Lube":
			fieldName = "lube"
		case "Printer":
			fieldName = "printer"
		case "Console":
			fieldName = "console"
		case "Uncoiler":
			fieldName = "uncoiler"
		case "Rollbed":
			fieldName = "rollbed"
		case "Pnuematic":
			fieldName = "pnuematic"
		case "Encoder":
			fieldName = "encoder"
		case "Conveyor":
			fieldName = "conveyor"
		case "StripGuide":
			fieldName = "strip_guide"
		}
		if fieldName != "" {
			DB.SQL(fmt.Sprintf("update machine set %s='Needs Attention' where id=$1", fieldName), evt.MachineID).Exec()
		}
	} else {
		// is a tool
		DB.SQL(`update component
			set status='Needs Attention'
			where id=$1`, evt.ToolID).
			Exec()
	}

	// Generate an SMS
	// err = SendSMS("0417824950",
	// 	fmt.Sprintf("%s on Machine %s %s: %s", "Alert", issue.Machine.Name, ToolName, issue.Descr),
	// 	fmt.Sprintf("%d", id))

	// Generate an Email with same details as SMS

	// Patch in any attached documents
	// _, err = DB.SQL(`update doc
	// 	set ref_id=$1, name=$3, type='toolevent'
	// 	where type='temptoolevent' and ref_id=$2
	// 	`, evt.ID, evt.ToolId, evt.Notes).Exec()

	logger(start, "Event.Raise",
		fmt.Sprintf("Channel %d, Machine %d, User %d %s %s",
			issue.Channel, issue.Machine.ID, conn.UserID, conn.Username, conn.UserRole),
		fmt.Sprintf("Event %d Tool %d:%s Desc %s", *id, evt.ToolID, ToolName, evt.Notes),
		issue.Channel, conn.UserID, "event", *id, true)

	siteName := ""
	DB.SQL(`select name from site where id=$1`, issue.Machine.SiteID).QueryScalar(&siteName)

	if Config.SMSOn {
		//		SendSMS("get the number from the db for the correct person",
		SendSMS("0417824950", // shane
			fmt.Sprintf("Alert at Site %s on Machine %s on %s: %s",
				siteName,
				issue.Machine.Name,
				ToolName,
				issue.Descr),
			fmt.Sprintf("%d", evt.ID), 8)
	} else {
		willSend := fmt.Sprintf("Alert at Site %s on Machine %s on %s: %s",
			siteName,
			issue.Machine.Name,
			ToolName,
			issue.Descr)

		log.Println("will send SMS", willSend)

	}

	return nil
}

func (e *EventRPC) List(channel int, events *[]shared.Event) error {
	start := time.Now()

	conn := Connections.Get(channel)

	switch conn.UserRole {
	case "Site Manager":
		// Limit the tasks to just the sites that we are in control of
		sites := []int{}

		DB.SQL(`select site_id from user_site where user_id=$1`, conn.UserID).QuerySlice(&sites)

		err := DB.SQL(`select 
		e.*,m.name as machine_name,s.name as site_name,u.username as username,x.highlight as site_highlight
		from event e
			left join machine m on m.id=e.machine_id
			left join site s on s.id=m.site_id
			left join users u on u.id=e.created_by
			left join user_site x on x.user_id=$2 and x.site_id=e.site_id
		where m.site_id in $1
			and e.completed is null
		order by e.startdate desc`, sites, conn.UserID).
			QueryStructs(events)

		if err != nil {
			log.Println(err.Error())
		}
	case "Admin":
		err := DB.SQL(`select 
		e.*,m.name as machine_name,s.name as site_name,u.username as username,x.highlight as site_highlight
		from event e
			left join machine m on m.id=e.machine_id
			left join site s on s.id=m.site_id
			left join users u on u.id=e.created_by	
			left join user_site x on x.user_id=$1 and x.site_id=e.site_id
		where e.completed is null	
		order by e.completed desc,e.startdate desc`, conn.UserID).
			QueryStructs(events)

		if err != nil {
			log.Println(err.Error())
		}
	}

	// fetch all assignments
	for i, v := range *events {
		DB.SQL(`select u.username
			from task t
			left join users u on u.id=t.assigned_to
			where t.event_id=$1`, v.ID).
			QueryStructs(&v.AssignedTo)

		// log.Println("assignments for event", v.ID, "=", v.AssignedTo)
		(*events)[i].AssignedTo = v.AssignedTo

		// truncate long notes
		if len(v.Notes) > 80 {
			(*events)[i].Notes = fmt.Sprintf("%s ...", v.Notes[:80])
		}

		// Get any thumbnails if present
		photo := shared.Photo{}

		DB.SQL(`select thumb from photo where entity='event' and entity_id=$1`, v.ID).
			QueryStruct(&photo)
		(*events)[i].Photo.Thumb = photo.Thumb

	}

	logger(start, "Event.List",
		fmt.Sprintf("Channel %d, User %d %s %s",
			channel, conn.UserID, conn.Username, conn.UserRole),
		fmt.Sprintf("%d Events", len(*events)),
		channel, conn.UserID, "event", 0, false)

	return nil
}

func (e *EventRPC) ListCompleted(channel int, events *[]shared.Event) error {
	start := time.Now()

	conn := Connections.Get(channel)

	switch conn.UserRole {
	case "Site Manager":
		// Limit the tasks to just the sites that we are in control of
		sites := []int{}

		DB.SQL(`select site_id from user_site where user_id=$1`, conn.UserID).QuerySlice(&sites)

		err := DB.SQL(`select 
		e.*,m.name as machine_name,s.name as site_name,u.username as username
		from event e
			left join machine m on m.id=e.machine_id
			left join site s on s.id=m.site_id
			left join users u on u.id=e.created_by
		where m.site_id in $1
			and e.completed is not null
			and e.startdate > NOW() - INTERVAL '1 month'
		order by e.startdate desc`, sites).
			QueryStructs(events)

		if err != nil {
			log.Println(err.Error())
		}
	case "Admin":
		err := DB.SQL(`select 
		e.*,m.name as machine_name,s.name as site_name,u.username as username
		from event e
			left join machine m on m.id=e.machine_id
			left join site s on s.id=m.site_id
			left join users u on u.id=e.created_by	
		where e.completed is not null	
			and e.startdate > NOW() - INTERVAL '1 month'
		order by e.completed desc,e.startdate desc`).
			QueryStructs(events)

		if err != nil {
			log.Println(err.Error())
		}
	}

	// fetch all assignments
	for i, v := range *events {
		DB.SQL(`select u.username
			from task t
			left join users u on u.id=t.assigned_to
			where t.event_id=$1`, v.ID).
			QueryStructs(&v.AssignedTo)

		// log.Println("assignments for event", v.ID, "=", v.AssignedTo)
		(*events)[i].AssignedTo = v.AssignedTo

		// truncate long notes
		if len(v.Notes) > 80 {
			(*events)[i].Notes = fmt.Sprintf("%s ...", v.Notes[:80])
		}

		// Get any thumbnails if present
		photo := shared.Photo{}

		DB.SQL(`select id,thumb 
			from photo 
			where entity='event' 
			and entity_id=$1`, v.ID).
			QueryStruct(&photo)
		(*events)[i].Photo.Thumb = photo.Thumb
	}

	logger(start, "Event.ListCompleted",
		fmt.Sprintf("Channel %d, User %d %s %s",
			channel, conn.UserID, conn.Username, conn.UserRole),
		fmt.Sprintf("%d Events", len(*events)),
		channel, conn.UserID, "event", 0, false)

	return nil
}

func (e *EventRPC) Get(data shared.EventRPCData, event *shared.Event) error {
	start := time.Now()

	conn := Connections.Get(data.Channel)
	id := data.ID

	// Read the sites that this user has access to
	err := DB.SQL(`select
		e.*,m.name as machine_name,s.name as site_name,u.username as username
		from event e
			left join machine m on m.id=e.machine_id
			left join site s on s.id=m.site_id
			left join users u on u.id=e.created_by
		where e.id=$1`, id).QueryStruct(event)

	// fetch all assignments
	DB.SQL(`select u.username
			from task t
			left join users u on u.id=t.assigned_to
			where t.event_id=$1`, id).
		QueryStructs(&event.AssignedTo)

	if err != nil {
		log.Println(err.Error())
	}

	// Get the photo preview if present
	DB.SQL(`select id,preview,filename,type,datatype,entity,entity_id,notes
		from photo 
		where entity='event' and entity_id=$1`, id).
		QueryScalar(&event.Photo)

	logger(start, "Event.Get",
		fmt.Sprintf("ID %d", id),
		event.Notes,
		data.Channel, conn.UserID, "event", id, false)

	return nil
}

func (e *EventRPC) Update(data shared.EventRPCData, done *bool) error {
	start := time.Now()

	conn := Connections.Get(data.Channel)

	DB.Update("event").
		SetWhitelist(data.Event, "notes").
		Where("id = $1", data.Event.ID).
		Exec()

	logger(start, "Event.Update",
		fmt.Sprintf("Channel %d, Event %d User %d %s %s",
			data.Channel, data.Event.ID, conn.UserID, conn.Username, conn.UserRole),
		fmt.Sprintf("%s", data.Event.Notes),
		data.Channel, conn.UserID, "event", data.Event.ID, true)

	conn.Broadcast("event", "update", data.Event.ID)
	return nil
}

func (e *EventRPC) Complete(data shared.EventRPCData, done *bool) error {
	start := time.Now()

	conn := Connections.Get(data.Channel)
	if conn.UserRole != "Admin" {
		return nil
	}

	// Read the sites that this user has access to
	event := shared.Event{}
	DB.SQL(`select
		e.*,m.name as machine_name,s.name as site_name,u.username as username
		from event e
			left join machine m on m.id=e.machine_id
			left join site s on s.id=m.site_id
			left join users u on u.id=e.created_by
		where e.id=$1`, data.Event.ID).QueryStruct(&event)

	// Mark the event as complete
	DB.SQL(`update event 
		set completed=now(), status='Complete'
		where id=$1`, data.Event.ID).Exec()

	// Reset the affected component - this code is the reverse of
	// the code in the RaiseEvent function above
	if event.ToolID == 0 {
		// Reset the status of the basic component on this machine
		fieldName := ""
		switch event.ToolType {
		case "Electrical":
			fieldName = "electrical"
		case "Hydraulic":
			fieldName = "hydraulic"
		case "Lube":
			fieldName = "lube"
		case "Printer":
			fieldName = "printer"
		case "Console":
			fieldName = "console"
		case "Uncoiler":
			fieldName = "uncoiler"
		case "Rollbed":
			fieldName = "rollbed"
		case "Pnuematic":
			fieldName = "pnuematic"
		case "Conveyor":
			fieldName = "conveyor"
		case "Encoder":
			fieldName = "encoder"
		case "StripGuide":
			fieldName = "strip_guide"
		}
		if fieldName != "" {
			DB.SQL(fmt.Sprintf("update machine set %s='Running' where id=$1", fieldName), event.MachineID).Exec()
		}
	} else {
		// is a tool
		DB.SQL(`update component
			set status='Running'
			where id=$1`, event.ToolID).
			Exec()
	}

	// Reset the whole machine if clear
	machineIsClear := true
	machine := shared.Machine{}
	DB.SQL(`select * from machine where id=$1`, event.MachineID).QueryStruct(&machine)

	if machine.Electrical != "Running" ||
		machine.Hydraulic != "Running" ||
		machine.Printer != "Running" ||
		machine.Console != "Running" ||
		machine.Rollbed != "Running" ||
		machine.Uncoiler != "Running" ||
		machine.Pnuematic != "Running" ||
		machine.Encoder != "Running" ||
		machine.StripGuide != "Running" ||
		machine.Conveyor != "Running" ||
		machine.Lube != "Running" {
		machineIsClear = false
	}

	if machineIsClear {
		badComps := 0
		DB.SQL(`select count(*) 
			from component 
			where status != 'Running' and machine_id=$1`, machine.ID).
			QueryScalar(&badComps)

		if badComps == 0 {
			DB.SQL("update machine set status='Running' where id=$1", event.MachineID).Exec()
		}
	}

	logger(start, "Event.Complete",
		fmt.Sprintf("Channel %d, Event %d User %d %s %s",
			data.Channel, data.Event.ID, conn.UserID, conn.Username, conn.UserRole),
		"Manually Completed by Admin",
		data.Channel, conn.UserID, "event", data.Event.ID, true)

	conn.Broadcast("event", "update", data.Event.ID)
	conn.Broadcast("sitestatus", "update", 1)

	return nil
}

// Add a site
func (e *EventRPC) Workorder(data shared.AssignEvent, id *int) error {
	start := time.Now()

	conn := Connections.Get(data.Channel)

	*id = 0
	// log.Printf("here with %v", data)

	now := time.Now()
	if data.StartDate == nil {
		data.StartDate = &now
	}
	if data.StartDate.Before(now) {
		data.StartDate = &now
	}
	if data.DueDate == nil {
		data.DueDate = &now
	}
	if data.DueDate.Before(now) {
		data.DueDate = &now
	}
	task := shared.Task{
		MachineID:    data.Event.MachineID,
		SchedID:      0,
		EventID:      data.Event.ID,
		CompType:     data.Event.ToolType,
		Component:    data.Event.ToolType,
		StartDate:    data.StartDate,
		DueDate:      data.DueDate,
		Descr:        data.Notes,
		AssignedBy:   &conn.UserID,
		AssignedTo:   &data.AssignTo,
		AssignedDate: &now,
		LabourEst:    data.LabourEst,
		MaterialEst:  data.MaterialEst,
	}
	// log.Printf("task is %v", task)

	DB.InsertInto("task").
		Whitelist("machine_id", "sched_id", "event_id", "comp_type", "tool_id", "component",
			"descr", "startdate", "due_date", "escalate_date",
			"assigned_by", "assigned_to", "assigned_date",
			"labour_est", "material_est").
		Record(&task).
		Returning("id").
		QueryScalar(&task.ID)

	*id = task.ID

	// if there is a new photo attached, then process it
	if data.Photo.Data != "" {
		data.Photo.Entity = "task"
		data.Photo.EntityID = task.ID
		// decodePhoto(photo.Data, &photo.Preview, &photo.Thumb, &photo.Type, &photo.Datatype)
		decodePhoto(&data.Photo)
		DB.InsertInto("photo").
			Columns("entity", "entity_id", "photo", "thumb", "preview").
			Record(data.Photo).
			Exec()
	}

	// Stamp the event as assigned
	DB.SQL(`update event set status='Assigned' where id=$1`, data.Event.ID).Exec()

	if false {
		print("TODO - all this code here is redundant - apply bits that are needed, and kill the rest")
		// Expand out using the hashtags
		hasHashtag := false
		oldDescr := data.Notes

		if strings.Contains(oldDescr, "#") {
			hashes := []shared.Hashtag{}
			// Apply the longest hashtag first

			DB.SQL(`select * from hashtag order by length(name) desc`).QueryStructs(&hashes)

			// Keep looping through doing text conversions until there is
			// nothing left to expand
			stillLooking := true
			for stillLooking {
				stillLooking = false
				for _, v := range hashes {
					theHash := "#" + v.Name
					if strings.Contains(oldDescr, theHash) {
						oldDescr = strings.Replace(oldDescr, theHash, v.Descr, -1)
						hasHashtag = true
						stillLooking = true
					}
				}
			}
		}

		// Now generate the task check items based on the description field of the schedtask
		lines := strings.Split(oldDescr, "\n")
		seq := 1
		descr := ""

		for _, l := range lines {
			theLine := strings.TrimSpace(l)
			if strings.HasPrefix(theLine, "- ") {
				check := shared.TaskCheck{
					TaskID: task.ID,
					Seq:    seq,
					Descr:  theLine[2:],
					Done:   false,
				}

				DB.InsertInto("task_check").
					Whitelist("task_id", "seq", "descr", "done").
					Record(check).
					Exec()
				seq++
			} else {
				descr += l
				descr += "\n"
			}
		}
		// log.Println("Modded desc from", task.Descr, "to", descr)
		if hasHashtag || seq > 1 {
			DB.SQL(`update task set descr=$1 where id=$2`, descr, task.ID).Exec()
		}
	}

	// Now generate an SMS to the technician
	// smsMsg := fmt.Sprintf("New Workorder at %s for Machine %s : %s",
	// 	data.SiteName,
	// 	data.MachineName,
	// 	data.ToolType)
	notes := data.Notes
	if len(notes) > 40 {
		notes = notes[:40] + "..."
	}
	smsMsg := fmt.Sprintf("Task %06d:\n %s - %s : %s",
		task.ID,
		notes,
		data.MachineName,
		data.ToolType)

	phoneNumber := ""
	DB.SQL(`select sms from users where id=$1`, data.AssignTo).QueryScalar(&phoneNumber)

	if Config.SMSOn {

		if phoneNumber != "" {
			SendSMS(phoneNumber, smsMsg, fmt.Sprintf("%d", task.ID), data.AssignTo)
		} else {
			log.Println("No Phone Number for SMS:", smsMsg)
		}
	} else {
		log.Println("Will send SMS:", smsMsg, "to", phoneNumber)
	}

	if false {
		// HET - yactn are no longer tightly coupled to the 3aAaya

		// Now add the parts to the task based on the dataset for the type of machine
		partClass := 0
		DB.SQL(`select part_class from machine where id=$1`, task.MachineID).QueryScalar(&partClass)
		if partClass != 0 {
			// log.Println("part class =", partClass)
			parts := []shared.Part{}
			DB.SQL(`select * from part where class=$1`, partClass).QueryStructs(&parts)
			for _, v := range parts {

				taskPart := shared.TaskPart{
					TaskID: task.ID,
					PartID: v.ID,
					Qty:    0,
					Notes:  "",
				}
				// log.Println("got part", taskPart)

				DB.InsertInto("task_part").
					Whitelist("task_id", "part_id", "qty", "notes").
					Record(taskPart).
					Exec()
			}
		}
	}

	logger(start, "Event.Workorder",
		fmt.Sprintf("Channel %d, Event %d, User %d %s %s",
			data.Channel, *id, conn.UserID, conn.Username, conn.UserRole),
		data.Notes,
		data.Channel, conn.UserID, "event", data.Event.ID, true)

	conn.Broadcast("task", "insert", task.ID)
	conn.Broadcast("event", "update", data.Event.ID)
	return nil
}
