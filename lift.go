package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	glog "github.com/ccding/go-logging/logging"
	_ "log"
	"math"
	"math/rand"
	"strconv"
	"sync"
	"time"
)

var random *rand.Rand
var Logger *glog.Logger

var Lifts map[string]*Lift
var LiftSystem *LiftSystemT

func init() {

	var err error
	Logger, err =
		glog.FileLogger("logfile",
			glog.DEBUG,
			glog.BasicFormat,
			glog.DefaultTimeFormat,
			"logfile", true)

	if err != nil {
		fmt.Printf("error opening file: %v", err)
	}
	fmt.Printf("Beginning logging")

	random = rand.New(rand.NewSource(time.Now().UnixNano()))
}

func GetLiftSystem() *LiftSystemT {
	return LiftSystem
}

func InitLiftsSystem() {

	Lifts = make(map[string]*Lift)

	for i := 1; i <= 4; i++ {
		id := "lift" + strconv.Itoa(i)
		lift := NewLift(id)
		Logger.Debugf("NewLift id = %v = %v", id, lift)
		Lifts[id] = lift
	}
	liftCtl := NewLiftSystem(Lifts)
	go liftCtl.run()

}

func (liftCtl *LiftSystemT) getMutex() *sync.Mutex {
	return liftCtl.mutex
}

func GetLift(id string) *Lift {
	return Lifts[id]
}

type FloorStops []*Stop

//type PendingStops map[string][]*Stop
//type PendingStops [MAX_FLOORS]FloorStops;
type PendingStops struct {
	Floors  [MAX_FLOORS]FloorStops
	highest int
	lowest  int
}

func newPendingStops() *PendingStops {
	p := new(PendingStops)
	return p
}

/*
func (stops *PendingStops) MarshalJSON() ([]byte, error) {
	b, err := json.Marshal(map[string]interface{}{
		"floors":  stops.floors,
		"highest": stops.lowest,
		"lowest":  stops.highest,
	})
	fmt.Printf("pendingstopsjson = %v\n", string(b))
	return b, err
}
*/

func (stops *PendingStops) clearFloor(floor int) {

	stops.Floors[floor-1] = nil

	if floor == stops.highest {
		var i int
		for i = floor - 1; i >= 1; i-- {
			if stops.Floors[i] != nil {
				break
			}
		}
		stops.highest = i
	}
	if floor == stops.lowest {
		var i int
		for i = floor + 1; i < MAX_FLOORS; i++ {
			if stops.Floors[i] != nil {
				break
			}
		}
		stops.lowest = i
	}
}

func (stops *PendingStops) queueStop(floor int, stop Stop) {

	if stops.Floors[floor-1] == nil {
		stops.Floors[floor-1] = make(FloorStops, 0)
	}
	stops.Floors[floor-1] = append(stops.Floors[floor-1], &stop)
	if floor > stops.highest {
		stops.highest = floor
	}
	if floor < stops.lowest {
		stops.lowest = floor
	}

}

func (stops PendingStops) String() string {

	var buf bytes.Buffer
	for k, v := range stops.Floors {
		if v != nil {
			fmt.Fprintf(&buf, "floor %v(", k+1)
			for _, s := range v {
				fmt.Fprintf(&buf, "%v, ", s.String())
			}
			fmt.Fprintf(&buf, ")")
		}
	}
	fmt.Fprintf(&buf, "\n")
	return buf.String()
}

/*
func (r *PendingStops) highest() int {
	var highest int = -1
	if len(*r) > 0 {
		for k, _ := range *r {
			kint, err := strconv.Atoi(k)
			if err != nil {
				panic(err)
			}
			if kint > highest {
				highest = kint
	000000		}
		}
	} else {
		return -1
	}
	return highest
}

func (r *PendingStops) lowest() int {
	var lowest int = 100
	if len(*r) > 0 {
		for k, _ := range *r {
			kint, err := strconv.Atoi(k)
			if err != nil {
				panic(err)
			}
			if kint < lowest {
				lowest = kint
			}
		}
	} else {
		return -1
	}
	return lowest
}
*/

