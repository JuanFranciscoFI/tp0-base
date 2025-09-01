package common

import (
	"encoding/csv"
	"io"
	"net"
	"os"
	"strconv"
	"time"

	"github.com/op/go-logging"
)

var log = logging.MustGetLogger("log")

type ClientConfig struct {
	ID              string
	ServerAddress   string
	LoopAmount      int
	LoopPeriod      time.Duration
	DatasetPath     string
	BatchMaxAmount  int
}

type Client struct {
	config   ClientConfig
	conn     net.Conn
	protocol *Protocol
}

func NewClient(cfg ClientConfig) *Client { return &Client{config: cfg} }

func (c *Client) createClientSocket() error {
	conn, err := net.Dial("tcp", c.config.ServerAddress)
	if err != nil {
		log.Criticalf("action: connect | result: fail | client_id: %v | error: %v", c.config.ID, err)
		return err
	}
	c.conn = conn
	c.protocol = NewProtocol(conn)
	return nil
}

func (c *Client) Close() {
	if c.protocol != nil {
		_ = c.protocol.Close()
		log.Infof("action: close_connection | result: success")
		return
	}
	if c.conn != nil {
		_ = c.conn.Close()
		log.Infof("action: close_connection | result: success")
	}
}

func (c *Client) SendBatchesFromDataset() error {
	if err := c.createClientSocket(); err != nil {
		return err
	}

	defer c.Close()

	f, err := os.Open(c.config.DatasetPath)
	if err != nil {
		return err
	}
	defer f.Close()

	r := csv.NewReader(f)
	r.FieldsPerRecord = 5

	maxN := c.config.BatchMaxAmount

	var (
		batch   []Bet
		curSize = 2
	)

	flush := func() error {
		if len(batch) == 0 {
			return nil
		}
		if err := c.protocol.SendBatch(batch); err != nil {
			return err
		}
		if _, err := c.protocol.RecvResponse(); err != nil {
			return err
		}
		log.Infof("action: apuestas_enviadas | result: success | cantidad: %d", len(batch))
		batch = batch[:0]
		curSize = 2
		return nil
	}

	for {
		rec, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		doc64, _ := strconv.ParseUint(rec[2], 10, 32)
		num64, _ := strconv.ParseUint(rec[4], 10, 32)
		next := Bet{
			Nombre:     rec[0],
			Apellido:   rec[1],
			Documento:  uint32(doc64),
			Nacimiento: rec[3],
			Numero:     uint32(num64),
		}

		betSize := sizeOfBet(next)
		if len(batch) > 0 && (curSize+betSize > maxPacketBytes) {
			if err := flush(); err != nil {
				return err
			}
		}
		if len(batch) == maxN {
			if err := flush(); err != nil {
				return err
			}
		}

		batch = append(batch, next)
		curSize += betSize
	}

	return flush()
}