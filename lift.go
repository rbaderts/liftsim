package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	glog "github.com/ccding/go-logging/logging"
	_ "log"
	"math/rand"
	//"os"
	"strconv"
	"sync"
	"time"
//	"tlog"
)

var random *rand.Rand
var Logger *glog.Logger

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

type ClientCommandType int

const (
	_ ClientCommandType = iota
	UpdateLift
	UpdateLiftSystem
)

var ClientCommandTypes = [...]string{
	"None",
	"UpdateLift",
	"UpdateLiftSystem",
}

func (b ClientCommandType) String() string {
	return ClientCommandTypes[b]
}

type ClientCommand struct {
	Command string      `json:"command"`
	Data    interface{} `json:"data"`
}

/*
func (b *CommandType) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]interface{}{
		"command":         command.String
		"lift":        liftId,
		"floor":       b.Floor,
		"direction":   b.Direction,
		"passengerId": b.PassengerId,?
		??

*/
func GetLiftSystem() *LiftSystemT {
	return LiftSystem
}

func InitLiftsSystem() {

	liftCtl := NewLiftSystem()
	go liftCtl.run()

}

func (this LiftSystemT) String() string {
	var buf bytes.Buffer
	fmt.Fprintf(&buf, "cycle %d\n", this.state.Cycle)
	for i, l := range this.state.Lifts {
		fmt.Fprintf(&buf, "lift(%v) : %v\n", i, l)
	}
	return buf.String()
}

func (liftCtl *LiftSystemT) getMutex() *sync.Mutex {
	return liftCtl.mutex
}

func (liftCtl *LiftSystemT) pressCallButton(floor int, dir Direction, passenger *Passenger) {

	quickestTime := 10000
	var quickestLift *Lift
	for _, lift := range liftCtl.state.Lifts {
		time := lift.estimateCostToPickup(floor, dir, false)

		if time >= 0 && time < quickestTime {
			quickestTime = time
			quickestLift = lift
		}
	}

	if quickestLift == nil {

		for _, lift := range liftCtl.state.Lifts {
			time := lift.estimateCostToPickup(floor, NoDirection, true)

			if time >= 0 && time < quickestTime {
				quickestTime = time
				quickestLift = lift
			}
		}
	}

	if quickestLift == nil {
		panic("no quickestList: \n")
	}
	quickestLift.getMutex().Lock()
	quickestLift.addStopWithDirection(floor, Pickup, dir, passenger)
	quickestLift.getMutex().Unlock()
}

func (liftCtl *LiftSystemT) GetLift(liftId string) *Lift {
	return liftCtl.state.Lifts[liftId]
}

type EventType int

const (
	_ EventType = iota
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
	return EventTypes[b]
}

type Event struct {
	Typ       EventType  `json:"typ"`
	LiftId    string     `json:"liftId"`
	Floor     int        `json:"floor"`
	Direction Direction  `json:"direction"`
	Passenger *Passenger `json:"passenger"`
}

func (e Event) String() string {

	var buf bytes.Buffer

	fmt.Fprintf(&buf, "event - lift: %v, pid: %v, typ: %v, fl: %v, dir = %v", e.LiftId, e.Passenger, e.Typ, e.Floor, e.Direction)
	return buf.String()

}

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
	"None",
	"GotoFloorButton",
	"OpenDoorButton",
	"CloseDoorButton",
	"PickupCallDownButton",
	"PickupCallUpButton",
}

func (b ButtonType) String() string {
	return ButtonTypes[b]
}

type Direction int

const (
	_ Direction = iota
	NoDirection
	Up
	Down
)

