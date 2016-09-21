// Go bindings for the NFQUEUE netfilter target
// libnetfilter_queue is a userspace library providing an API to access packets
// that have been queued by the Linux kernel packet filter.
//
// This provides an easy way to filter packets from userspace, and use tools
// or libraries that are not accessible from kernelspace.
//
// BUG(nfqueue): This package currently displays lots of debug information
package nfqueue

// XXX we should use something like
// pkg-config --libs libnetfilter_queue

// #cgo pkg-config: libnetfilter_queue
/*
#include <stdio.h>
#include <stdint.h>
#include <arpa/inet.h>
#include <linux/netfilter.h>
#include <libnetfilter_queue/libnetfilter_queue.h>

extern int GoCallbackWrapper(void *data, void *nfad);

int _process_loop(struct nfq_handle *h,
                  int fd,
                  int flags,
                  int max_count) {
        int rv;
        char buf[65535];
        int count;

        count = 0;

        while ((rv = recv(fd, buf, sizeof(buf), flags)) >= 0) {
                nfq_handle_packet(h, buf, rv);
                count++;
                if (max_count > 0 && count >= max_count) {
                        break;
                }
        }
        return count;
}

int c_nfq_cb(struct nfq_q_handle *qh,
             struct nfgenmsg *nfmsg,
             struct nfq_data *nfad, void *data) {
    return GoCallbackWrapper(data, nfad);
}

// wrap nfq_get_payload so cgo always have the same prototype
// (libnetfilter_queue 0.17 uses a signed char)
static int _c_get_payload (struct nfq_data *nfad, unsigned char **data)
{
    return nfq_get_payload (nfad, data);
}
*/
import "C"

import (
	"errors"
	// "fmt"
	"log"
	"unsafe"
)

var ErrNotInitialized = errors.New("nfqueue: queue not initialized")
var ErrOpenFailed = errors.New("nfqueue: open failed")
var ErrRuntime = errors.New("nfqueue: runtime error")

var NF_DROP = C.NF_DROP
var NF_ACCEPT = C.NF_ACCEPT
var NF_QUEUE = C.NF_QUEUE
var NF_REPEAT = C.NF_REPEAT
var NF_STOP = C.NF_STOP

var NFQNL_COPY_NONE uint8 = C.NFQNL_COPY_NONE
var NFQNL_COPY_META uint8 = C.NFQNL_COPY_META
var NFQNL_COPY_PACKET uint8 = C.NFQNL_COPY_PACKET

// Prototype for a NFQUEUE callback.
// The callback receives the NFQUEUE ID of the packet, and
// the packet payload.
// Packet data start from the IP layer (ethernet information are not included).
// It must return the verdict for the packet.
type Callback func(*Payload) int

// Queue is an opaque structure describing a connection to a kernel NFQUEUE,
// and the associated Go callback.
type Queue struct {
	c_h  (*C.struct_nfq_handle)
	c_qh (*C.struct_nfq_q_handle)

	cb Callback
}

// Init creates a netfilter queue which can be used to receive packets
// from the kernel.
func (q *Queue) Init() error {
	log.Println("Opening queue")
	q.c_h = C.nfq_open()
	if q.c_h == nil {
		log.Println("nfq_open failed")
		return ErrOpenFailed
	}
	return nil
}

// SetCallback sets the callback function, fired when a packet is received.
func (q *Queue) SetCallback(cb Callback) error {
	q.cb = cb
	return nil
}

func (q *Queue) Close() {
	if q.c_h != nil {
		log.Println("Closing queue")
		C.nfq_close(q.c_h)
		q.c_h = nil
	}
}

// Bind binds a Queue to a given protocol family.
//
// Usually, the family is syscall.AF_INET for IPv4, and syscall.AF_INET6 for IPv6
func (q *Queue) Bind(af_family int) error {
	if q.c_h == nil {
		return ErrNotInitialized
	}
	log.Println("Binding to selected family")
	/* Errors in nfq_bind_pf are non-fatal ...
	 * This function just tells the kernel that nfnetlink_queue is
	 * the chosen module to queue packets to userspace.
	 */
	_ = C.nfq_bind_pf(q.c_h, C.u_int16_t(af_family))
	return nil
}

// Unbind a queue from the given protocol family.
//
// Note that errors from this function can usually be ignored.
func (q *Queue) Unbind(af_family int) error {
	if q.c_h == nil {
		return ErrNotInitialized
	}
	log.Println("Unbinding to selected family")
	rc := C.nfq_unbind_pf(q.c_h, C.u_int16_t(af_family))
	if rc < 0 {
		log.Println("nfq_unbind_pf failed")
		return ErrRuntime
	}
	return nil
}

