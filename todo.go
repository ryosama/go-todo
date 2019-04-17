package main

import (
	"encoding/json"
	"fmt"
	//"github.com/davecgh/go-spew/spew"
	"github.com/gotk3/gotk3/gdk"
	"github.com/gotk3/gotk3/gtk"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
	"time"
)

type Widget interface {
	GetName() (string, error)
	GetStyleContext() (*gtk.StyleContext, error)
}

type Config struct {
	Width       int    `json:"width"`
	Height      int    `json:"height"`
	Left        int    `json:"left"`
	Top         int    `json:"top"`
	CurrentList string `json:"currentList"`
}

type Todo struct {
	Done bool   `json:"done"`
	Text string `json:"text"`
}

const configFile = "config.json"

const css = `
.done {
	color:grey;
	text-decoration:line-through;
	font-weight:normal;
}

.beingEdited {
	background-color:lightgrey;
}

entry {
	color:black;
	text-decoration:none;
	font-weight:bold;
}
`

var config Config
var todoList []*Todo
var win *gtk.Window
var btn_add, btn_remove, btn_remove_all *gtk.Button
var listbox *gtk.ListBox
var entry *gtk.Entry
var cssProvider *gtk.CssProvider
var builder *gtk.Builder
var widgets map[string]Widget // contains all the widgets

func main() {
	widgets = make(map[string]Widget)
	readConfig()
	readTodoList()
	registerPosition() // remerber window and size position

	// Initialize GTK without parsing any command line arguments.
	gtk.Init(nil)

	// Create a new toplevel window, set its title, and connect it to the
	win, _ = gtk.WindowNew(gtk.WINDOW_TOPLEVEL)
	win.SetTitle(config.CurrentList)

	drawInterface()

	// load entries style
	screen, _ := gdk.ScreenGetDefault()
	cssProvider, _ = gtk.CssProviderNew()
	gtk.AddProviderForScreen(screen, cssProvider, gtk.STYLE_PROVIDER_PRIORITY_APPLICATION)
	cssProvider.LoadFromData(css)

	// draw the todo List
	buildListGui()

	// Set the default window size.
	if config.Width == 0 {
		config.Width = 300
	}
	if config.Height == 0 {
		config.Height = 500
	}
	win.SetDefaultSize(config.Width, config.Height)
	win.Move(config.Left, config.Top)

	// events
	entry.Connect("activate", func() {
		s, _ := entry.GetText()
		fmt.Printf("DEBUG : Add item '%s'\n", s)
		todoList = append(todoList, &Todo{Text: s, Done: false})
		addItemToListbox(len(todoList) - 1)
		entry.SetText("")
	})

	// Window events
	win.Connect("destroy", func() {
		saveConfig()
		saveTodoList()
		gtk.MainQuit()
	})

	// Recursively show all widgets contained in this window.
	win.ShowAll()

	// Begin executing the GTK main loop.  This blocks until
	gtk.Main()
}

func addItemToListbox(id int) {
	id_string := strconv.Itoa(id)

	// create a new row
	item, _ := gtk.ListBoxRowNew()

	// this row contains 3 element horizontaly : delete btn, checkbtn, entry
	boxContainer, _ := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 0)

	// create a delete button
	deleteIcon, _ := gtk.ImageNew()
	deleteIcon.SetFromIconName("edit-delete", gtk.ICON_SIZE_LARGE_TOOLBAR)
	btn_delete, _ := gtk.ButtonNew()
	btn_delete.SetImage(deleteIcon)
	btn_delete.SetName("Delete_" + id_string)
	btn_delete.SetTooltipText("Delete this thing")
	btn_delete.SetMarginEnd(2)

	// create a delete button
	editIcon, _ := gtk.ImageNew()
	editIcon.SetFromIconName("accessories-text-editor", gtk.ICON_SIZE_LARGE_TOOLBAR)
	btn_edit, _ := gtk.ButtonNew()
	btn_edit.SetImage(editIcon)
	btn_edit.SetName("Edit_" + id_string)
	btn_edit.SetTooltipText("Edit this thing")
	btn_edit.SetMarginEnd(5)

	// create a check button
	checkbutton, _ := gtk.CheckButtonNew()
	checkbutton.SetActive(todoList[id].Done)
	checkbutton.SetTooltipText("Check as done")
	checkbutton.SetName("Check_" + id_string)
	checkbutton.SetMarginEnd(5)

	// create an entry
	entry, _ := gtk.EntryNew()
	entry.SetText(todoList[id].Text)
	entry.SetName("Entry_" + id_string)
	entry.SetEditable(false)
	registerWidget(entry) // to use getWidgetByName(string) later

	// add style to entry
	chooseClassDone(entry, todoList[id].Done)

	// add elemenbt to line
	boxContainer.Add(btn_delete)
	boxContainer.Add(btn_edit)
	boxContainer.Add(checkbutton)
	boxContainer.PackEnd(entry, true, true, 0)

	// add line to list row
	item.Add(boxContainer)

	// add list row to listbox and display it
	listbox.Insert(item, -1)
	listbox.ShowAll()

	// events
	btn_delete.Connect("clicked", func(elm *gtk.Button) {
		elm_name, _ := elm.GetName()
		id, _ := strconv.Atoi(strings.Split(elm_name, "_")[1])
		fmt.Printf("DEBUG : Remove element %d\n", id)
		todoList = append(todoList[:id], todoList[id+1:]...)
		clearListGui()
		buildListGui()
	})

	btn_edit.Connect("clicked", func(elm *gtk.Button) {
		elm_name, _ := elm.GetName()
		id, _ := strconv.Atoi(strings.Split(elm_name, "_")[1])
		fmt.Printf("DEBUG : Edit element %d\n", id)
		widget, _ := getWidgetByName("Entry_" + strconv.Itoa(id))
		widget.(*gtk.Entry).SetEditable(true)
		chooseClassBeingEdited(widget.(*gtk.Entry), true)
	})

	checkbutton.Connect("clicked", func(elm *gtk.CheckButton) {
		elm_name, _ := elm.GetName()
		id, _ := strconv.Atoi(strings.Split(elm_name, "_")[1])
		todoList[id].Done = !todoList[id].Done
		fmt.Printf("DEBUG : Invert status of %d\n", id)

		// invert style
		widget, _ := getWidgetByName("Entry_" + strconv.Itoa(id))
		chooseClassDone(widget.(*gtk.Entry), todoList[id].Done)
	})

	entry.Connect("key-press-event", func(elm *gtk.Entry, event *gdk.Event) {
		//spew.Dump(event)
		eventKey := gdk.EventKeyNewFromEvent(event)
		key := eventKey.KeyVal()

		if key == 65293 { // ENTER
			elm_name, _ := elm.GetName()
			id, _ := strconv.Atoi(strings.Split(elm_name, "_")[1])
			s, _ := elm.GetText()
			todoList[id].Text = s
			elm.SetEditable(false)
			chooseClassBeingEdited(elm, false)
		}
	})
}