var Directions = [...]string{
	"None",
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

func (d Direction) String() string {
	return Directions[d]
}

func DirectionFromString(s string) Direction {
	var r Direction
	for i, t := range Directions {
		if t == s {
			r = Direction(i)
			break
		}
	}
	return r
}

type Command int

const (
	// 0-50 are reserved for Floor request buttons on the Lifts

	_ Command = iota

	Stop_Command
	GoHome_Command
	Go_Command
	Load_Command
	Continue_Command
)

type LiftCommand struct {
	commandType Command
	parameter   interface{}
}

type Stop struct {
	Floor     int       `json:"floor"`
	StopType  StopType  `json:"stopType"`
	Direction Direction `json:"direction"`
	Passenger *Passenger
}

func (s Stop) MarshalJSON() ([]byte, error) {

	b, err := json.Marshal(map[string]interface{}{
		"stopType":    s.StopType.String(),
		"direction":   s.Direction.String(),
		"floor":       s.Floor,
		"passengerId": s.Passenger.Id,
	})

	if err != nil {
		panic("error marshall Stop\n")
	}
	return b, err
}
func (s *Stop) UnmarshalJSON(data []byte) error {
	var fields map[string]string
	err := json.Unmarshal(data, &fields)
	if err != nil {
		return err
	}

	s.Floor, err = strconv.Atoi(fields["floor"])
	s.StopType = StopTypeFromString(fields["stopType"])
	s.Direction = DirectionFromString(fields["direction"])
	//	s.Passenger, err = strconv.Atoi(fields["passengerId"])

	return nil
}

func (s Stop) String() string {
	var buf bytes.Buffer
	fmt.Fprintf(&buf, "%v(%v:%v:passgner=%v)", s.Floor, s.StopType, s.Direction, s.Passenger.Id)
	return buf.String()
}

type StopType int

const (
	_ StopType = iota
	Pickup
	Dropoff
)

var StopTypes = [...]string{
	"None",
	"Pickup",
	"Dropoff",
}

func (s StopType) String() string {
	return StopTypes[s]
}

func StopTypeFromString(s string) StopType {
	var r StopType
	for i, t := range StopTypes {
		if t == s {
			r = StopType(i)
			break
		}
	}
	return r
}

type LiftStatus int

const (
	_ LiftStatus = iota
	Idle
	MovingDown
	MovingUp
	DoorOpening
	DoorClosing
)

var LiftStatuses = [...]string{
	"None",
	"Idle",
	"MovingDown",
	"MovingUp",
	"DoorOpening",
	"DoorClosing",
}

func (b LiftStatus) String() string {
	return LiftStatuses[b]
}

func LiftStatusFromString(s string) LiftStatus {
	var r LiftStatus
	for i, t := range LiftStatuses {
		if t == s {
			r = LiftStatus(i)
			break
		}
	}
	return r
}

type LiftSystemState struct {
	Cycle int              `json:"cycle"`
	Lifts map[string]*Lift `json:"lifts"`
}

type LiftSystemT struct {
	state       LiftSystemState
	Events      chan Event
	LiftUpdates chan Lift
	EventLog    []Event
	eventQueue  chan *Event
	mutex       *sync.Mutex
//	StateLog    *tlog.TLog
}

func (liftCtl *LiftSystemT) getLiftStatesJson() string {

	jsonBytes, err := json.Marshal(liftCtl.state.Lifts)
	if err != nil {
		panic("getLiftsStatesJson")
	}
	return string(jsonBytes)
}

func NewLiftSystem() *LiftSystemT {
	LiftSystem = new(LiftSystemT)
	LiftSystem.state.Cycle = 1
	LiftSystem.Events = make(chan Event, 10)
	LiftSystem.LiftUpdates = make(chan Lift, 20)
	LiftSystem.eventQueue = make(chan *Event, 10)
	LiftSystem.mutex = new(sync.Mutex)

	LiftSystem.state.Lifts = make(map[string]*Lift)

//	err := os.Mkdir("logs", os.FileMode(0755))

//	if err != nil {
//		panic(err)
//	}

//	statelog, err := tlog.NewTLog("logs/statelog")
//	if err != nil {
//		panic(err)
//	}
//	LiftSystem.StateLog = statelog

	for i := 1; i <= 4; i++ {
		id := "lift" + strconv.Itoa(i)
		lift := NewLift(id)
		Logger.Debugf("NewLift id = %v = %v", id, lift)
		LiftSystem.state.Lifts[id] = lift
	}

	return LiftSystem
}

func (liftCtl *LiftSystemT) RecordState() {
//	jsonbytes, err := json.Marshal(liftCtl.state)
//	if err != nil {
//		fmt.Printf("state before panic: %v\n", liftCtl)
//		panic(fmt.Sprintf("can't marshal: err = %v\n", err))
//	}
//	_, err = liftCtl.StateLog.LogEvent(jsonbytes)
//	if err != nil {
//		panic(err)
//	}
}

func (liftCtl *LiftSystemT) QueueEvent(event *Event) {
	Logger.Debugf("Queueing Event: %v\n", event)
	liftCtl.eventQueue <- event
}

func (liftCtl *LiftSystemT) forAllLifts(f func(lift *Lift)) {

	for _, lift := range liftCtl.state.Lifts {
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

	case PickupRequest:
		liftCtl.pressCallButton(event.Floor, event.Direction, event.Passenger)
		break

	case Tick:

		var wg sync.WaitGroup

		for _, lift := range liftCtl.state.Lifts {
			wg.Add(1)
			go func(l *Lift) {
				defer wg.Done()
				l.Step()
			}(lift)
		}
		wg.Wait()

		ws := WS()
		if ws != nil {

			c := ClientCommand{UpdateLiftSystem.String(), liftCtl.state}

			jsonbytes, err := json.Marshal(c)

			if err != nil {
				fmt.Printf("lift state before panic: %v\n", liftCtl)
				panic(fmt.Sprintf("cannot marshal to json - err=%v\n", err))
			}
			ws.EmitMessage([]byte(string(jsonbytes)))
		}

		liftCtl.RecordState()
		liftCtl.state.Cycle += 1

	}
	Logger.Debugf("Done Processing Event %v\n", event)

}

// per Lift StopQueue
// map indexed by floor, each entry contains a array of 0 or more Stop
//   A stop may be a Pickup or Dropoff

type LiftState struct {
	LiftId string `json:"liftId"`

	// state variables
	Direction Direction          `json:"direction"`
	Status    LiftStatus         `json:"status"` // Idle, Door Opening, Doors Closing,
	Floor     int                `json:"floor"`
	Speed     int                `json:"-"`
	Occupants int                `json:"occupants"`
	Stops     map[string][]*Stop `json:"stops"`

	Passengers map[PassengerId]*Passenger

	floorsTraveled int `json:"-"`

	totalRides       int `json:"total-rides"`
	totalExtraFloors int `json:"total-extra-floors"`

//	rideLog *tlog.TLog
	//Passengers map[PassengerId]bool `json:"-"`

}

type Lift struct {
	LiftState

	mutex *sync.Mutex `json:"-"`
}

func NewLift(id string) *Lift {
	var lift *Lift = new(Lift)

	lift.LiftId = id
	lift.Floor = 1
	lift.Direction = NoDirection
	lift.Status = Idle
	lift.Occupants = 0

	lift.mutex = new(sync.Mutex)
	lift.Stops = make(map[string][]*Stop)

	lift.Speed = 1

	lift.Passengers = make(map[PassengerId]*Passenger)

//	ridelog, err := tlog.NewTLog("logs/" + id + ".ridelog")
//	if err != nil {
//		panic(err)
//	}

//	lift.rideLog = ridelog

	return lift
}

func (lift *Lift) addStop(floor int, stopType StopType, passenger *Passenger) {
	lift.addStopWithDirection(floor, stopType, NoDirection, passenger)
}

func (lift *Lift) addStopWithDirection(floor int, stopType StopType, dir Direction, passenger *Passenger) {

	Logger.Debugf("addStop: %v to lift: %v\n", floor, lift.LiftId)
	floorStr := strconv.Itoa(floor)
	if lift.Stops[floorStr] == nil {
		lift.Stops[floorStr] = make([]*Stop, 0)
	}
	lift.Stops[floorStr] = append(lift.Stops[floorStr], &Stop{floor, stopType, dir, passenger})
}

func (lift *Lift) clearFloor(floor int) {
	floorStr := strconv.Itoa(floor)
	delete(lift.Stops, floorStr)
}

func (lift *Lift) getNextStop() int {

	next := lift.getNextStopInDirection(lift.Direction)

	if next == 0 {
		next = lift.getNextStopInDirection(NoDirection)
	}

	return next
}

func (lift *Lift) getNextStopInDirection(dir Direction) int {

	next := 0

	if len(lift.Stops) == 0 {
		return 0
	}

	if dir == Up {

		for f, _ := range lift.Stops {
			fl, _ := strconv.Atoi(f)
			if fl >= lift.Floor {
				if next == 0 {
					next = fl
				} else if next != 0 && fl < next {
					next = fl
				}
			}
		}
		return next

	} else if dir == Down {

		for f, _ := range lift.Stops {
			fl, _ := strconv.Atoi(f)
			if fl <= lift.Floor {
				if next == 0 {
					next = fl
				} else if next != 0 && fl > next {
					next = fl
				}
			}
		}
		return next

	} else if dir == NoDirection {

		closest := 0
		minDis := 50

		for f, _ := range lift.Stops {
			fl, _ := strconv.Atoi(f)
			dis := lift.Floor - fl
			if dis < 0 {
				dis = -dis
			}
			if dis < minDis {
				minDis = dis
				closest = fl
			}

		}
		return closest
	}

	return 0
}

func (lift *Lift) getMutex() *sync.Mutex {
	return lift.mutex
}

func contains(s []int, e int) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}