func (r *PendingStops) NextUp(from int) int {
	var nearest int = 100
	var result int = -1
	for k, _ := range r.Floors {
		if k+1 > from && k+1 < nearest {
			result = k + 1
		}
	}
	return result
}

func (r *PendingStops) NextDown(from int) int {
	var nearest int = 0
	var result int = -1
	for k, _ := range r.Floors {
		if k+1 < from && k+1 > nearest {
			result = k + 1
		}
	}
	return result
}

func (r *PendingStops) Contains(floor int) bool {
	//	floorStr := strconv.Itoa(floor)
	if r.Floors[floor-1] == nil {
		return false
	}
	//_, ok := *r.floors[floor]
	//return ok
	return true
}

type EventType int

const (
	_ EventType = iota
	None
	PickupRequest
	DropoffRequest
	Tick
	FloorArrival
	CalledHome
)

var EventTypes = [...]string{
	"None",
	"PickupRequest",
	"DropoffRequest",
	"Tick",
	"FloorArrival",
	"CalledHome",
}

func (b EventType) String() string {
	return EventTypes[b-1]
}

type Event struct {
	Typ         EventType   `json:"typ"`
	LiftId      string      `json:"liftId"`
	Floor       int         `json:"floor"`
	Direction   Direction   `json:"direction"`
	PassengerId PassengerId `json:"passengerId"`
}

func (e Event) String() string {

	var buf bytes.Buffer
	fmt.Fprintf(&buf, "event - lift: %v, pid: %v, typ: %v, fl: %v, dir = %v", e.LiftId, e.PassengerId, e.Typ, e.Floor, e.Direction)
	return buf.String()

}

/*
func (b *Event) MarshalJSON() ([]byte, error) {
	liftId := b.LiftId
	return json.Marshal(map[string]interface{}{
		"typ":         b.Typ.String(),
		"lift":        liftId,
		"floor":       b.Floor,
		"direction":   b.Direction,
		"passengerId": b.PassengerId,
	})
}
*/

const MAX_FLOORS = 50

type ButtonType int

const (
	// 0-50 are reserved for Floor request buttons on the Lifts

	_ ButtonType = iota

	GotoFloorButton
	OpenDoorButton
	CloseDoorButton
	PickupCallDownButton
	PickupCallUpButton
)

var ButtonTypes = [...]string{
	"GotoFloorButton",
	"OpenDoorButton",
	"CloseDoorButton",
	"PickupCallDownButton",
	"PickupCallUpButton",
}

func (b ButtonType) String() string {
	return ButtonTypes[b-1]
}

type Direction int

const (
	_ Direction = iota
	NoDirection
	Up
	Down
)

var Directions = [...]string{
	"NoDirection",
	"Up",
	"Down",
}

func (b Direction) opposite() Direction {
	if b == Up {
		return Down
	} else if b == Down {
		return Up
	}
	return NoDirection

}

func (b Direction) String() string {
	return Directions[b-1]
}

type Stop struct {
	StopType    StopType    `json:"stopType"`
	Dir         Direction   `json:"direction"`
	PassengerId PassengerId `json:passengerId"`
}

/*
func (s *Stop) MarshalJSON() ([]byte, error) {
	b, err := json.Marshal(map[string]interface{}{
		"StopType":    s.StopType.String(),
		"Dir":         s.Dir.String(),
		"PassengerId": s.PassengerId,
	})
	return b, err
}
*/
func (s Stop) String() string {
	var buf bytes.Buffer
	fmt.Fprintf(&buf, "%v(%v) p:%v", s.StopType, s.Dir, s.PassengerId)
	return buf.String()
}

type StopType int

const (
	_ StopType = iota
	Zero
	Pickup
	Dropoff
)

var StopTypes = [...]string{
	"Zero",
	"Pickup",
	"Dropoff",
}

func (s StopType) String() string {
	return StopTypes[s-1]
}

type LiftStatus int

const (
	_ LiftStatus = iota
	Idle
	Returning
	OnRoute
	Loading
	Unloading
)

