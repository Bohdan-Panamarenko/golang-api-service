package rabbitMQ

import (
	"api-service/api"
	"api-service/logging"
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"time"

	"github.com/streadway/amqp"
)

type BrockerMessage struct {
	Msg string
}

func PrintError(err error, msg string) {
	log.Printf("%s: %s\n", msg, err)
}

func ConnectToRebbitMQ() (*amqp.Queue, error) {
	conn, err := amqp.Dial("amqp://guest:guest@localhost:5672/")
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	ch, err := conn.Channel()
	if err != nil {
		return nil, err
	}
	defer ch.Close()

	q, err := ch.QueueDeclare(
		"hello", // name
		false,   // durable
		false,   // delete when unused
		false,   // exclusive
		false,   // no-wait
		nil,     // arguments
	)
	if err != nil {
		return nil, err
	}

	return &q, nil
}

type BrockerWriter logging.LogWriter

func (w *BrockerWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	h, ok := w.ResponseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, errors.New("hijack not supported")
	}
	return h.Hijack()
}

func (w *BrockerWriter) WriteHeader(status int) {
	w.ResponseWriter.WriteHeader(status)
	w.StatusCode = status
}

func (w *BrockerWriter) Write(p []byte) (int, error) {
	w.Response.Write(p)
	return w.ResponseWriter.Write(p)
}

func LogRequest(msgs chan<- BrockerMessage, h http.HandlerFunc) http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		writer := &BrockerWriter{
			ResponseWriter: rw,
		}

		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			log.Println("Could not read request body", err)
			api.HandleError(errors.New("Could not read requst"), rw)

			return
		}
		r.Body = ioutil.NopCloser(bytes.NewBuffer(body))

		started := time.Now()
		h(writer, r)
		done := time.Since(started)

		msg := fmt.Sprintf(
			"PATH: %s -> %d. Finished in %v.\n\tParams: %s\n\tResponse: %s",
			r.URL.Path,
			writer.StatusCode,
			done,
			string(body),
			writer.Response.String(),
		)

		msgs <- BrockerMessage{
			Msg: msg,
		}
	}
}