func ResetStats(lift *Lift) {
	Logger.Debugf("Reseting stats for lift: %v\n", lift.LiftId)
	lift.totalRides = 0
	lift.totalExtraFloors = 0
	lift.floorsTraveled = 0

}

// Cost includes wait time for the Pickup + any addional lift time imparted on current
//  occupants
//
// -1 is returned if lift is "too far" to pickup
//    In the event all Lifts return -1 for a given Pickup request a
//    more complex calculation will be made

func (lift *Lift) estimateCostToPickup(floor int, dir Direction, aggresive bool) int {

	var cost int = -1

	if lift.Status == Idle {
		cost = abs_int(floor - lift.Floor)
	} else if lift.Direction == Up {
		highest := lift.highestScheduledStop()
		if floor >= lift.Floor {
			if highest > floor {
				cost = floor - lift.Floor
			} else {
				addedFloors := floor - highest
				// dist to target + additional floors * # of occupants
				cost = (floor - lift.Floor) + (addedFloors * lift.Occupants)
			}
		} else {
			cost = (highest - lift.Floor) + (highest - floor)
		}
	} else if lift.Direction == Down {
		lowest := lift.lowestScheduledStop()
		if floor <= lift.Floor {
			if floor >= lowest {
				cost = lift.Floor - floor
			} else {
				addedFloors := lowest - floor
				cost = (lift.Floor - floor) + (addedFloors * lift.Occupants)
			}
		} else {
			cost = (lift.Floor - lowest) + (floor - lowest)
		}
	}
	Logger.Debugf("est for lift(%v) to floor (%v):  cost (%v)  current-floor(%v)\n", lift.LiftId, floor, cost, lift.Floor)
	return cost

}

