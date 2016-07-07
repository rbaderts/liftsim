package main

import (
	"fmt"

	"bytes"
	"sync"
)

type PassengerStatus int
type PassengerId int

var PassengerRepo map[int]*Passenger

const (
	_ PassengerStatus = iota
	WaitingForPickup
	Moving
	Walkon
	Arrived
	Pergatory
)

var PassengerStatuses = [...]string{
	"None",
	"WaitingForPickup",
	"Moving",
	"Walkon",
	"Arrived",
	"Pergatory",
}

func (b PassengerStatus) String() string {
	return PassengerStatuses[b]
}

type Ride struct {
	passengerId    PassengerId `json:"passengerId"`
	liftId         string      `json:"liftId"`
	StartFloor     int         `json:"start-floor"`
	DestFloor      int         `json:"dest-floor"`
	FloorsTraveled int         `json:"floors-traveled"`
	CycleCompleted int         `json:"cycle-completed"`
}

// The # of floors above abs(startFloor-DestFloor)
func (this *Ride) ComputeExtraFloors() int {
	dist := this.StartFloor - this.DestFloor
	if dist < 0 {
		dist = -dist
	}

	return this.FloorsTraveled - dist

}

type Passenger struct {
	Id              PassengerId     `json:"id"`
	LiftId          string          `json:"liftId"`
	StartFloor      int             `json:"start-floor"`
	Direction       Direction       `json:"direction"`
	DestFloor       int             `json:"dest-floor"`
	Status          PassengerStatus `json:"status"`
	StartFloorCount int             `json:"start-floor-count"`
}

var nextId int
var mutex *sync.Mutex

func init() {
	nextId = 1
	mutex = new(sync.Mutex)
}
func nextPassengerId() PassengerId {

	v := nextId
	mutex.Lock()
	nextId++
	mutex.Unlock()

	return PassengerId(v)
}

func NewPassengerForPickup(floor int, destFloor int, dir Direction) *Passenger {

	p := new(Passenger)
	p.Id = nextPassengerId()
	p.StartFloor = floor
	p.DestFloor = destFloor
	p.Direction = dir
	p.Status = WaitingForPickup

	return p
}

func NewPassengerWalkon(liftId string) *Passenger {

	p := new(Passenger)
	p.Id = nextPassengerId()
	p.Status = Walkon

	return p
}

func (p *Passenger) Pickup(liftId string) {
	p.LiftId = liftId
}

func (p Passenger) String() string {
	var buf bytes.Buffer
	fmt.Fprintf(&buf, "     id=%v, destFloor=%v, startFloor=%v, status=%v\n", p.Id, p.DestFloor, p.StartFloor, p.Status)
	//fmt.Fprintf(&buf, "%v", p.Id)
	return buf.String()
}
