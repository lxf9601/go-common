package amqp

import (
	"errors"
	"time"

	"github.com/lxf9601/go-common/logc"

	"github.com/streadway/amqp"
)

type Session struct {
	name            string
	connection      *amqp.Connection
	channel         *amqp.Channel
	done            chan bool
	notifyConnClose chan *amqp.Error
	notifyChanClose chan *amqp.Error
	notifyConfirm   chan amqp.Confirmation
	isReady         bool
}

const (
	// When reconnecting to the server after connection failure
	reconnectDelay = 5 * time.Second

	// When setting up the channel after a channel exception
	reInitDelay = 2 * time.Second

	// When resending messages the server didn't confirm
	resendDelay = 5 * time.Second
)

var (
	errNotConnected  = errors.New("not connected to a server")
	errAlreadyClosed = errors.New("already closed: not connected to the server")
	errShutdown      = errors.New("session is shutting down")
)

// New creates a new consumer state instance, and automatically
// attempts to connect to the server.
func New(name string, addr string) *Session {
	session := Session{
		name: name,
		done: make(chan bool),
	}
	go session.handleReconnect(addr)
	return &session
}

// handleReconnect will wait for a connection error on
// notifyConnClose, and then continuously attempt to reconnect.
func (session *Session) handleReconnect(addr string) {
	for {
		session.isReady = false
		logc.Info("Amqp Attempting to connect")

		conn, err := session.connect(addr)

		if err != nil {
			logc.Errorf("Amqp Failed to connect. Retrying... %s", err)

			select {
			case <-session.done:
				return
			case <-time.After(reconnectDelay):
			}
			continue
		}

		if done := session.handleReInit(conn); done {
			break
		}
	}
}

// connect will create a new AMQP connection
func (session *Session) connect(addr string) (*amqp.Connection, error) {
	conn, err := amqp.DialConfig(addr, amqp.Config{
		Heartbeat: 30,
		Locale:    "en_US",
	})

	if err != nil {
		return nil, err
	}

	session.changeConnection(conn)
	logc.Info("Amqp Connected!")
	return conn, nil
}

// handleReconnect will wait for a channel error
// and then continuously attempt to re-initialize both channels
func (session *Session) handleReInit(conn *amqp.Connection) bool {
	for {
		session.isReady = false

		err := session.init(conn)

		if err != nil {
			logc.Errorf("Amqp Failed to initialize channel. Retrying... %s", err)

			select {
			case <-session.done:
				return true
			case <-time.After(reInitDelay):
			}
			continue
		}

		select {
		case <-session.done:
			return true
		case <-session.notifyConnClose:
			logc.Info("Amqp Connection closed. Reconnecting...")
			return false
		case <-session.notifyChanClose:
			logc.Info("Amqp Channel closed. Re-running init...")
		}
	}
}

// init will initialize channel & declare queue
func (session *Session) init(conn *amqp.Connection) error {
	ch, err := conn.Channel()

	if err != nil {
		return err
	}

	err = ch.Confirm(false)

	if err != nil {
		return err
	}
	_, err = ch.QueueDeclare(
		session.name,
		false, // Durable
		false, // Delete when unused
		false, // Exclusive
		false, // No-wait
		nil,   // Arguments
	)

	if err != nil {
		return err
	}

	session.changeChannel(ch)
	session.isReady = true
	logc.Info("Amap Setup!")

	return nil
}

// changeConnection takes a new connection to the queue,
// and updates the close listener to reflect this.
func (session *Session) changeConnection(connection *amqp.Connection) {
	session.connection = connection
	session.notifyConnClose = make(chan *amqp.Error)
	session.connection.NotifyClose(session.notifyConnClose)
}

// changeChannel takes a new channel to the queue,
// and updates the channel listeners to reflect this.
func (session *Session) changeChannel(channel *amqp.Channel) {
	session.channel = channel
	session.notifyChanClose = make(chan *amqp.Error)
	session.notifyConfirm = make(chan amqp.Confirmation, 1)
	session.channel.NotifyClose(session.notifyChanClose)
	session.channel.NotifyPublish(session.notifyConfirm)
}

// Push will push data onto the queue, and wait for a confirm.
// If no confirms are received until within the resendTimeout,
// it continuously re-sends messages until a confirm is received.
// This will block until the server sends a confirm. Errors are
// only returned if the push action itself fails, see UnsafePush.
func (session *Session) Push(key string, data []byte) error {
	if !session.isReady {
		return errors.New("failed to push push: not connected")
	}
	for {
		err := session.UnsafePush(key, data)
		if err != nil {
			logc.Errorf("Amqp Push failed. Retrying... %s", err)
			select {
			case <-session.done:
				return errShutdown
			case <-time.After(resendDelay):
			}
			continue
		}
		select {
		case confirm := <-session.notifyConfirm:
			if confirm.Ack {
				logc.Info("Amqp Push confirmed!")
				return nil
			}
		case <-time.After(resendDelay):
		}
		logc.Info("Amqp Push didn't confirm. Retrying...")
	}
}

// UnsafePush will push to the queue without checking for
// confirmation. It returns an error if it fails to connect.
// No guarantees are provided for whether the server will
// recieve the message.
func (session *Session) UnsafePush(key string, data []byte) error {
	if !session.isReady {
		return errNotConnected
	}
	return session.channel.Publish(
		"",    // Exchange
		key,   // Routing key
		false, // Mandatory
		false, // Immediate
		amqp.Publishing{
			ContentType: "text/plain",
			Body:        data,
		},
	)
}

// Stream will continuously put queue items on the channel.
// It is required to call delivery.Ack when it has been
// successfully processed, or delivery.Nack when it fails.
// Ignoring this will cause data to build up on the server.
func (session *Session) Stream() (<-chan amqp.Delivery, error) {
	if !session.isReady {
		return nil, errNotConnected
	}
	return session.channel.Consume(
		session.name,
		"",    // Consumer
		false, // Auto-Ack
		false, // Exclusive
		false, // No-local
		true,  // No-Wait
		nil,   // Args
	)
}

// session is ready
func (session *Session) IsReady() bool {
	return session.isReady
}

// session get connection
func (session *Session) GetConnection() *amqp.Connection {
	return session.connection
}

// get channel
func (session *Session) GetChannel() *amqp.Channel {
	return session.channel
}

// new session
func (session *Session) New(name string) *Session {
	newSession := &Session{
		name: name,
		done: make(chan bool),
	}
	newSession.connection = session.connection
	newSession.notifyConnClose = make(chan *amqp.Error)
	newSession.connection.NotifyClose(session.notifyConnClose)
	go newSession.handleReInit(session.connection)
	return newSession
}

// Close will cleanly shutdown the channel and connection.
func (session *Session) Close() error {
	if !session.isReady {
		return errAlreadyClosed
	}
	session.channel.Close()
	if !session.connection.IsClosed() {
		session.connection.Close()
	}
	close(session.done)
	session.isReady = false
	return nil
}