var LiftStatusStrings = [...]string{
	"Idle",
	"Returning",
	"OnRoute",
	"Loading",
	"Unloading",
}

func (b LiftStatus) String() string {
	return LiftStatusStrings[b-1]
}

type LiftSystemT struct {
	Lifts map[string]*Lift
	//pickupRequests []*Event
	Events      chan Event
	LiftUpdates chan Lift
	EventLog    []Event
	eventQueue  chan *Event
	mutex       *sync.Mutex
}

func NewLiftSystem(lifts map[string]*Lift) *LiftSystemT {
	LiftSystem = new(LiftSystemT)
	LiftSystem.Lifts = lifts
	LiftSystem.Events = make(chan Event, 10)
	LiftSystem.LiftUpdates = make(chan Lift, 20)
	LiftSystem.eventQueue = make(chan *Event, 10)
	LiftSystem.mutex = new(sync.Mutex)

	return LiftSystem
}

func (liftCtl *LiftSystemT) QueueEvent(event *Event) {
	Logger.Debugf("Queueing Event: %v\n", event)
	liftCtl.eventQueue <- event
	//	Logger.Debugf("Done Queueing Event: %v\n", event)
}
func (liftCtl *LiftSystemT) calculateLiftForPickup(request *Event) *Lift {

	var currentBest *Lift
	leastTime := math.MaxInt32

	for _, lift := range liftCtl.Lifts {
		d := lift.EstimateTimeToPickup(request)

		if d < leastTime {
			leastTime = d
			currentBest = lift
		}
	}

	Logger.Debugf("send for pickup:  lift: %v fl: %v dir: %v dest-floor: %v\n",
		currentBest.LiftId, currentBest.Floor, currentBest.Direction, request.Floor)

	return currentBest

}

func (liftCtl *LiftSystemT) forAllLifts(f func(lift *Lift)) {

	for _, lift := range Lifts {
		f(lift)
	}
}

func (liftCtl *LiftSystemT) recordEvent(event *Event) {

	liftCtl.EventLog = append(liftCtl.EventLog, *event)
	if event.Typ != Tick {
		Logger.Debugf("Sending event on event channel - %v\n", *event)
	}
	liftCtl.Events <- *event

}

func (liftCtl *LiftSystemT) run() {

	for {
		select {
		case event := <-liftCtl.eventQueue:
			liftCtl.ProcessEvent(event)
		}
	}

}

func (liftCtl *LiftSystemT) ProcessEvent(event *Event) {

	Logger.Debugf("Processing Event %v\n", event)

	switch event.Typ {

	case DropoffRequest:
		lift := GetLift(event.LiftId)
		Logger.Debugf("DropoffRequest:  lift: %v, fl: %v,  dir: %v,  dest-floor: %v", event.LiftId, lift.Floor, lift.Direction, event.Floor)
		lift.getMutex().Lock()
		lift.queueFloorStop(event)
		lift.getMutex().Unlock()
		lift.addPassenger(event.PassengerId)

	case PickupRequest:
		lift := LiftSystem.calculateLiftForPickup(event)
		Logger.Debugf("PickupRequest:  lift: %v, fl: %v,  dir: %v,  dest-floor: %v", event.LiftId, lift.Floor, lift.Direction, event.Floor)
		event.LiftId = lift.LiftId
		lift.getMutex().Lock()
		lift.queueFloorStop(event)
		lift.getMutex().Unlock()

	case CalledHome:

	case Tick:
		for _, l := range Lifts {
			//	go l.Tick()
			l.Tick()
		}

	}

	Logger.Debugf("Done Processing Event %v\n", event)
	//LiftSystem.recordEvent(event)

}

// Lift

