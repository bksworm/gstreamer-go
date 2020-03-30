package msif

import(
	"errors"
	"fmt"
	"bytes"
	"io/ioutil"
	"io"
	"log"
	"github.com/oxtoacart/bpool"
)

/*
bits
31 - Резерв
30- 26  Адp.a6oнeнтa
25- 21  Aдp.терминала  
20 - K 
19 - E  
18-16 Резерв 
15-0  Счетчик сообщений 

31-16 Колличество фрагментов в сообщении 
15-0  Номер фрагмента
*/

const (
	addrMask =	0b0111_1100_0000_0000
	termMask = 	0b0000_0011_1110_0000
	//kMask = 	0b0000_00000_0001_0000
	//eMask = 	0b0000_00000_0000_1000
	kMask = 	0b0001_0000
	eMask = 	0b0000_1000
	msgHeaderSize = 8
	maxDataSize = 1364
	MaxPacketSize = msgHeaderSize +maxDataSize
)

type MsgHeader struct {
	//Addr uint16
	//Terminal uint16
	//PartsInMsg uint16
	LastOne bool
	Broken bool
	Msg uint16
	Part uint16 
	
}



func UnmarshallMsgHeader(buf []byte) (h *MsgHeader, err error) {

	if len(buf) < 8 {
		return h, errors.New("Buffer fewer than 8 bytes!")
	}
	
	h = new(MsgHeader)
	//These fields are not in use 
	//ush := uint16(buf[0]<<8) + uint16(buf[1])
	//h.Addr = (ush & addrMask) >> 10
	//h.Terminal = (ush & termMask) >> 4
	//h.PartsInMsg = uint16(buf[4]<<8) + uint16(buf[5])
	//h.ErrorFlag = (ush & eMask) != 0
	//h.LastPart = (ush & kMask) != 0
	ub := buf[1]
	h.Broken = (ub & eMask) != 0
	h.LastOne = (ub & kMask) != 0
	
	h.Msg = uint16(buf[2])<<8 + uint16(buf[3])
	h.Part = uint16(buf[6])<<8 + uint16(buf[7])
	
	return h, err 
}

func (h *MsgHeader) Marshall(buf *bytes.Buffer )  {
		var bb =make([]byte, msgHeaderSize)
		if h.Broken {
			bb[1] |= eMask 			
			}
		if h.LastOne {
			bb[1] |= kMask 			
			}
		bb[2] = byte((h.Msg& 0xFF00) >> 8)
		bb[3] = byte(h.Msg & 0xFF ) 
		bb[6]= byte((h.Part & 0xFF00) >> 8)
		bb[7] = byte(h.Part & 0xFF) 
		buf.Read(bb)
}

func (h *MsgHeader) String() string {
	return fmt.Sprintf("last:%t err:%t msg:%04x part:%04x", 
		h.LastOne, h.Broken, h.Msg,  h.Part)
}

func makeFileName(n uint16) string {
	return fmt.Sprintf("data/i%03x.msif", n)
}
//Splits big buffer to the number of part with size == maxDataSize
func Bb2Que(msgNumber uint16,  bb *bytes.Buffer, sp *bpool.SizedBufferPool, queue chan <- *bytes.Buffer )(err error) {
	hdr := MsgHeader{} 
	hdr.Msg = msgNumber
	tocopy := int64( bb.Len())
	var copied int64
	var part uint16
	
	for hdr.LastOne == false  {
		next := sp.Get()
		hdr.Part = part
		copied += maxDataSize
		if copied >= tocopy {
			hdr.LastOne = true
		}
		hdr.Marshall(next)
		_, err = io.CopyN(next, bb, maxDataSize)
		
		if err != nil && err != io.EOF{
			log.Println(err.Error())
			break 
		} 

		select {
			case queue <- next:
			default:
				log.Println("UPS!", part, tocopy, "->", copied)
			}			
		part+=1
	} 
	return err
} 

type MsifImage struct {
	MsgHeader //the last MsgHeader 
	bb bytes.Buffer
}

func NewMsifImage() (mi *MsifImage) {
	mi = new(MsifImage)
	mi.bb = bytes.Buffer{}	
	return mi
}

func(mi *MsifImage) Reset() {
	mi.bb.Reset()
	mi.LastOne = false 
	mi.Broken = false 
	mi.Msg = 0
	mi.Part = 0
} 

func (mi *MsifImage) NextChunk(buf []byte, size int)(err error){
	mh, err := UnmarshallMsgHeader(buf)
	log.Println(mh.String(), size)
	
	if mh.LastOne == true  { //Is end of image?
		//Is it the same msg and next part of it? 
		if mi.Msg == mh.Msg && (mi.Part + 1) == mh.Part {
			if mi.bb.Len() != 0 && mi.Broken == false { 
				mi.bb.Write(buf[msgHeaderSize:size]) //copy the last chunk
				err = ioutil.WriteFile(makeFileName(mi.Msg), mi.bb.Bytes(), 0644)	
				mi.Reset() 
				log.Println("save it -->")
			} else {
				log.Println("end of broken -->")
			}
		} else {
			mi.Reset() 
			log.Println("last in broken -->")
		}
	} else {
		//Is it the same msg?
		if mi.Msg == mh.Msg {
			if (mi.Part + 1 == mh.Part ) { //Is it the next chunk in row?
				mi.Part = mh.Part
				if mi.Broken == false{ //So far is good :)
						 mi.bb.Write(buf[msgHeaderSize:size]) //copy the next chunk
					}
		} else {
				mi.Part = mh.Part
				mi.Broken = true
				log.Println("wrong part number -->")
			}
		} else {
			mi.Reset()
			mi.Msg = mh.Msg 
			mi.Part = mh.Part
			log.Println("new msg -->")
			if mi.Part != 0 {
				mi.Broken = true
				log.Println("1st is not 0 -->")
			} else {
				mi.bb.Write(buf[msgHeaderSize:size]) //copy the 1st chunk

			}
		}
	} 

	return err
}
