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
	statusOK        = uint8(0x00)
	statusFail      = uint8(0x01)
	maxPacketBytes  = 8192
)

func NewProtocol(skt net.Conn) *Protocol { return &Protocol{skt: skt} }


func (p *Protocol) writeAll(b []byte) error {
	for off := 0; off < len(b); {
		n, err := p.skt.Write(b[off:])
		if err != nil {
			return err
		}
		off += n
	}
	return nil
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

// Tamaño “real” en bytes del bet serializado (para respetar 8 KiB)
func sizeOfBet(b Bet) int {
	return 2 + len(b.Nombre) + // nombre (u16 + bytes)
		2 + len(b.Apellido) +  // apellido
		4 +                    // documento u32
		2 + len(b.Nacimiento) +// nacimiento
		4                      // numero u32
}

func sizeOfBatch(bets []Bet) int {
	size := 2
	for _, b := range bets {
		size += sizeOfBet(b)
	}
	return size
}

func EncodeBatch(bets []Bet) ([]byte, error) {
	if len(bets) > 0xFFFF {
		return nil, fmt.Errorf("batch too big (count %d > 65535)", len(bets))
	}
	total := sizeOfBatch(bets)
	if total > maxPacketBytes {
		return nil, fmt.Errorf("batch exceeds %d bytes (got %d)", maxPacketBytes, total)
	}

	buf := bytes.NewBuffer(make([]byte, 0, total))
	if err := putU16(buf, uint16(len(bets))); err != nil {
		return nil, fmt.Errorf("write count: %w", err)
	}
	for i, b := range bets {
		if err := encodeBetTo(buf, b); err != nil {
			return nil, fmt.Errorf("encode bet[%d]: %w", i, err)
		}
	}
	return buf.Bytes(), nil
}

func (p *Protocol) SendBatch(bets []Bet) error {
	payload, err := EncodeBatch(bets)
	if err != nil {
		return err
	}
	return p.writeAll(payload)
}

func (p *Protocol) RecvResponse() (uint8, error) {
	var one [1]byte
	if _, err := io.ReadFull(p.skt, one[:]); err != nil {
		return statusFail, err
	}
	if one[0] != statusOK {
		return statusFail, fmt.Errorf("status: %d", one[0])
	}
	return one[0], nil
}

func (p *Protocol) Close() error {
	if p.skt != nil {
		return p.skt.Close()
	}
	return nil
}