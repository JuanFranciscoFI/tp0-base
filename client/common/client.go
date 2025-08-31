package common

import (
	"net"
	"time"

	"github.com/op/go-logging"
)

var log = logging.MustGetLogger("log")

// ClientConfig Configuration used by the client
type ClientConfig struct {
	ID            string
	ServerAddress string
	LoopAmount    int
	LoopPeriod    time.Duration
}

// Client Entity that encapsulates how
type Client struct {
	config ClientConfig
	conn   net.Conn
	protocol *Protocol
}

// NewClient Initializes a new client receiving the configuration
// as a parameter
func NewClient(config ClientConfig) *Client {
	client := &Client{
		config: config,
		protocol: nil,
	}
	return client
}

// CreateClientSocket Initializes client socket. In case of
// failure, error is printed in stdout/stderr and exit 1
// is returned
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

// StartClientLoop Send messages to the client until some time threshold is met
func (c *Client) StartClientLoop(bet Bet) {
	if err := c.createClientSocket(); err != nil {
		log.Fatalf("Error al crear el socket: %v", err)
	}
	defer c.protocol.Close()

	if err := c.protocol.SendBet(bet); err != nil {
		log.Errorf("action: apuesta_enviada | result: fail | error: %v", err)
		return
	}

	_, err := c.protocol.RecvResponse()
	if err != nil {
		log.Errorf("action: apuesta_enviada | result: fail | error: %v", err)
		return
	}

	log.Infof("action: apuesta_enviada | result: success | dni: %v | numero: %v", bet.Documento, bet.Numero)
}

// Close the client socket
func (c *Client) Close() {
	if c.conn != nil {
		log.Infof("action: close_connection | result: success")
		c.conn.Close()
	}
}