type Lift struct {
	LiftId    string       `json:"liftId"`
	Direction Direction    `json:"direction"`
	Status    LiftStatus   `json:"status"`
	Floor     int          `json:"floor"`
	Stops     PendingStops `json:"stops"`
	Speed     int          `json:"speed"`

	IdleTicks     int         `json:"-"`
	LoadIdleTicks int         `json:"-"`
	mutex         *sync.Mutex `json:"-"`

	nextStop int `json:"-"`

	requestChannel chan int `json:"-"`
	controlChannel chan int `json:"-"`
	updateChannel  chan int `json:"-"`

	floorsTraveled int `json:"floorsTraveled"`
	stopsMade      int `json:"stopsMade"`
	TotalRiders    int `json:"totalRiders"`

	Passengers map[PassengerId]bool `json:"-"`
}

func NewLift(id string) *Lift {
	var lift *Lift = new(Lift)
	lift.LiftId = id
	lift.Floor = 1
	lift.Direction = NoDirection
	lift.Status = Idle
	lift.mutex = new(sync.Mutex)

	lift.requestChannel = make(chan int, 10)
	lift.controlChannel = make(chan int, 2)

	lift.Passengers = make(map[PassengerId]bool)
	lift.Speed = 1
	return lift
}

func (lift *Lift) getStops() []int {
	stops := make([]int, 0)

	for k, s := range lift.Stops.Floors {
		if s != nil {
			stops = append(stops, k+1)
			continue
		}
	}
	return stops
}

func (lift *Lift) getMutex() *sync.Mutex {
	return lift.mutex
}

func (lift *Lift) addPassenger(pid PassengerId) {
	lift.Passengers[pid] = true

}

func (lift *Lift) removePassenger(pid PassengerId) {
	delete(lift.Passengers, pid)
}

func contains(s []int, e int) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}

func (lift *Lift) clearFloorStops(floor int) {

	if lift.Stops.Contains(floor) == true {

		for _, v := range lift.Stops.Floors[floor-1] {
			if v.StopType == Pickup {
				lift.addPassenger(v.PassengerId)
			} else if v.StopType == Dropoff {
				lift.removePassenger(v.PassengerId)
			}
		}
		lift.Stops.clearFloor(floor)
		//delete(lift.pendingStops, floorStr)
	}

}

//func (lift *Lift) ResetStats(lift *Lift) {
func ResetStats(lift *Lift) {
	Logger.Debugf("Reseting stats for lift: %v\n", lift.LiftId)
	lift.TotalRiders = 0
	lift.stopsMade = 0
	lift.floorsTraveled = 0

}

func (lift *Lift) queueFloorStop(e *Event) {

	floor := e.Floor
	Logger.Debugf("queuing stop: lift: %v, nextstop: %v, fl: %v, dir:%v, dest-fl: %v, typ: %v\n", lift.LiftId, lift.nextStop, lift.Floor, lift.Direction, floor, e.Typ)

	s := new(Stop)
	s.PassengerId = e.PassengerId
	if e.Typ == DropoffRequest {
		s.StopType = Dropoff
		s.Dir = NoDirection
		lift.TotalRiders++
	} else if e.Typ == PickupRequest {
		s.StopType = Pickup
		s.Dir = e.Direction
	}

	lift.Stops.queueStop(floor, *s)

	if lift.Status == Idle || lift.Status == Returning {
		if lift.Floor > e.Floor {
			lift.Direction = Down
		} else {
			lift.Direction = Up
		}
		lift.Status = OnRoute
	}
}