func getListboxLength(lb *gtk.ListBox) int {
	i := 0
	item := listbox.GetRowAtIndex(i)
	for item != nil {
		i++
		item = listbox.GetRowAtIndex(i)
	}
	return i
}

func drawInterface() {
	// create label
	label, _ := gtk.LabelNew("ToDo List : " + config.CurrentList)

	// create listBox
	listbox, _ = gtk.ListBoxNew()
	listbox.SetSelectionMode(gtk.SELECTION_NONE) // no need of selection

	// create scrollable container
	scrollableContainer, _ := gtk.ScrolledWindowNew(nil, nil)

	// add listbox to scrollable container
	scrollableContainer.Add(listbox)

	// create entry (textfield)
	entry, _ = gtk.EntryNew()
	entry.SetTooltipText("Add new thing to do")

	// create main container boxContainer and add widget
	boxContainer, _ := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 0)
	boxContainer.Add(label)
	boxContainer.PackStart(scrollableContainer, true, true, 0)
	boxContainer.PackEnd(entry, false, true, 0)

	// add main container to window
	win.Add(boxContainer)
}

func buildListGui() {
	for i := 0; i < len(todoList); i++ {
		addItemToListbox(i)
	}
}

func clearListGui() {
	for i := getListboxLength(listbox) - 1; i >= 0; i-- {
		listbox.Remove(listbox.GetRowAtIndex(i))
	}
}

func readConfig() {
	if _, err := os.Stat(configFile); err == nil {
		file, _ := ioutil.ReadFile(configFile)
		json.Unmarshal([]byte(file), &config)
		fmt.Printf("DEBUG : read config file\n")
	}

	if len(config.CurrentList) == 0 {
		config.CurrentList = "default_list.json"
	}
}

func saveConfig() {
	jsonString, _ := json.Marshal(config)
	ioutil.WriteFile(configFile, jsonString, 0755)
	fmt.Printf("DEBUG : save config file\n")
}

func readTodoList() {
	if _, err := os.Stat(config.CurrentList); err == nil {
		file, _ := ioutil.ReadFile(config.CurrentList)
		json.Unmarshal([]byte(file), &todoList)
		fmt.Printf("DEBUG : read todo list\n")
	}

	// for debug
	/*todoList = nil
	todoList = append(todoList, &Todo{Text: "Ceci est un texte d'exemple", Done: true})
	todoList = append(todoList, &Todo{Text: "Ceci est un 2eme exemple", Done: false})
	todoList = append(todoList, &Todo{Text: "Ceci est un 3eme exemple", Done: true})
	*/
}

func saveTodoList() {
	jsonString, _ := json.Marshal(todoList)
	ioutil.WriteFile(config.CurrentList, jsonString, 0755)
	fmt.Printf("DEBUG : save todo list\n")
}

func registerPosition() {
	ticker := time.NewTicker(5000 * time.Millisecond)
	go func() {
		for range ticker.C {
			config.Left, config.Top = win.GetPosition()
			config.Width, config.Height = win.GetSize()
		}
	}()
}

func registerWidget(w Widget) {
	name, _ := w.GetName()
	widgets[name] = w
}

func getWidgetByName(n string) (val Widget, ok bool) {
	val, ok = widgets[n]
	return val, ok
}

func chooseClassDone(entry *gtk.Entry, done bool) {
	styleContext, _ := entry.GetStyleContext()
	if done {
		styleContext.AddClass("done")
	} else {
		styleContext.RemoveClass("done")
	}
}

func chooseClassBeingEdited(entry *gtk.Entry, beingEdited bool) {
	styleContext, _ := entry.GetStyleContext()
	if beingEdited {
		styleContext.AddClass("beingEdited")
	} else {
		styleContext.RemoveClass("beingEdited")
	}
}
