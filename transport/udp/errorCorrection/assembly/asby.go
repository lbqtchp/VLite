package assembly

import (
	"context"
	"github.com/patrickmn/go-cache"
	"github.com/xiaokangwang/VLite/interfaces"
	"github.com/xiaokangwang/VLite/transport/http/adp"
	rsf "github.com/xiaokangwang/VLite/transport/udp/errorCorrection/reconstruction/reedSolomon"
	"io"
	"net"
	"time"
)

func NewPacketAssembly(ctx context.Context, conn net.Conn) *PacketAssembly {
	pa := &PacketAssembly{}
	pa.ctx = ctx
	pa.conn = conn

	pa.RxMaxTimeInSecond = 4
	pa.TxNextSeq = 1
	pa.TxRingBufferSize = 30
	pa.TxRingBuffer = make([]packetAssemblyTxChunkHolder, pa.TxRingBufferSize)
	pa.MaxDataShardPerChunk = 40
	pa.TxFECSoftPacketSoftLimitPerEpoch = 40
	pa.TxEpochTimeInMs = 35
	pa.RxChan = make(chan []byte, 8)
	pa.TxChan = make(chan []byte, 8)
	pa.TxNoFECChan = make(chan []byte, 8)

	pa.FECEnabled = 1

	pa.ecff = rsf.NewRSErrorCorrectionFacilityFactory()

	pa.RxReassembleBuffer = cache.New(
		time.Second*time.Duration(pa.RxMaxTimeInSecond),
		4*time.Second*time.Duration(pa.RxMaxTimeInSecond))

	eov := ctx.Value(interfaces.ExtraOptionsFECPacketAssemblyOpt)
	if eov != nil {
		eovs := eov.(*interfaces.ExtraOptionsFECPacketAssemblyOptValue)
		pa.RxMaxTimeInSecond = eovs.RxMaxTimeInSecond
		pa.TxEpochTimeInMs = eovs.TxEpochTimeInMs
	}

	go pa.Rx()
	go pa.Tx()

	return pa
}

type PacketAssembly struct {
	RxReassembleBuffer *cache.Cache

	TxNextSeq        uint32
	TxRingBuffer     []packetAssemblyTxChunkHolder
	TxRingBufferSize int

	RxChan      chan []byte
	TxChan      chan []byte
	TxNoFECChan chan []byte

	conn io.ReadWriteCloser

	ecff interfaces.ErrorCorrectionFacilityFactory

	ctx context.Context

	MaxDataShardPerChunk int
	RxMaxTimeInSecond    int

	FECEnabled uint32

	TxEpochTimeInMs                  int
	TxFECSoftPacketSoftLimitPerEpoch int
}

func (pa *PacketAssembly) Close() error {
	return pa.conn.Close()
}

type PacketWireHead struct {
	Seq uint32 `struc:"uint32"`
	Id  int    `struc:"uint16"`
}

func (pa *PacketAssembly) AsConn() net.Conn {
	return adp.NewRxTxToConn(pa.TxChan, pa.RxChan, pa)
}
