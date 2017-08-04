package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	glog "github.com/ccding/go-logging/logging"
	"github.com/gorilla/websocket"
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

var LiftSystems map[string]*LiftSystemT

//var LiftSystem *LiftSystemT

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
	LiftSystems = make(map[string]*LiftSystemT)
}

func GetLiftSystem(id string) *LiftSystemT {
	return LiftSystems[id]
}

func NewLiftSystem(id string) *LiftSystemT {

	liftCtl := newLiftSystem()
	LiftSystems[id] = liftCtl
	go liftCtl.run()
	return liftCtl
}

/*
 * CommandType
 */

type CommandType int

const (
	_ CommandType = iota
	PickupRequest
	DropoffRequest
	Tick
	FloorArrival
	CalledHome
)

var CommandTypes = [...]string{
	"None",
	"PickupRequest",
	"DropoffRequest",
	"Tick",
	"FloorArrival",
	"CalledHome",
}

func (b CommandType) String() string {
	return CommandTypes[b]
}

type Command struct {
	Typ       CommandType `json:"typ"`
	LiftId    string      `json:"liftId"`
	Floor     int         `json:"floor"`
	Direction Direction   `json:"direction"`
	Passenger *Passenger  `json:"passenger"`
}

func (e Command) String() string {

	var buf bytes.Buffer

	fmt.Fprintf(&buf, "event - lift: %v, pid: %v, typ: %v, fl: %v, dir = %v", e.LiftId, e.Passenger, e.Typ, e.Floor, e.Direction)
	return buf.String()

}

const MAX_FLOORS = 50

/*
 * ButtonType
 */

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

/*
 * Direction
 */

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

/*
 * Stop
 */

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
	//    s.Passenger, err = strconv.Atoi(fields["passengerId"])

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

/*
 * ListStatus
 */

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

/*
 * LiftSystem
 */

type LiftSystemState struct {
	Cycle      int              `json:"cycle"`
	Lifts      map[string]*Lift `json:"lifts"`
	Speed      int              `json:"speed"`
	EventSpeed int              `json:"eventSpeed"`
	Paused     bool             `json:"paused"`
}

type LiftSystemT struct {
	State       LiftSystemState
	LiftUpdates chan Lift
	//EventLog    []Event
	commandQueue chan *Command
	tickQueue    chan *Command
	stateChanges chan *LiftEvent
	mutex        *sync.Mutex
	StateLog     *tlog.TLog
	Clients      map[*websocket.Conn]bool
}

