package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/kataras/iris"
	"github.com/kataras/iris/config"
	"github.com/kataras/iris/websocket"
	"math/rand"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"
	//"net/http"
	//	"text/template"
)

var addr = flag.String("addr", ":8080", "http service address")

var TickSpeedFactor int = 1
var Paused bool = false

var WSC websocket.Connection

func WS() websocket.Connection {
	return WSC
}

func main() {

	config := config.Iris{
		Profile:     true,
		ProfilePath: "",
	}
	_ = config

	//iris.Config().Render.Template.Engine = iris.HTMLTemplate

	flag.Parse()

	//InitBuilding()
	InitLiftsSystem()

	iris.Static("/css", "./resources/css", 1)
	iris.Static("/js", "./resources/js", 1)
	iris.Static("/img", "./resources/img", 1)

	iris.Get("/", func(ctx *iris.Context) {
		if err := ctx.Render("index.html", nil); err != nil {
			println(err.Error())
		}
	})

	iris.Config().Render.Template.Layout = "layouts/layout.html"
	iris.Config().Websocket.Endpoint = "/updates"

	ws := iris.Websocket()

	ws.OnConnection(func(c websocket.Connection) {
		WSC = c
		fmt.Printf("websocket connection: %v\n", c)
	})

	iris.Get("/api/fastforward", func(ctx *iris.Context) {
		entries, _, err := LiftSystem.StateLog.GetNextNEntries(1)
		if err != nil {
			panic(err)
		}

		var state *LiftSystemState = &LiftSystemState{}
		state.Lifts = make(map[string]*Lift)
		err = json.Unmarshal(entries[0], state)
		if err != nil {
			panic(err)
		}
		ctx.JSON(iris.StatusOK, state)
	})

	iris.Get("/api/rewind", func(ctx *iris.Context) {
		//liftId := ctx.Param("liftId")
		//cycle, err := ctx.URLParamInt("cycle")
		/*
			if err != nil {
				Logger.Debugf("error getting cycle: %v\n", err)
				cycle = 0
			}
		*/

		//	if cycle != 0 {
		entries, _, err := LiftSystem.StateLog.GetLastNEntries(1)
		if err != nil {
			panic(err)
		}

		var state *LiftSystemState = &LiftSystemState{}
		state.Lifts = make(map[string]*Lift)
		///		var animals *Animal = &Animal{}
		err = json.Unmarshal(entries[0], state)
		if err != nil {
			panic(err)
		}

		//lift := LiftSystem.GetOldLift(liftId, int64(cycle))
		//Logger.Debugf("get old lift %v\n", lift)
		ctx.JSON(iris.StatusOK, state)
		/*
			else {
				lift := LiftSystem.GetLift(liftId)
				Logger.Debugf("get lift %v\n", lift)
				ctx.JSON(iris.StatusOK, lift)
			}
		*/

	})

	iris.Post("/api/setspeed", func(ctx *iris.Context) {
		speed := ctx.PostFormValue("speed")
		Logger.Debugf("Set speed to %v\n", speed)
		s, _ := strconv.Atoi(speed)
		TickSpeedFactor = s
	})

	iris.Post("/api/resetstats", func(ctx *iris.Context) {
		Logger.Debugf("resetstats\n")
		LiftSystem.forAllLifts(ResetStats)
	})

	iris.Post("/api/pause", func(ctx *iris.Context) {
		Logger.Debugf("resetstats\n")
		Paused = !Paused
	})

	go ticker()
	go eventGenerator()

	/*
		InitEventStore()
	*/

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	signal.Notify(c, syscall.SIGTERM)
	go func() {
		<-c
		//CloseEventStore()
		os.Exit(1)
	}()

	iris.Listen(":8080")

}

func ticker() {
	timer := time.NewTimer(time.Second / time.Duration(TickSpeedFactor))
	for {
		if timer != nil {
			<-timer.C
			go LiftSystem.QueueEvent(&Event{Tick, "", 0, NoDirection, nil})
		}

		if TickSpeedFactor == 0 || Paused == true {
			if timer != nil {
				timer.Stop()
				timer = nil
			}
			time.Sleep(1000 * time.Millisecond)
		} else {
			if timer != nil {
				timer.Reset(time.Second / time.Duration(TickSpeedFactor))
			} else {
				timer = time.NewTimer(time.Second / time.Duration(TickSpeedFactor))
			}
		}
	}
}

func eventGenerator() {

	random := rand.New(rand.NewSource(time.Now().UnixNano()))
	timer := time.NewTimer(time.Second / time.Duration(TickSpeedFactor) * 5)
	for {

		if timer != nil {

			Logger.Debug("eventGeenrator\n")
			<-timer.C

			floor := random.Intn(49) + 1

			var b *Event

			destFloor := 1
			dir := Down
			if floor == 1 {
				destFloor = random.Intn(48) + 2
				dir = Up
			}

			p := NewPassengerForPickup(floor, destFloor, Up)

			b = &Event{PickupRequest, "", floor, dir, p}

			if b != nil {
				go LiftSystem.QueueEvent(b)
			}
		}

		if TickSpeedFactor == 0 || Paused == true {
			if timer != nil {
				timer.Stop()
				timer = nil
			}
			time.Sleep(1000 * time.Millisecond)
		} else {
			if timer != nil {
				timer.Reset(time.Second / time.Duration(TickSpeedFactor) * 5)
			} else {
				timer = time.NewTimer(time.Second / time.Duration(TickSpeedFactor) * 5)
			}
		}

	}
}
