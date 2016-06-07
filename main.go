package main

import (
	"flag"
	"fmt"
	"github.com/kataras/iris"
	"github.com/kataras/iris/config"
	"github.com/kataras/iris/websocket"
	"math/rand"
	"strconv"
	"time"
	//"net/http"
	//	"text/template"
)

var addr = flag.String("addr", ":8080", "http service address")

var TickSpeedFactor int = 1

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

	iris.Get("/api/lift/:liftId", func(ctx *iris.Context) {
		liftId := ctx.Param("liftId")
		lift := GetLift(liftId)
		ctx.JSON(iris.StatusOK, lift)
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

	go ticker()
	go eventGenerator()

	iris.Listen(":8080")

}

func ticker() {
	timer := time.NewTimer(time.Second / time.Duration(TickSpeedFactor))
	for {
		<-timer.C
		go LiftSystem.QueueEvent(&Event{Tick, "", 0, NoDirection, 0})
		timer.Reset(time.Second / time.Duration(TickSpeedFactor))
	}
}

func eventGenerator() {

	random := rand.New(rand.NewSource(time.Now().UnixNano()))
	timer := time.NewTimer(time.Second / time.Duration(TickSpeedFactor) * 5)
	for {
		Logger.Debug("eventGeenrator\n")

		<-timer.C
		liftId := "lift" + strconv.Itoa(random.Intn(4)+1)
		lift := GetLift(liftId)

		floor := random.Intn(49) + 1

		var b *Event

		if lift.Status == Idle {
			p := NewPassengerForPickup(floor, Up)
			b = &Event{DropoffRequest, liftId, floor, NoDirection, p.Id}
		} else {
			randomEvent := random.Intn(2)
			dir := Up
			switch randomEvent {

			case 0:
				dir = Up

			case 1:
				dir = Down

			default:
			}

			p := NewPassengerForPickup(floor, dir)
			b = &Event{PickupRequest, "", floor, dir, p.Id}

		}
		if b != nil {
			go LiftSystem.QueueEvent(b)
		}

		timer.Reset(time.Second / time.Duration(TickSpeedFactor) * 5)
	}
}
