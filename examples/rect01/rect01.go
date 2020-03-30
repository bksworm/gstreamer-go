package main

import (
	"gstreamer/gstreamer-go"
	"log"
	"math/rand"
)

const (
	SIZE      = 320 * 240 * 3
	frameRate = 25
	movieLen  = 3
)

func main() {
	gstreamer.Init()
	//pipeline, err := gstreamer.New("appsrc name=mysource   emit-signals=true  format=time is-live=true do-timestamp=true ! videoconvert ! autovideosink")
	//pipeline, err := gstreamer.New("appsrc name=mysource format=time stream-type=0 is-live=true  do-timestamp=true !" +
	//	" videoconvert ! jpegenc ! multipartmux  ! filesink location=multipart.mjpeg ")
	pipeline, err := gstreamer.New("appsrc name=mysource stream-type=0 format=time is-live=true do-timestamp=true !" +
		" videoconvert ! " +
		" vp9enc ! matroskamux  !" +
		" filesink location=vp9.mkv ")
	if err != nil {
		log.Fatalln("pipeline create error", err)
	}
	defer pipeline.Stop()

	appsrc := pipeline.FindElement("mysource")
	mb := pipeline.PullMessage()
	go func() {
		for {
			select {
			case <-mb:
				break
			}
		}
	}()

	appsrc.SetCap("video/x-raw,format=RGB,width=320,height=240,framerate=25/1")

	pch := appsrc.PushBuffer()
	ml := gstreamer.NewMainLoop()
	defer ml.Close()

	go func() {
		b := make([]byte, SIZE)
		for n := 0; n < frameRate*movieLen; n++ {
			for i := 0; i < SIZE; i++ {
				b[i] = byte(rand.Intn(255))
			}
			pch <- b
		}
		ml.Quit()
	}()

	appsrc.StartPush()

	pipeline.Start()
	defer pipeline.Stop()

	ml.Run()
}
