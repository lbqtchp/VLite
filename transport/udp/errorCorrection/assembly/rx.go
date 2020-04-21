package assembly

import (
	"bytes"
	"fmt"
	"github.com/boljen/go-bitmap"
	"github.com/lunixbochs/struc"
	"github.com/xiaokangwang/VLite/interfaces"
	"io/ioutil"
	"time"
)

type packetAssemblyRxChunkHolder struct {
	ef         interfaces.ErrorCorrectionFacility
	doneAll    bool
	doneBitmap bitmap.Bitmap
}

func (pa *PacketAssembly) Rx() {

	for {
		if pa.ctx.Err() != nil {
			fmt.Println(pa.ctx.Err().Error())
			return
		}
		inbuf := make([]byte, 1650)
		n, err := pa.conn.Read(inbuf)
		if err != nil {
			fmt.Println(err.Error())
			return
		}
		readendata := inbuf[:n]
		readendataReader := bytes.NewReader(readendata)
		pw := &PacketWireHead{}
		err = struc.Unpack(readendataReader, pw)
		if err != nil {
			fmt.Println(err.Error())
		}
		payload, err2 := ioutil.ReadAll(readendataReader)
		if err2 != nil {
			fmt.Println(err2.Error())
		}
		if pw.Seq == 0 {
			pa.RxChan <- payload
		} else {
			parch := &packetAssemblyRxChunkHolder{}
			parch.doneAll = false
			parch.doneBitmap = bitmap.New(pa.MaxDataShardPerChunk)
			parch.ef = pa.ecff.Create(pa.ctx)

			err = pa.RxReassembleBuffer.Add(string(pw.Seq), parch, time.Second*time.Duration(pa.RxMaxTimeInSecond))
			if err != nil {
				item, _ := pa.RxReassembleBuffer.Get(string(pw.Seq))
				parch = item.(*packetAssemblyRxChunkHolder)
			}

			if !parch.doneAll {

				done, data := parch.ef.AddShard(pw.Id, payload)
				if data != nil {
					pa.RxChan <- data
					parch.doneBitmap.Set(pw.Id, true)
				}
				if done {
					reconres := parch.ef.Reconstruct()
					if reconres != nil {
						for i, v := range reconres {
							if !parch.doneBitmap.Get(i) {
								pa.RxChan <- v
							}
						}
					}
					parch.doneAll = true
				}

			}
		}

	}

}