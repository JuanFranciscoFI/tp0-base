package common

import (
	"encoding/binary"
	"fmt"
	"net"
)

type Protocol struct {
	skt net.Conn
}

const (
	statusOK       = uint8(0x00)
	statusFail     = uint8(0x01)
)

type Bet struct {
	Nombre     string
	Apellido   string
	Documento  uint32
	Nacimiento string
	Numero     uint32
}

func NewProtocol(skt net.Conn) *Protocol {
	return &Protocol{
		skt: skt,
	}
}

// acá me evito tener un short-write
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

func (p *Protocol) writeU16(v uint16) error {
	var buf [2]byte
	binary.BigEndian.PutUint16(buf[:], v)
	return p.writeAll(buf[:])
}

func (p *Protocol) writeStringU16(s string) error {
	b := []byte(s)

	if err := p.writeU16(uint16(len(b))); err != nil {
		return err
	}
	return p.writeAll(b)
}

func (p *Protocol) writeU32(v uint32) error {
	var buf [4]byte
	binary.BigEndian.PutUint32(buf[:], v)
	return p.writeAll(buf[:])
}

func (p *Protocol) SendBet(bet Bet) error {
	// nombre
	if err := p.writeStringU16(bet.Nombre); err != nil {
		return fmt.Errorf("write nombre: %w", err)
	}
	// apellido
	if err := p.writeStringU16(bet.Apellido); err != nil {
		return fmt.Errorf("write apellido: %w", err)
	}
	// documento
	if err := p.writeU32(bet.Documento); err != nil {
		return fmt.Errorf("write documento: %w", err)
	}
	// nacimiento
	if err := p.writeStringU16(bet.Nacimiento); err != nil {
		return fmt.Errorf("write nacimiento: %w", err)
	}
	// numero
	if err := p.writeU32(bet.Numero); err != nil {
		return fmt.Errorf("write numero: %w", err)
	}
	return nil
}

func (p *Protocol) RecvResponse() (uint8, error) {
	var one [1]byte
	_, err := p.skt.Read(one[:])
	if err != nil {
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
