package logging

import (
	"api-service/api"
	"bufio"
	"bytes"
	"errors"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"time"
)

type LogWriter struct {
	http.ResponseWriter

	StatusCode int
	Response   bytes.Buffer
}

func (w *LogWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	h, ok := w.ResponseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, errors.New("hijack not supported")
	}
	return h.Hijack()
}

func (w *LogWriter) WriteHeader(status int) {
	w.ResponseWriter.WriteHeader(status)
	w.StatusCode = status
}

func (w *LogWriter) Write(p []byte) (int, error) {
	w.Response.Write(p)
	return w.ResponseWriter.Write(p)
}

func LogRequest(h http.HandlerFunc) http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		writer := &LogWriter{
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

		log.Printf(
			"PATH: %s -> %d. Finished in %v.\n\tParams: %s\n\tResponse: %s",
			r.URL.Path,
			writer.StatusCode,
			done,
			string(body),
			writer.Response.String(),
		)
	}
}
