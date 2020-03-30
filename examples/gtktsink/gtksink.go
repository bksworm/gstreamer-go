package main

import (
	"gstreamer/gstreamer-go"
	"log"
	"os"

	"github.com/gotk3/gotk3/glib"
	"github.com/gotk3/gotk3/gtk"
)

type Player struct {
	pipe   *gstreamer.Pipeline
	window *gtk.Window
}

func NewPlayer() *Player {
	var err error
	p := new(Player)

	p.window, err = gtk.WindowNew(gtk.WINDOW_TOPLEVEL)
	if err != nil {
		return p
	}
	p.window.SetTitle("Player")
	p.window.Connect("destroy", gtk.MainQuit, nil)

	vbox, err := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 0)
	vbox.SetSizeRequest(640, 480)
	p.window.Add(vbox)

	p.pipe, err = gstreamer.New(" tcpserversrc host=0.0.0.0 port=7001 ! multipartdemux ! jpegdec ! " +
		" videoconvert ! video/x-raw,format=BGRA ! gtksink name=sink ") //
	if err != nil {
		log.Fatalln("pipeline create error", err)
	}

	sink := p.pipe.FindElement("sink")
	if sink == nil {
		log.Println("Cann't get sink!")
		return nil
	}

	wdg := getWidget(sink)

	if wdg == nil {
		log.Println("Cann't get move area widget!")
		return nil
	}

	vbox.PackStart(wdg, true, true, 0)

	return p
}

func (p *Player) Run() {
	p.window.ShowAll()
	p.pipe.Start()
	gtk.Main()
}

func main() {
	gstreamer.Init()
	gtk.Init(nil)
	p := NewPlayer()
	if p != nil {
		p.Run()
	}

}

//the most time is  spent here due to  go type system and memory model
func getWidget(e *gstreamer.Element) (w *gtk.Widget) {
	var ok bool
	obj := glib.Take(e.AsObj())
	p, err := obj.GetProperty("widget")
	if err != nil {
		log.Println("No property widget!")
		return w
	}
	ip := p.(interface{})
	w, ok = ip.(*gtk.Widget)
	if ok {
		return w
	}
	return w
}
