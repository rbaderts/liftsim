package main

import (
	//"encoding/json"
	"flag"
	"github.com/kataras/iris"
    "github.com/kataras/go-template/html"

        "github.com/kataras/go-websocket"
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

//var irisFramework *iris.Framework;
//var party *iris.MuxAPI;

type mypage struct {
    Title   string
    Message string
}


func main() {

 // example 2: iris.New(iris.Configuration{IsDevelopment:true, Charset: "UTF-8", Sessions: iris.SessionsConfiguration{Cookie:"mycookieid"}, Websocket: iris.WebsocketConfiguration{Endpoint:"/my_endpoint"}})



	//irisFramework = iris.New(config)



	//iris.Config().Render.Template.Engine = iris.HTMLTemplate

	flag.Parse()

	//InitBuilding()
	InitLiftsSystem()

    iris.Config.IsDevelopment = true // this will reload the templates on each request, defaults to false

    iris.UseTemplate(html.New(html.Config{
           Layout: "layouts/layout.html",
       }))


	iris.StaticWeb("/css", "./resources/css")
	iris.StaticWeb("/js", "./resources/js")
	iris.StaticWeb("/img", "./resources/img")

  	iris.Get("/", func(ctx *iris.Context){
  		if err := ctx.Render("index.html", nil); err != nil {
    			println(err.Error())
     	}
        //ctx.Render("index.html", mypage{"My Page title", "Hello world!"})
    })

    iris.Config.Websocket.Endpoint = "/updates"
    iris.Websocket.OnConnection(func(c iris.WebsocketConnection) {
		WSC = c
    })


//func (ws *WebsocketServer) OnConnection(connectionListener func(WebsocketConnection)) {


	iris.Get("/api/fastforward", func(ctx *iris.Context) {
		var state *LiftSystemState = &LiftSystemState{}
		state.Lifts = make(map[string]*Lift)
		ctx.JSON(iris.StatusOK, state)
	})

	iris.Get("/api/rewind", func(ctx *iris.Context) {

		var state *LiftSystemState = &LiftSystemState{}
		state.Lifts = make(map[string]*Lift)

		ctx.JSON(iris.StatusOK, state)

	})

	iris.Post("/api/setspeed", func(ctx *iris.Context) {
		speed := ctx.PostValue("speed")
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