// Create a new queue handle
//
// The queue must be initialized (using Init) and bound (using Bind), and
// a callback function must be set (using SetCallback).
func (q *Queue) CreateQueue(queue_num int) error {
	if q.c_h == nil {
		return ErrNotInitialized
	}
	if q.cb == nil {
		return ErrNotInitialized
	}
	log.Println("Creating queue")
	q.c_qh = C.nfq_create_queue(q.c_h, C.u_int16_t(queue_num), (*C.nfq_callback)(C.c_nfq_cb), unsafe.Pointer(q))
	if q.c_qh == nil {
		log.Println("nfq_create_queue failed")
		return ErrRuntime
	}
	// Default mode
	C.nfq_set_mode(q.c_qh, C.NFQNL_COPY_PACKET, 0xffff)
	return nil
}

// SetMode sets the amount of packet data that nfqueue copies to userspace
//
// Default mode is NFQNL_COPY_PACKET
func (q *Queue) SetMode(mode uint8) error {
	if q.c_h == nil {
		return ErrNotInitialized
	}
	if q.c_qh == nil {
		return ErrNotInitialized
	}
	C.nfq_set_mode(q.c_qh, C.u_int8_t(mode), 0xffff)
	return nil
}

// Main loop: TryRun starts an infinite loop, receiving kernel events
// and processing packets using the callback function.
//
// BUG(TryRun): The TryRun function really is an infinite loop.
func (q *Queue) TryRun() error {
	if q.c_h == nil {
		return ErrNotInitialized
	}
	if q.c_qh == nil {
		return ErrNotInitialized
	}
	if q.cb == nil {
		return ErrNotInitialized
	}
	log.Println("Try Run")
	fd := C.nfq_fd(q.c_h)
	if fd < 0 {
		log.Println("nfq_fd failed")
		return ErrRuntime
	}
	// XXX
	C._process_loop(q.c_h, fd, 0, -1)
	return nil
}

// Payload is a structure describing a packet received from the kernel
type Payload struct {
	c_qh (*C.struct_nfq_q_handle)
	nfad *C.struct_nfq_data

	// NFQueue ID of the packet
	Id uint32
	// Packet data
	Data []byte
}

func build_payload(c_qh *C.struct_nfq_q_handle, ptr_nfad *unsafe.Pointer) *Payload {
	var payload_data *C.uchar
	var data []byte

	nfad := (*C.struct_nfq_data)(unsafe.Pointer(ptr_nfad))

	ph := C.nfq_get_msg_packet_hdr(nfad)
	id := C.ntohl(C.uint32_t(ph.packet_id))
	payload_len := C._c_get_payload(nfad, &payload_data)
	if payload_len >= 0 {
		data = C.GoBytes(unsafe.Pointer(payload_data), C.int(payload_len))
	}

	p := new(Payload)
	p.c_qh = c_qh
	p.nfad = nfad
	p.Id = uint32(id)
	p.Data = data

	return p
}

// SetVerdict issues a verdict for a packet.
//
// Every queued packet _must_ have a verdict specified by userspace.
func (p *Payload) SetVerdict(verdict int) error {
	// log.Printf("Setting verdict for packet %d: %d\n", p.Id, verdict)
	// fmt.Printf(" √\n")
	C.nfq_set_verdict(p.c_qh, C.u_int32_t(p.Id), C.u_int32_t(verdict), 0, nil)
	return nil
}

// SetVerdictModified issues a verdict for a packet, but replaces the packet
// with the provided one.
//
// Every queued packet _must_ have a verdict specified by userspace.
func (p *Payload) SetVerdictModified(verdict int, data []byte) error {
	log.Printf("Setting verdict for NEW packet %d: %d\n", p.Id, verdict)
	C.nfq_set_verdict(
		p.c_qh,
		C.u_int32_t(p.Id),
		C.u_int32_t(verdict),
		C.u_int32_t(len(data)),
		(*C.uchar)(unsafe.Pointer(&data[0])),
	)
	return nil
}

// Returns the packet mark
func (p *Payload) GetNFMark() uint32 {
	return uint32(C.nfq_get_nfmark(p.nfad))
}

// Returns the interface that the packet was received through
func (p *Payload) GetInDev() uint32 {
	return uint32(C.nfq_get_indev(p.nfad))
}

// Returns the interface that the packet will be routed out
func (p *Payload) GetOutDev() uint32 {
	return uint32(C.nfq_get_outdev(p.nfad))
}

// Returns the physical interface that the packet was received through
func (p *Payload) GetPhysInDev() uint32 {
	return uint32(C.nfq_get_physindev(p.nfad))
}

// Returns the physical interface that the packet will be routed out
func (p *Payload) GetPhysOutDev() uint32 {
	return uint32(C.nfq_get_physoutdev(p.nfad))
}