// Just uses raw distance in floors for now
func (lift *Lift) EstimateTimeToPickup(e *Event) int {

	floor := e.Floor

	var dist int = 0

	if lift.Direction == e.Direction {

		if (e.Direction == Up && floor > lift.Floor+1) ||
			(e.Direction == Down && floor < lift.Floor-1) ||
			(lift.Status == Idle) || (lift.Direction == NoDirection) {

			dist = abs_int(floor - lift.Floor)

		}
		Logger.Debugf("est dest (%v) lift: %v fl: %v dir: %v dest-fl %v, dest-dir %v", dist, lift.LiftId, lift.Floor, lift.Direction,
			floor, e.Direction)

	} else {

		if lift.Direction == Up {
			highest := lift.Stops.highest
			leg1 := 0
			leg2 := 0
			if highest == -1 {
				if floor > lift.Floor {
					leg2 = floor - lift.Floor
				} else {
					leg2 = lift.Floor - floor
				}
				dist += leg1 + leg2
			} else {
				leg1 = highest - lift.Floor
				if floor > highest {
					leg2 = floor - highest
				} else {
					leg2 = highest - floor
				}
				dist += leg1 + leg2
			}
			Logger.Debugf("est dist %v(%v+%v) for lift: %v fl: %v, dir: %v, dest-fl: %v, dest-dir: %v",
				dist, leg1, leg2, lift.LiftId, lift.Floor, lift.Direction, floor, e.Direction)

		} else if lift.Direction == Down {
			lowest := lift.Stops.lowest
			leg1 := 0
			leg2 := 0
			if lowest == -1 {
				if floor < lift.Floor {
					leg2 = lift.Floor - floor
				} else {
					leg2 = floor - lift.Floor
				}
				dist += leg1 + leg2
			} else {
				leg1 = lift.Floor - lowest
				if floor < lowest {
					leg2 = lowest - floor
				} else {
					leg2 = floor - lowest
				}
			}
			dist = leg1 + leg2
			Logger.Debugf("est dist %v(%v+%v) for lift: %v fl: %v, dir: %v, dest-fl: %v, dest-dir: %v",
				dist, leg1, leg2, lift.LiftId, lift.Floor, lift.Direction, floor, e.Direction)
		}
	}
	dist += 1 // for lag
	return dist

}

func (lift *Lift) nextStopInDirection(dir Direction) int {

	closest := 0
	if dir == NoDirection {
		cdir := lift.Direction
		if lift.Direction == NoDirection {
			cdir = Up
		}
		n := lift.nextStopInDirection(cdir)
		if n == 0 {
			n = lift.nextStopInDirection(cdir.opposite())
		}
		return n
	}

	for k, s := range lift.Stops.Floors {

		if s == nil {
			continue
		}

		if dir == Up && k+1 > lift.Floor {
			if closest == 0 || (k+1-lift.Floor < closest-lift.Floor) {
				closest = k + 1
			}
		} else if dir == Down && k+1 < lift.Floor {
			if closest == 0 || (lift.Floor-k+1 > lift.Floor-closest) {
				closest = k + 1
			}
		}

	}
	return closest

}

/**

lift state machine

State:  Direction, Passengers, Currently queued floor stops, DoorState(open, closed)

States:   Idle, OnRoute(moving) Loading, Unloading, Returning

Events which immact the state


*/

func (lift *Lift) accelerate() {
	if lift.Speed == 3 {
		return
	}
	lift.Speed += 1
}

/*
func (lift *Lift) accelerate() {

	ahead := 1
	if lift.speed >= 2 {
		ahead = 2
	}
	var checkahead int
	var stopahead bool = false

	for checkahead = 1; checkahead <= ahead; checkahead++ {
		floorToCheck := lift.Floor
		if lift.Direction == Up {
			floorToCheck += checkahead
		} else if lift.Direction == Down {
			floorToCheck -= checkahead
		}
		if lift.pendingStops.Contains(floorToCheck) {
			stopahead = true
			break
		} else if lift.Direction == Up && floorToCheck == 50 {
			stopahead = true
			break
		} else if lift.Direction == Down && floorToCheck == 1 {
			stopahead = true
			break
		}
	}

	if !stopahead && lift.speed == 1 {
		lift.speed = 2
	} else if !stopahead && lift.speed == 2 {
		lift.speed = 3
	} else if stopahead {
		lift.speed = 1
	}
}
*/