func (this LiftSystemT) String() string {
	var buf bytes.Buffer
	fmt.Fprintf(&buf, "cycle %d\n", this.State.Cycle)
	for i, l := range this.State.Lifts {
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
	for _, lift := range liftCtl.State.Lifts {
		time := lift.estimateCostToPickup(floor, dir, false)

		if time >= 0 && time < quickestTime {
			quickestTime = time
			quickestLift = lift
		}
	}

	if quickestLift == nil {

		for _, lift := range liftCtl.State.Lifts {
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
	quickestLift.addStopWithDirection(floor, Pickup, dir, passenger)
}

func (liftCtl *LiftSystemT) GetLift(liftId string) *Lift {
	return liftCtl.State.Lifts[liftId]
}

func (liftCtl *LiftSystemT) getLiftStatesJson() string {

	jsonBytes, err := json.Marshal(liftCtl.State.Lifts)
	if err != nil {
		panic("getLiftsStatesJson")
	}
	return string(jsonBytes)
}

func newLiftSystem() *LiftSystemT {
	liftSystem := new(LiftSystemT)
	liftSystem.State.Cycle = 1
	liftSystem.State.Speed = 1
	liftSystem.State.EventSpeed = 1
	liftSystem.State.Paused = false
	liftSystem.LiftUpdates = make(chan Lift, 2)
	liftSystem.commandQueue = make(chan *Command, 2)
	liftSystem.tickQueue = make(chan *Command, 1)
	liftSystem.stateChanges = make(chan *LiftEvent, 10)
	liftSystem.mutex = new(sync.Mutex)

	liftSystem.State.Lifts = make(map[string]*Lift)
	liftSystem.Clients = make(map[*websocket.Conn]bool) // connected clients

	err := os.Mkdir("logs", os.FileMode(0755))
	if err != nil && os.IsNotExist(err) {
		panic(err)
	}

	statelog, err := tlog.NewTLog("logs/statelog")
	if err != nil {
		panic(err)
	}
	liftSystem.StateLog = statelog

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
		lift := liftSystem.NewLift(id)
		Logger.Debugf("NewLift id = %v = %v", id, lift)
		liftSystem.State.Lifts[id] = lift
	}

	return liftSystem
}

//func GetLif(id string) *Lift {
///    return LiftSystem.state.Lifts[id]
//}

func (liftCtl *LiftSystemT) RecordState() {
	jsonbytes, err := json.Marshal(liftCtl.State)
	if err != nil {
		fmt.Printf("state before panic: %v\n", liftCtl)
		panic(fmt.Sprintf("can't marshal: err = %v\n", err))
	}
	_, err = liftCtl.StateLog.LogEvent(jsonbytes)
	if err != nil {
		panic(err)
	}
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

func (liftCtl *LiftSystemT) forAllLifts(f func(lift *Lift)) {

	for _, lift := range liftCtl.State.Lifts {
		f(lift)
	}
}

func (liftCtl *LiftSystemT) QueueTick(cmd *Command) {
	liftCtl.tickQueue <- cmd
}

func (liftCtl *LiftSystemT) QueueCommand(cmd *Command) {
	liftCtl.commandQueue <- cmd
}

func (liftCtl *LiftSystemT) run() {

	for {
		select {
		case command := <-liftCtl.commandQueue:
			liftCtl.ProcessCommand(command)
			break

		case tick := <-liftCtl.tickQueue:
			liftCtl.ProcessCommand(tick)

		case event := <-liftCtl.stateChanges:
			_ = event

		default:

		}
	}

}

func (liftCtl *LiftSystemT) ProcessCommand(cmd *Command) {

	Logger.Debugf("Processing Command %v\n", cmd)

	switch cmd.Typ {

	case PickupRequest:
		liftCtl.pressCallButton(cmd.Floor, cmd.Direction, cmd.Passenger)
		break

	case Tick:

		var wg sync.WaitGroup

		for _, lift := range liftCtl.State.Lifts {
			wg.Add(1)
			go func(l *Lift) {
				defer wg.Done()
				l.Step()
			}(lift)
		}
		wg.Wait()

		command := ClientCommand{UpdateLiftSystem.String(), liftCtl.State}
		for client := range liftCtl.Clients {
			client.WriteJSON(command)
		}

		liftCtl.State.Cycle += 1
	}
	Logger.Debugf("Done Processing Command %v\n", cmd)

}

/*
 * At any point in time a lifts overall state is defined by the following structure
 *
 * A Lift has an initial state when it is created, each step in the simulation can be
 *   defined by a structure containing the changes in the state (LiftEvent) of the lift
 *   that ccured during that step (eg.  Direction changed to "Down",
 *   Floor increased by 1, Stop for floor 7 removed)
 *
 */

/*
 * LiftEvent
 */

type LiftEventType int

const (
	_ LiftEventType = iota
	SetFloor
	OpenDoor
	CloseDoor
	PassengerBoarded
	PassengerDisembarked
	SetLiftStatus
	SetSpeed
	SetDirection
	AddStop
	ClearStop
)

var LiftEventTypes = [...]string{
	"None",
	"SetFloor",
	"OpenDoor",
	"CloseDoor",
	"PassengerBoarded",
	"PassengerDisembarked",
	"SetLiftStatus",
	"SetSpeed",
	"SetDirection",
	"AddStop",
	"ClearStop",
}

func (b LiftEventType) String() string {
	return LiftEventTypes[b]
}

type LiftEvent struct {
	liftId    string        `json:"liftId"`
	time      time.Time     `json:"time"`
	eventType LiftEventType `json:"event-type"`
	eventData string        `json:"event-data"`
}

func (s LiftEvent) MarshalJSON() ([]byte, error) {

	b, err := json.Marshal(map[string]interface{}{
		"liftId":    s,
		"time":      s.time.String(),
		"eventType": s.eventType.String(),
		"eventData": s.eventData,
	})

	if err != nil {
		panic("error marshall Stop\n")
	}
	return b, err
}

/*
 * Lift
 */

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
	liftCtl *LiftSystemT

	mutex *sync.Mutex `json:"-"`
}

func (liftCtl *LiftSystemT) NewLift(id string) *Lift {
	var lift *Lift = new(Lift)

	lift.liftCtl = liftCtl
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

func (lift *Lift) emitStateChangeEvent(eType LiftEventType, val string) {

	//ev := &LiftEvent{lift.LiftId, time.Now(), eType, val}
	//	LiftSystem.stateChanges <- ev

}

func (lift *Lift) getMutex() *sync.Mutex {
	return lift.mutex
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

func (lift *Lift) setStatus(st LiftStatus) {
	lift.Status = st
	lift.emitStateChangeEvent(SetLiftStatus, st.String())
}

func (lift *Lift) Step() {

	Logger.Debugf("Step for lift:%v\n", lift.LiftId)
	Logger.Debugf("   lift; %v\n", lift)

	lift.getMutex().Lock()

	switch lift.Status {

	case Idle:

		dest := lift.getNextStopInDirection(NoDirection)
		if dest == 0 {
			lift.setStatus(Idle)
		} else if lift.Floor > dest {
			lift.setStatus(MovingDown)
			lift.Direction = Down
			lift.Status = MovingDown
		} else if lift.Floor < dest {
			lift.Direction = Up
			lift.setStatus(MovingUp)
			lift.emitStateChangeEvent(SetDirection, Up.String())
		} else if lift.Floor == dest {
			lift.setStatus(DoorOpening)
		}

	case MovingUp:

		floorStr := strconv.Itoa(lift.Floor)
		if lift.Stops[floorStr] != nil {
			lift.setStatus(DoorOpening)
		} else if lift.Floor == 50 {
			lift.setStatus(MovingDown)
		} else {
			lift.Floor += 1
			lift.floorsTraveled += 1
			lift.emitStateChangeEvent(SetFloor, strconv.Itoa(lift.Floor))
		}
		break

	case MovingDown:

		floorStr := strconv.Itoa(lift.Floor)
		if lift.Stops[floorStr] != nil {
			lift.setStatus(DoorOpening)
		} else if lift.Floor == 1 {
			lift.setStatus(DoorOpening)
		} else {
			lift.Floor -= 1
			lift.floorsTraveled += 1
			lift.emitStateChangeEvent(SetFloor, strconv.Itoa(lift.Floor))
		}
		break

	case DoorClosing:
		dest := lift.getNextStopInDirection(lift.Direction)
		lift.emitStateChangeEvent(CloseDoor, strconv.Itoa(lift.Floor))
		if dest > 0 && lift.Floor > dest {
			lift.Direction = Down
			lift.setStatus(MovingDown)
			lift.emitStateChangeEvent(SetDirection, Down.String())
		} else if dest > 0 {
			lift.Direction = Up
			lift.setStatus(MovingUp)
			lift.emitStateChangeEvent(SetDirection, Up.String())
		} else if dest == 0 {
			lift.Direction = NoDirection
			lift.setStatus(Idle)
			lift.emitStateChangeEvent(SetDirection, NoDirection.String())
		}
		break

	case DoorOpening:
		floorStr := strconv.Itoa(lift.Floor)
		if lift.Stops[floorStr] != nil {
			for _, s := range lift.Stops[floorStr] {
				if s.StopType == Dropoff {
					lift.Occupants -= 1
					s.Passenger.Status = Arrived
					lift.logRide(s.Passenger, lift.floorsTraveled-s.Passenger.StartFloorCount, lift.liftCtl.State.Cycle)
					delete(lift.Passengers, s.Passenger.Id)
					lift.emitStateChangeEvent(PassengerDisembarked, s.Passenger.GetIdAsString())

				} else if s.StopType == Pickup {
					lift.Occupants += 1
					if s.Direction == Up {
						f := lift.Floor + random.Intn(50-lift.Floor) + 1
						lift.addStop(f, Dropoff, s.Passenger)
					} else if s.Direction == Down {
						lift.addStop(1, Dropoff, s.Passenger)
					}
					//                    if s.Passenger != nil {
					s.Passenger.Status = Moving
					s.Passenger.StartFloorCount = lift.floorsTraveled
					lift.Passengers[s.Passenger.Id] = s.Passenger
					//                    }
					lift.emitStateChangeEvent(PassengerBoarded, s.Passenger.GetIdAsString())
				}
			}
			lift.clearFloor(lift.Floor)
			lift.setStatus(DoorClosing)
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

func contains(s []int, e int) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}
