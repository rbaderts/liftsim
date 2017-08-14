package main

import (
	"encoding/json"
	"fmt"
	"github.com/gorilla/websocket"
	"strconv"
	"time"
)

const (
	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer.
	pongWait = 60 * time.Second

	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer.
	maxMessageSize = 512
)

type ClientCommandType int

const (
	_ ClientCommandType = iota
	UpdateLift
	UpdateLiftSystem
	SetSpeed
	SetEventFrequency
	ResetStatsCmd
	Pause
	Unpause
	NewSimulation
)

var ClientCommandTypes = [...]string{
	"None",
	"UpdateLift",
	"UpdateLiftSystem",
	"SetSpeed",
	"SetEventFrequency",
	"ResetStatsCmd",
	"Pause",
	"Unpause",
	"NewSimulation",
}

func (b ClientCommandType) String() string {
	return ClientCommandTypes[b]
}

type ClientCommand struct {
	Command string      `json:"command"`
	Data    interface{} `json:"data"`
}

func NewCommand(typ ClientCommandType, data interface{}) *ClientCommand {
	return &ClientCommand{typ.String(), data}
}

type Client struct {
	simulation *LiftSystem
	ws         *websocket.Conn
	simId      string
	send       chan []byte
}

func NewClient(s *LiftSystem, con *websocket.Conn, simId string) *Client {
	c := &Client{s, con, simId, make(chan []byte)}
	s.Clients[c] = true
	return c
}

func (c *Client) ping(ws *websocket.Conn, done chan struct{}) {
	fmt.Printf("ping\n")
	ticker := time.NewTicker(pingPeriod)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			if err := ws.WriteControl(websocket.PingMessage, []byte{}, time.Now().Add(writeWait)); err != nil {
				fmt.Println("ping:", err)
			}
		case <-done:
			return
		}
	}
}

func (c *Client) WriteJSON(data interface{}) {
	c.ws.WriteJSON(data)
}

func (c *Client) ProcessCommands() {
	defer func() {
		c.ws.Close()
	}()
	c.ws.SetReadLimit(maxMessageSize)
	c.ws.SetReadDeadline(time.Now().Add(pongWait))
	c.ws.SetPongHandler(func(string) error { c.ws.SetReadDeadline(time.Now().Add(pongWait)); return nil })
	for {
		mtype, messageBytes, err := c.ws.ReadMessage()
		fmt.Printf("processCommands type = %v\n", mtype)

		if err != nil {
			/*
				fmt.Printf("err = %v\n", err)
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway) {
					fmt.Printf("error: %v", err)
				}
			*/
			c.ws.Close()
			break
		}

		if mtype == websocket.TextMessage {
			var cmd ClientCommand
			_ = json.Unmarshal(messageBytes, &cmd)
			c.handleCommand(&cmd)

		} else if mtype == -1 {
			c.ws.Close()
			delete(c.simulation.Clients, c)
			break
		}

		//c.hub.broadcast <- message
	}

}

func (c *Client) handleCommand(cmd *ClientCommand) {
	if cmd.Command == "SetSpeed" {
		strdata, ok := cmd.Data.(string)
		if !ok {
			fmt.Printf("Error with cmd data\n")
			return
		}

		speed, _ := strconv.Atoi(strdata)
		c.simulation.State.Speed = speed
	} else if cmd.Command == "SetEventFrequency" {
		strdata, ok := cmd.Data.(string)
		if !ok {
			fmt.Printf("Error with cmd data\n")
			return
		}
		speed, _ := strconv.Atoi(strdata)
		c.simulation.State.EventSpeed = speed
	} else if cmd.Command == "ResetStatsCmd" {
	} else if cmd.Command == "Pause" {
		c.simulation.State.Paused = true
	} else if cmd.Command == "Unpause" {
		c.simulation.State.Paused = false
	} else if cmd.Command == "NewSimulation" {
		fmt.Printf("Newsimiulatiion")
		delete(c.simulation.Clients, c)
		newSimId := CreateSimulation()
		c.simulation = Simulations[newSimId]
		c.simId = newSimId
		c.simulation.Clients[c] = true
	} else {
		fmt.Printf("Unknown command type %s\n", cmd.Command)
	}

}

/*
func (c *Client) serveWs(simulation *Simulation, w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		fmt.Println(err)
		return
	}
	client := &Client{simulation: simulation, ws: conn, send: make(chan []byte, 256)}
	client.simulation.register <- client

	// Allow collection of memory referenced by the caller by doing all work in
	// new goroutines.
	go client.writePump()
	go client.readPump()
}
*/
