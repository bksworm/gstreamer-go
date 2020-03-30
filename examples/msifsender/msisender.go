package main

import (
	"bytes"
	"gstreamer/examples/msifsender/msif"
	"gstreamer/gstreamer-go"
	"log"

	"github.com/oxtoacart/bpool"
)

const maxPackets = 1024

func main() {
	gstreamer.Init()

	queue := make(chan *bytes.Buffer, maxPackets)
	packetPool := bpool.NewSizedBufferPool(maxPackets, msif.MaxPacketSize+4)

	pipeline, err := gstreamer.New(" tcpserversrc host=0.0.0.0 port=7001 ! multipartdemux ! appsink name=jpgrcv")
	if err != nil {
		log.Fatalln("pipeline create error", err)
	}

	appsink := pipeline.FindElement("jpgrcv")
	ml := gstreamer.NewMainLoop()
	defer ml.Close()

	frames := appsink.Poll()

	go func() {
		var i uint16
		for fr := range frames {
			i += 1
			msif.Bb2Que(i, fr, packetPool, queue)
			log.Printf("fr # %03d %d Kb ", i, fr.Len()/1024)
			fr = nil
		}
	}()

	go func() {
		var i int
		for packet := range queue {
			i += 1
			log.Printf("p # %08d", i)
			packetPool.Put(packet)
		}

	}()

	pipeline.Start()
	defer pipeline.Stop()

	ml.Run()

}
