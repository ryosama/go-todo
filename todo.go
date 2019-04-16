package main

import (
	"encoding/json"
	"fmt"
	"github.com/gotk3/gotk3/gdk"
	"github.com/gotk3/gotk3/gtk"
	//"github.com/davecgh/go-spew/spew"
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
.notdone {
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
	loadEvents()

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
	btn_delete.SetMarginEnd(5)
	//registerWidget(btn_delete)

	// create a check button
	checkbutton, _ := gtk.CheckButtonNew()
	checkbutton.SetActive(todoList[id].Done)
	checkbutton.SetTooltipText("Check as done")
	checkbutton.SetName("Check_" + id_string)
	checkbutton.SetMarginEnd(5)
	//registerWidget(checkbutton)

	// create an entry
	entry, _ := gtk.EntryNew()
	entry.SetText(todoList[id].Text)
	entry.SetName("Entry_" + id_string)
	registerWidget(entry) // to use getWidgetByName(string) later

	// add style to entry
	styleContext, _ := entry.GetStyleContext()
	if todoList[id].Done {
		styleContext.RemoveClass("notdone")
		styleContext.AddClass("done")
	} else {
		styleContext.RemoveClass("done")
		styleContext.AddClass("notdone")
	}

	boxContainer.Add(btn_delete)
	boxContainer.Add(checkbutton)
	boxContainer.PackEnd(entry, true, true, 0)

	item.Add(boxContainer)

	listbox.Insert(item, -1)
	listbox.ShowAll()

	entry.Connect("key-press-event", func(elm *gtk.Entry) {
		elm_name, _ := elm.GetName()
		id, _ := strconv.Atoi(strings.Split(elm_name, "_")[1])
		s, _ := elm.GetText()
		todoList[id].Text = s
	})

	checkbutton.Connect("clicked", func(elm *gtk.CheckButton) {
		elm_name, _ := elm.GetName()
		id, _ := strconv.Atoi(strings.Split(elm_name, "_")[1])
		todoList[id].Done = !todoList[id].Done

		// invert style
		widget, _ := getWidgetByName("Entry_" + strconv.Itoa(id))
		styleContext, _ := widget.GetStyleContext()
		if todoList[id].Done {
			styleContext.RemoveClass("notdone")
			styleContext.AddClass("done")
		} else {
			styleContext.RemoveClass("done")
			styleContext.AddClass("notdone")
		}
	})

	btn_delete.Connect("clicked", func(elm *gtk.Button) {
		elm_name, _ := elm.GetName()
		id, _ := strconv.Atoi(strings.Split(elm_name, "_")[1])
		fmt.Printf("DEBUG : Remove element %d\n", id)
		todoList = append(todoList[:id], todoList[id+1:]...)
		clearListGui()
		buildListGui()
	})
}

func loadEvents() {
	// widget events

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
		fmt.Printf("DEBUG : read config file\n")
	}

	// for debug
	/*	todoList = nil
		todoList = append(todoList, &Todo{Text: "Ceci est un texte d'exemple", Done: true})
		todoList = append(todoList, &Todo{Text: "Ceci est un 2eme exemple", Done: false})
		todoList = append(todoList, &Todo{Text: "Ceci est un 3eme exemple", Done: true})
	*/
	fmt.Printf("DEBUG : read todo list\n")
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
