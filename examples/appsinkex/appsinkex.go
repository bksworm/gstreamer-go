// appsinkex.go
package main

import (
	"bytes"
	"gstreamer/gstreamer-go"
	"io"
	"mime/multipart"
	"net/textproto"
	"strconv"
	"time"

	"github.com/rs/zerolog/log"
)

const (
	SIZE      = 320 * 240 * 3
	frameRate = 25
	movieLen  = 3
)

func main() {
	gstreamer.Init()

	pipeline, err := gstreamer.New(" tcpserversrc host=0.0.0.0 port=7001 ! multipartdemux  ! tee name=t  !" +
		" queue ! jpegdec ! autovideosink " +
		" t. ! queue ! appsink name=jpgrcv ")

	if err != nil {
		log.Fatal().Msg("pipeline create error " + err.Error())
	}
	defer pipeline.Stop()

	appsink := pipeline.FindElement("jpgrcv")
	if appsink == nil {
		log.Fatal().Msg("appsink create error")
	}

	ml := gstreamer.NewMainLoop()
	defer ml.Close()

	frames := appsink.Poll()

	go func() {
		var i int
		for fr := range frames {
			i += 1
			log.Printf("fr # %03d %d Kb ", i, fr.Len()/1024)
			fr = nil
		}
	}()
	pipeline.Start()
	defer pipeline.Stop()

	ml.Run()
}

const boundary = "SimpleRandomString"

func send(frames <-chan *bytes.Buffer, w io.Writer) {
	multipartWriter := multipart.NewWriter(w)
	multipartWriter.SetBoundary(boundary)

	for fr := range frames {
		tst := int(makeTimestampMilli())
		iw, err := multipartWriter.CreatePart(textproto.MIMEHeader{
			"Content-type":   []string{"image/jpeg"},
			"Content-length": []string{strconv.Itoa(fr.Len())},
			"X-Timestamp":    []string{strconv.Itoa(tst)},
		})
		if err != nil {
			log.Error().Msg(err.Error())
			return
		}
		_, err = iw.Write(fr.Bytes())
		if err != nil {
			log.Error().Msg(err.Error())
			return
		}
	}
}

func unixMilli(t time.Time) int64 {
	return t.Round(time.Millisecond).UnixNano() / (int64(time.Millisecond) / int64(time.Nanosecond))
}

func makeTimestampMilli() int64 {
	return unixMilli(time.Now())
}
