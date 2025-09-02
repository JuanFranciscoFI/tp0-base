package common

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"net"
)

type Protocol struct{ skt net.Conn }

const (
	MSG_HELLO        = uint8(0x01)
	MSG_BATCH        = uint8(0x02)
	MSG_DONE         = uint8(0x03)

	statusOK       = uint8(0x00)
	statusFail     = uint8(0x01)
	maxPacketBytes = 8192
)

func NewProtocol(skt net.Conn) *Protocol { return &Protocol{skt: skt} }

func (p *Protocol) writeAll(b []byte) error {
	for off := 0; off < len(b); {
		n, err := p.skt.Write(b[off:])
		if err != nil {
			return err
		}
		if n == 0 {
			return fmt.Errorf("short write: wrote 0 bytes")
		}
		off += n
	}
	return nil
}

func (p *Protocol) writeByte(v byte) error {
	var one [1]byte
	one[0] = v
	return p.writeAll(one[:])
}

func readFull(conn net.Conn, b []byte) error {
	_, err := io.ReadFull(conn, b)
	return err
}

func putU16(buf *bytes.Buffer, v uint16) error {
	var tmp [2]byte
	binary.BigEndian.PutUint16(tmp[:], v)
	_, err := buf.Write(tmp[:])
	return err
}
func putU32(buf *bytes.Buffer, v uint32) error {
	var tmp [4]byte
	binary.BigEndian.PutUint32(tmp[:], v)
	_, err := buf.Write(tmp[:])
	return err
}
func putStringU16(buf *bytes.Buffer, s string) error {
	b := []byte(s)
	if err := putU16(buf, uint16(len(b))); err != nil {
		return err
	}
	_, err := buf.Write(b)
	return err
}
func encodeBetTo(buf *bytes.Buffer, b Bet) error {
	if err := putStringU16(buf, b.Nombre); err != nil { return fmt.Errorf("nombre: %w", err) }
	if err := putStringU16(buf, b.Apellido); err != nil { return fmt.Errorf("apellido: %w", err) }
	if err := putU32(buf, b.Documento); err != nil { return fmt.Errorf("documento: %w", err) }
	if err := putStringU16(buf, b.Nacimiento); err != nil { return fmt.Errorf("nacimiento: %w", err) }
	if err := putU32(buf, b.Numero); err != nil { return fmt.Errorf("numero: %w", err) }
	return nil
}
func sizeOfBet(b Bet) int {
	return 2 + len(b.Nombre) + 2 + len(b.Apellido) + 4 + 2 + len(b.Nacimiento) + 4
}
func sizeOfBatchPayload(bets []Bet) int {
	n := 2
	for _, b := range bets {
		n += sizeOfBet(b)
	}
	return n
}

func (p *Protocol) SendHello(agencyID uint16) error {
	if err := p.writeByte(MSG_HELLO); err != nil { return err }
	var tmp [2]byte
	binary.BigEndian.PutUint16(tmp[:], agencyID)
	return p.writeAll(tmp[:])
}

func (p *Protocol) SendBatch(bets []Bet) error {
	payloadSize := sizeOfBatchPayload(bets)
	if 1+payloadSize > maxPacketBytes {
		return fmt.Errorf("batch exceeds %d bytes (got %d incl. type)", maxPacketBytes, 1+payloadSize)
	}
	buf := bytes.NewBuffer(make([]byte, 0, payloadSize))
	if err := putU16(buf, uint16(len(bets))); err != nil { return err }
	for i, b := range bets {
		if err := encodeBetTo(buf, b); err != nil {
			return fmt.Errorf("encode bet[%d]: %w", i, err)
		}
	}
	if err := p.writeByte(MSG_BATCH); err != nil { return err }
	return p.writeAll(buf.Bytes())
}
func (p *Protocol) SendDone() error { return p.writeByte(MSG_DONE) }

func (p *Protocol) RecvAck() error {
	var b [1]byte
	if err := readFull(p.skt, b[:]); err != nil { return err }
	if b[0] != statusOK {
		return fmt.Errorf("status: %d", b[0])
	}
	return nil
}

func (p *Protocol) RecvWinners() ([]uint32, error) {
	var st [1]byte
	if err := readFull(p.skt, st[:]); err != nil { return nil, err }
	if st[0] != statusOK {
		return nil, fmt.Errorf("not_ready")
	}
	var h [2]byte
	if err := readFull(p.skt, h[:]); err != nil { return nil, err }
	count := binary.BigEndian.Uint16(h[:])
	if count == 0 {
		return []uint32{}, nil
	}
	raw := make([]byte, int(count)*4)
	if err := readFull(p.skt, raw); err != nil { return nil, err }
	out := make([]uint32, int(count))
	for i := range out {
		off := i * 4
		out[i] = binary.BigEndian.Uint32(raw[off : off+4])
	}
	return out, nil
}

func (p *Protocol) Close() error {
	if p.skt != nil { return p.skt.Close() }
	return nil
}
