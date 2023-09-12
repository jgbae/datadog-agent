// Code generated by cmd/cgo -godefs; DO NOT EDIT.
// cgo -godefs -- -I ../../ebpf/c -I ../../../ebpf/c -fsigned-char offsetguess_types.go

package offsetguess

type Proc struct {
	Comm [16]int8
}

const ProcCommMaxLen = 0x10 - 1

type TracerOffsets struct {
	Saddr                    uint64
	Daddr                    uint64
	Sport                    uint64
	Dport                    uint64
	Netns                    uint64
	Ino                      uint64
	Family                   uint64
	Rtt                      uint64
	Rtt_var                  uint64
	Daddr_ipv6               uint64
	Saddr_fl4                uint64
	Daddr_fl4                uint64
	Sport_fl4                uint64
	Dport_fl4                uint64
	Saddr_fl6                uint64
	Daddr_fl6                uint64
	Sport_fl6                uint64
	Dport_fl6                uint64
	Socket_sk                uint64
	Sk_buff_sock             uint64
	Sk_buff_transport_header uint64
	Sk_buff_head             uint64
}
type TracerValues struct {
	Saddr                    uint32
	Daddr                    uint32
	Sport                    uint16
	Dport                    uint16
	Netns                    uint32
	Family                   uint16
	Rtt                      uint32
	Rtt_var                  uint32
	Daddr_ipv6               [4]uint32
	Saddr_fl4                uint32
	Daddr_fl4                uint32
	Sport_fl4                uint16
	Dport_fl4                uint16
	Saddr_fl6                [4]uint32
	Daddr_fl6                [4]uint32
	Sport_fl6                uint16
	Dport_fl6                uint16
	Sport_via_sk             uint16
	Dport_via_sk             uint16
	Sport_via_sk_via_sk_buff uint16
	Dport_via_sk_via_sk_buff uint16
	Transport_header         uint16
	Network_header           uint16
	Mac_header               uint16
	Pad_cgo_0                [2]byte
}
type TracerStatus struct {
	State              uint64
	What               uint64
	Err                uint64
	Proc               Proc
	Info_kprobe_status uint64
	Offsets            TracerOffsets
	Values             TracerValues
	Pad_cgo_0          [4]byte
}

type State uint8

const (
	StateUninitialized State = 0x0
	StateChecking      State = 0x1
	StateChecked       State = 0x2
	StateReady         State = 0x3
)

type ConntrackOffsets struct {
	Origin uint64
	Reply  uint64
	Status uint64
	Netns  uint64
	Ino    uint64
}
type ConntrackValues struct {
	Saddr  uint32
	Daddr  uint32
	Status uint32
	Netns  uint32
}
type ConntrackStatus struct {
	State   uint64
	What    uint64
	Err     uint64
	Proc    Proc
	Offsets ConntrackOffsets
	Values  ConntrackValues
}
type ConntrackState uint8

type GuessWhat uint64

const (
	GuessSAddr     GuessWhat = 0x0
	GuessDAddr     GuessWhat = 0x1
	GuessFamily    GuessWhat = 0x2
	GuessSPort     GuessWhat = 0x3
	GuessDPort     GuessWhat = 0x4
	GuessNetNS     GuessWhat = 0x5
	GuessRTT       GuessWhat = 0x6
	GuessDAddrIPv6 GuessWhat = 0x7

	GuessSAddrFl4 GuessWhat = 0x8
	GuessDAddrFl4 GuessWhat = 0x9
	GuessSPortFl4 GuessWhat = 0xa
	GuessDPortFl4 GuessWhat = 0xb

	GuessSAddrFl6   GuessWhat = 0xc
	GuessDAddrFl6   GuessWhat = 0xd
	GuessSPortFl6   GuessWhat = 0xe
	GuessDPortFl6   GuessWhat = 0xf
	GuessSocketSK   GuessWhat = 0x10
	GuessSKBuffSock GuessWhat = 0x11

	GuessSKBuffTransportHeader GuessWhat = 0x12
	GuessSKBuffHead            GuessWhat = 0x13

	GuessCtTupleOrigin GuessWhat = 0x14
	GuessCtTupleReply  GuessWhat = 0x15
	GuessCtStatus      GuessWhat = 0x16
	GuessCtNet         GuessWhat = 0x17

	GuessNotApplicable GuessWhat = 99999
)
