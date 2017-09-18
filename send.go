package gosocketio

import (
	"encoding/json"
	"errors"
	"log"
	"strings"
	"time"

	"github.com/Omegaice/golang-socketio/protocol"
)

var (
	ErrorSendTimeout     = errors.New("Timeout")
	ErrorSocketOverflood = errors.New("Socket overflood")
)

/**
Send message packet to socket
*/
func send(msg *protocol.Message, c *Channel) error {
	command, err := protocol.Encode(msg)
	if err != nil {
		return err
	}

	if len(c.out) == queueBufferSize {
		return ErrorSocketOverflood
	}

	c.out <- command

	return nil
}

/**
Create packet based on given data and send it
*/
func (c *Channel) Emit(method string, args interface{}) error {
	//preventing json/encoding "index out of range" panic
	defer func() {
		if r := recover(); r != nil {
			log.Println("socket.io send panic: ", r)
		}
	}()

	msg := &protocol.Message{
		Type:   protocol.MessageTypeEmit,
		Method: method,
	}

	if args != nil {
		json, err := json.Marshal(&args)
		if err != nil {
			return err
		}

		msg.Args = string(json)
	}

	return send(msg, c)
}

/**
Create ack packet based on given data and send it and receive response
*/
func (c *Channel) Ack(method string, timeout time.Duration, args ...interface{}) (string, error) {
	//preventing json/encoding "index out of range" panic
	defer func() {
		if r := recover(); r != nil {
			log.Println("socket.io send panic: ", r)
		}
	}()

	msg := &protocol.Message{
		Type:   protocol.MessageTypeAckRequest,
		AckId:  c.ack.getNextId(),
		Method: method,
	}

	if args != nil {
		var argString []string
		for _, arg := range args {
			json, err := json.Marshal(&arg)
			if err != nil {
				return "", err
			}
			argString = append(argString, string(json))
		}
		msg.Args = strings.Join(argString, ",")
	}

	waiter := make(chan string)
	c.ack.addWaiter(msg.AckId, waiter)

	err := send(msg, c)
	if err != nil {
		c.ack.removeWaiter(msg.AckId)
	}

	select {
	case result := <-waiter:
		return result, nil
	case <-time.After(timeout):
		c.ack.removeWaiter(msg.AckId)
		return "", ErrorSendTimeout
	}
}
