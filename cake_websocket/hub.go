// Copyright 2013 The Gorilla WebSocket Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cake_websocket

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

// Hub maintains the set of active clients and broadcasts messages to the
// clients.
type Hub struct {
	// Registered clients.
	clients map[*Client]bool

	// Inbound messages from the clients.
	broadcast chan []byte

	// Register requests from the clients.
	register chan *Client

	// Unregister requests from clients.
	unregister chan *Client

	// Connections number metric
	numOfConnections prometheus.Gauge
}

func (h *Hub) SendMessage(msg string) {
	h.broadcast <- []byte(msg)
}

func (h *Hub) SendMessages(d time.Duration) {
	for {
		h.broadcast <- []byte("Hello world!")
		time.Sleep(d)
	}
}

func NewHub(metrics prometheus.Gauge) *Hub {
	return &Hub{
		broadcast:        make(chan []byte),
		register:         make(chan *Client),
		unregister:       make(chan *Client),
		clients:          make(map[*Client]bool),
		numOfConnections: metrics,
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.clients[client] = true
			h.numOfConnections.Inc()
		case client := <-h.unregister:
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
				h.numOfConnections.Dec()
			}
		case message := <-h.broadcast:
			for client := range h.clients {
				select {
				case client.send <- message:
				default:
					close(client.send)
					delete(h.clients, client)
				}
			}
		}
	}
}
