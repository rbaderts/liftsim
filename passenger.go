package main

import (
	_ "fmt"

	"sync"
)

type PassengerStatus int
type PassengerId int

const (
	_ PassengerStatus = iota
	WaitingForPickup
	Moving
	Walkon
	Arrived
)

var PassengerStatuses = [...]string{
	"WaitingForPickup",
	"Moving",
	"Walkon",
	"Arrived",
}

func (b PassengerStatus) String() string {
	return PassengerStatuses[b-1]
}

type Passenger struct {
	Id         PassengerId
	liftId     string
	startFloor int
	direction  Direction
	destFloor  int
	status     PassengerStatus
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

func NewPassengerForPickup(floor int, dir Direction) *Passenger {

	p := new(Passenger)
	p.Id = nextPassengerId()
	p.startFloor = floor
	p.direction = dir
	p.status = WaitingForPickup

	return p
}

func NewPassengerWalkon(liftId string) *Passenger {

	p := new(Passenger)
	p.Id = nextPassengerId()
	p.status = Walkon

	return p
}

func (p *Passenger) Pickup(liftId string) {
	p.liftId = liftId
}
