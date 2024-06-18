// Copyright 2024 the u-root Authors. All rights reserved
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package traceroute

import (
	"bytes"
	"encoding/binary"
	"log"

	"golang.org/x/net/ipv4"
)

type UDPHeader struct {
	Src    uint16
	Dst    uint16
	Length uint16
	Chksum uint16
}

const (
	TCP_FIN = 1 << 0
	TCP_SYN = 1 << 1
	TCP_RST = 1 << 2
	TCP_PSH = 1 << 3
	TCP_ACK = 1 << 4
	TCP_URG = 1 << 5
)

type TCPHeader struct {
	Src        uint16
	Dst        uint16
	SeqNum     uint32
	AckNum     uint32
	DataOffset uint8 // only use high 4 bits
	Flags      uint8 // only use low 6 bits
	Window     uint16
	Checksum   uint16
	Urgent     uint16
}

// checksum function
func checkSum(buf []byte) uint16 {
	sum := uint32(0)

	for ; len(buf) >= 2; buf = buf[2:] {
		sum += uint32(buf[0])<<8 | uint32(buf[1])
	}
	if len(buf) > 0 {
		sum += uint32(buf[0]) << 8
	}
	for sum > 0xffff {
		sum = (sum >> 16) + (sum & 0xffff)
	}
	csum := ^uint16(sum)
	/*
	 * From RFC 768:
	 * If the computed checksum is zero, it is transmitted as all ones (the
	 * equivalent in one's complement arithmetic). An all zero transmitted
	 * checksum value means that the transmitter generated no checksum (for
	 * debugging or for higher level protocols that don't care).
	 */
	if csum == 0 {
		csum = 0xffff
	}
	return csum
}

func (u *UDPHeader) checksum(ip *ipv4.Header, payload []byte) {
	var pseudoHeader []byte

	pseudoHeader = append(pseudoHeader, ip.Src.To4()...)
	pseudoHeader = append(pseudoHeader, ip.Dst.To4()...)
	pseudoHeader = append(pseudoHeader, []byte{
		0,
		17,
		0, byte(u.Length),
	}...)

	var b bytes.Buffer
	binary.Write(&b, binary.BigEndian, pseudoHeader)
	binary.Write(&b, binary.BigEndian, u)
	binary.Write(&b, binary.BigEndian, &payload)
	u.Chksum = checkSum(b.Bytes())
}

func (t *TCPHeader) checksum(ip *ipv4.Header, payload []byte) {
	var pseudoHeader []byte

	pseudoHeader = append(pseudoHeader, ip.Src.To4()...)
	pseudoHeader = append(pseudoHeader, ip.Dst.To4()...)
	pseudoHeader = append(pseudoHeader, []byte{
		0,
		6,
		0, byte(len(payload) + 20),
	}...)

	var b bytes.Buffer
	binary.Write(&b, binary.BigEndian, pseudoHeader)
	binary.Write(&b, binary.BigEndian, t)
	binary.Write(&b, binary.BigEndian, &payload)
	t.Checksum = checkSum(b.Bytes())
}

func (u *UDP4Trace) BuildIPv4UDPkt(srcPort uint16, dstPort uint16, ttl uint8, id uint16, tos int) (*ipv4.Header, []byte) {
	iph := &ipv4.Header{
		Version:  ipv4.Version,
		TOS:      tos,
		Len:      ipv4.HeaderLen,
		TotalLen: 60,
		ID:       int(id),
		Flags:    0,
		FragOff:  0,
		TTL:      int(ttl),
		Protocol: 17,
		Checksum: 0,
		Src:      u.src,
		Dst:      u.Dest,
	}

	h, err := iph.Marshal()
	if err != nil {
		log.Fatal(err)
	}
	iph.Checksum = int(checkSum(h))

	udp := UDPHeader{
		Src: srcPort,
		Dst: dstPort,
	}

	payload := make([]byte, 32)
	for i := 0; i < 32; i++ {
		payload[i] = uint8(i + 64)
	}
	udp.Length = uint16(len(payload) + 8)
	udp.checksum(iph, payload)

	var buf bytes.Buffer
	binary.Write(&buf, binary.BigEndian, &udp)
	binary.Write(&buf, binary.BigEndian, &payload)
	return iph, buf.Bytes()
}