func (lift *Lift) Tick() {

	Logger.Debugf("Tick -  lift: %v, status: %v, speed: %v, riders: %v, floor: %v, dir: %v, stops: %v, ",
		lift.LiftId, lift.Status, lift.Speed, lift.TotalRiders, lift.Floor, lift.Direction, lift.Stops.String())

	lift.getMutex().Lock()

	switch lift.Status {

	case Idle:

		fl := lift.nextStopInDirection(NoDirection)

		if fl != 0 {
			if fl > lift.Floor {
				lift.Direction = Up
			} else if fl < lift.Floor {
				lift.Direction = Down
			}
			lift.Status = OnRoute
			break
		} else {
			lift.IdleTicks++
			if lift.IdleTicks > 10 && lift.Floor != 1 {
				lift.IdleTicks = 0
				e := Event{CalledHome, lift.LiftId, 1, Down, 0}
				go LiftSystem.QueueEvent(&e)
				lift.Status = Returning
				lift.Direction = Down
				lift.IdleTicks = 0
			}

		}
	case OnRoute:

		if lift.Direction == NoDirection {
			panic("Bad direction for being onroute")
		}

		if lift.Stops.Contains(lift.Floor) {
			for _, s := range lift.Stops.Floors[lift.Floor-1] {
				if s.StopType == Pickup {
					floor := 1
					if s.Dir == Up && lift.Stops.NextUp(lift.Floor) > 1 {
						floor = lift.Floor + random.Intn(50-lift.Floor) + 1
					}

					e := Event{DropoffRequest, lift.LiftId, floor, s.Dir, s.PassengerId}
					go LiftSystem.QueueEvent(&e)
					lift.Status = Loading
				} else if s.StopType == Dropoff {
					lift.Status = Unloading
				}
			}

		} else {

			if lift.Direction == Down {
				nextStop := lift.nextStopInDirection(Down)
				if nextStop > lift.Floor {
					panic(fmt.Sprintf("Heading down but next stop is up - lift:%v\n", lift.LiftId))
				}
				distance := lift.Floor - nextStop
				if distance <= lift.Speed {
					lift.Speed = 1
					lift.Floor = nextStop
					lift.floorsTraveled += distance
				} else {
					lift.Floor -= lift.Speed
					lift.floorsTraveled += lift.Speed
				}
				if lift.Floor-nextStop > lift.Speed {
					lift.accelerate()
				}
			} else if lift.Direction == Up {
				nextStop := lift.nextStopInDirection(Up)
				if nextStop < lift.Floor {
					panic(fmt.Sprintf("Heading up but next stop is down - lift:%v\n", lift.LiftId))
				}
				distance := nextStop - lift.Floor
				if distance <= lift.Speed {
					lift.Speed = 1
					lift.Floor = nextStop
					lift.floorsTraveled += distance
				} else {
					lift.Floor += lift.Speed
					lift.floorsTraveled += lift.Speed
				}
				if nextStop-lift.Floor > lift.Speed {
					lift.accelerate()
				}
			} else {
				panic(fmt.Sprintf("NoDirection but OnRoute - lift:%v\n", lift.LiftId))
			}

		}

	case Loading:
		fallthrough
	case Unloading:

		lift.LoadIdleTicks++
		if lift.LoadIdleTicks > 1 {
			lift.clearFloorStops(lift.Floor)
			lift.LoadIdleTicks = 0
			nextStop := lift.nextStopInDirection(NoDirection)
			lift.Speed = 1
			if nextStop == 0 {
				lift.Status = Idle
				lift.Direction = NoDirection
			} else if nextStop > lift.Floor {
				lift.Direction = Up
				lift.Status = OnRoute
			} else if nextStop < lift.Floor {
				lift.Direction = Down
				lift.Status = OnRoute
			}
		}

	case Returning:

		if lift.Floor <= 1 {
			lift.Floor = 1
			lift.Speed = 1
			lift.Status = Idle
		} else {
			lift.Floor -= lift.Speed
			if lift.Floor <= 1 {
				lift.Floor = 1
			}
			lift.accelerate()
		}

	}

	ws := WS()
	if ws != nil {
		Logger.Debugf("Tick - publishing updated lift to websocket: %v\n", lift.String())
		ws.EmitMessage(lift.String())
	}

	lift.getMutex().Unlock()

}

func (lift Lift) String() string {

	jsonBytes, _ := json.Marshal(lift)
	return string(jsonBytes)

}

func abs_int(v int) int {
	if v < 0 {
		return -v
	}
	return v
}
