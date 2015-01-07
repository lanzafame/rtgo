package rtgo

import (
	"encoding/json"
	"log"
)

type RTRoom struct {
	name    string
	members map[*RTConn]bool
	stop    chan bool
	join    chan *RTConn
	leave   chan *RTConn
	send    chan []byte
}

// RoomManager manages, or holds, all existing rooms.
var RoomManager = make(map[string]*RTRoom)

// Start activates the room.
func (r *RTRoom) Start() {
	for {
		select {
		case c := <-r.join:
			payload := &Message{
				Room:    r.name,
				Event:   "join",
				Payload: c.id,
			}
			data, err := json.Marshal(payload)
			if err != nil {
				log.Println(err)
				break
			}
			c.send <- data
			r.members[c] = true
		case c := <-r.leave:
			if _, ok := r.members[c]; ok {
				payload := &Message{
					Room:    r.name,
					Event:   "leave",
					Payload: c.id,
				}
				data, err := json.Marshal(payload)
				if err != nil {
					log.Println(err)
					break
				}
				c.send <- data
				delete(r.members, c)
			}
		case data := <-r.send:
			for c := range r.members {
				select {
				case c.send <- data:
				default:
					close(c.send)
					delete(r.members, c)
				}
			}
		case <-r.stop:
			return
		}
	}
}

// Stop deactivates the room.
func (r *RTRoom) Stop() {
	r.stop <- true
}

// Join will add a connection to the room.
func (r *RTRoom) Join(c *RTConn) {
	r.join <- c
}

// Leave will remove a connection from a room.
func (r *RTRoom) Leave(c *RTConn) {
	r.leave <- c
}

// Emit will send a message to all connections in the room.
func (r *RTRoom) Emit(payload *Message) {
	data, err := json.Marshal(payload)
	if err != nil {
		log.Println(err)
		return
	}
	r.send <- data
}

// NewRoom will create a new room with the specified name,
// start it, and add it to RoomManager.
// It returns the new room.
func NewRoom(name string) *RTRoom {
	r := &RTRoom{
		name:    name,
		members: make(map[*RTConn]bool),
		stop:    make(chan bool),
		join:    make(chan *RTConn),
		leave:   make(chan *RTConn),
		send:    make(chan []byte, 256),
	}
	RoomManager[name] = r
	go r.Start()
	return r
}