func (lift *Lift) accelerate() {
	if lift.Speed == 3 {
		return
	}
	lift.Speed += 1
}

func (lift *Lift) Step() {

	Logger.Debugf("Step for lift:%v\n", lift.LiftId)
	Logger.Debugf("   lift; %v\n", lift)

	lift.getMutex().Lock()

	switch lift.Status {

	case Idle:

		dest := lift.getNextStopInDirection(NoDirection)
		if dest == 0 {
			lift.Status = Idle
		} else if lift.Floor > dest {
			lift.Direction = Down
			lift.Status = MovingDown
		} else if lift.Floor < dest {
			lift.Direction = Up
			lift.Status = MovingUp
		} else if lift.Floor == dest {
			lift.Status = DoorOpening
		}

	case MovingUp:

		floorStr := strconv.Itoa(lift.Floor)
		if lift.Stops[floorStr] != nil {
			lift.Status = DoorOpening
		} else if lift.Floor == 50 {
			lift.Status = MovingDown
		} else {
			lift.Floor += 1
			lift.floorsTraveled += 1
		}
		break

	case MovingDown:

		floorStr := strconv.Itoa(lift.Floor)
		if lift.Stops[floorStr] != nil {
			lift.Status = DoorOpening
		} else if lift.Floor == 1 {
			lift.Status = DoorOpening
		} else {
			lift.Floor -= 1
			lift.floorsTraveled += 1
		}
		break

	case DoorClosing:
		dest := lift.getNextStopInDirection(lift.Direction)
		if dest > 0 && lift.Floor > dest {
			lift.Direction = Down
			lift.Status = MovingDown
		} else if dest > 0 {
			lift.Direction = Up
			lift.Status = MovingUp
		} else if dest == 0 {
			lift.Direction = NoDirection
			lift.Status = Idle
		}
		break

	case DoorOpening:
		floorStr := strconv.Itoa(lift.Floor)
		if lift.Stops[floorStr] != nil {
			for _, s := range lift.Stops[floorStr] {
				if s.StopType == Dropoff {
					lift.Occupants -= 1
					s.Passenger.Status = Arrived
					lift.logRide(s.Passenger, lift.floorsTraveled-s.Passenger.StartFloorCount, GetLiftSystem().state.Cycle)
					delete(lift.Passengers, s.Passenger.Id)
				} else if s.StopType == Pickup {
					lift.Occupants += 1
					if s.Direction == Up {
						f := lift.Floor + random.Intn(50-lift.Floor) + 1
						lift.addStop(f, Dropoff, s.Passenger)
					} else if s.Direction == Down {
						lift.addStop(1, Dropoff, s.Passenger)
					}
					if s.Passenger != nil {
						s.Passenger.Status = Moving
						s.Passenger.StartFloorCount = lift.floorsTraveled
						lift.Passengers[s.Passenger.Id] = s.Passenger
					}
				}
			}
			lift.clearFloor(lift.Floor)
			lift.Status = DoorClosing
			break
		} else {
			lift.Status = Idle
		}

	}

	lift.getMutex().Unlock()
}

func (lift *Lift) logRide(p *Passenger, floorsTraveled int, cycle int) {
	r := &Ride{p.Id, p.LiftId, p.StartFloor, p.DestFloor, floorsTraveled, cycle}

	extraFloors := r.ComputeExtraFloors()
	lift.totalRides += 1
	lift.totalExtraFloors += extraFloors
	//jsonbytes, err := json.Marshal(r)
	//if err != nil {
//		panic(err)
//	}
//	lift.rideLog.LogEvent(jsonbytes)
}

func (lift Lift) String() string {

	var buf bytes.Buffer
	fmt.Fprintf(&buf, " id=%v, direction=%v, status=%v, floor=%v\n    passengers:%v\n    stops=%v\n",
		lift.LiftId, lift.Direction, lift.Status, lift.Floor, lift.Passengers, lift.Stops)

	return buf.String()
}

func (lift *Lift) highestScheduledStop() int {

	highest := -1
	for fl, _ := range lift.Stops {
		f, err := strconv.Atoi(fl)
		if err != nil {
			panic(err)
		}
		if f > highest {
			highest = f
		}
	}
	return highest
}

func (lift *Lift) lowestScheduledStop() int {

	lowest := 100
	for fl, _ := range lift.Stops {
		f, err := strconv.Atoi(fl)
		if err != nil {
			panic(err)
		}
		if f < lowest {
			lowest = f
		}
	}
	return lowest
}

func (lift Lift) MarshalJSON() ([]byte, error) {

	b, err := json.Marshal(map[string]interface{}{
		"liftId":           lift.LiftId,
		"direction":        lift.Direction.String(),
		"status":           lift.Status.String(),
		"floor":            strconv.Itoa(lift.Floor),
		"stops":            lift.Stops,
		"occupants":        lift.Occupants,
		"totalRides":       lift.totalRides,
		"totalExtraFloors": lift.totalExtraFloors,
	})
	if err != nil {
		panic(err)
	}
	return b, err

}

func (lift *Lift) UnmarshalJSON(data []byte) error {
	var fields map[string]interface{}
	err := json.Unmarshal(data, &fields)
	if err != nil {
		return err
	}
	lift.LiftId = fields["liftId"].(string)
	lift.Direction = DirectionFromString(fields["direction"].(string))
	lift.Status = LiftStatusFromString(fields["status"].(string))
	f, err := strconv.Atoi(fields["floor"].(string))
	if err != nil {
		panic(err)
	}
	lift.Floor = f

	stopsData := fields["stops"].(map[string]interface{})
	lift.Stops = make(map[string][]*Stop)

	for k, s := range stopsData {
		stops := s.([]interface{})
		lift.Stops[k] = make([]*Stop, len(stops), len(stops))
		for i, st := range stops {
			stopProps := st.(map[string]interface{})
			var s *Stop = new(Stop)
			s.Floor = int(stopProps["floor"].(float64))
			if fields["stopType"] != nil {
				s.StopType = StopTypeFromString(fields["stopType"].(string))
			}
			if fields["direction"] != nil {
				s.Direction = DirectionFromString(fields["direction"].(string))
			}
			lift.Stops[k][i] = s
		}
	}

	lift.totalRides, err = strconv.Atoi(fields["totalRides"].(string))
	if err != nil {
		panic(err)
	}
	lift.totalExtraFloors, err = strconv.Atoi(fields["totalExtraFloors"].(string))
	if err != nil {
		panic(err)
	}
	return nil
}

func abs_int(v int) int {
	if v < 0 {
		return -v
	}
	return v
}
