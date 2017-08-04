
        var cycle
        var biggestCycle = 0
        var cycleBadge = document.getElementById("cycle");
        var speed = document.getElementById("speed");
        var eventfrequency = document.getElementById("eventfrequency");
        var ws = new WebSocket('ws://' + window.location.host + '/updates');
        const Idle = 'Idle';
        const MovingUp = 'MovingUp';
        const MovingDown = 'MovingDown';
        const DoorOpening = 'DoorOpening';
        const DoorClosing = 'DoorClosing';

        const Pickup = 'Pickup';
        const Dropoff = 'Dropoff';

        const Up = 'Up';
        const Down = 'Down';
        const NoDirection = 'NoDirection';

        var lastFloors = {};
        var images=[]
       
        ws.onopen = function () {
            console.log("Websocket connection enstablished");
        };

        ws.onerror = function (error) {
            console.log("Websocket error: " + error);
        };

        
        ws.onmessage = function(lift) {

            console.log("Websocket onmessage: " +lift)
                   var l = JSON.parse(lift.data);
                   console.log(JSON.stringify(l, null, 4))
                   if (l.command == 'UpdateLift') {
                      var data = l.data;
                      draw(data, data.liftId)
                   }
                   else if (l.command == 'UpdateLiftSystem') {
                      var data = l.data;
                      cycle = data.cycle
                      cycleBadge.innerHTML = data.cycle.toString()
                      if (cycle > biggestCycle) {
                           biggestCycle = cycle
                      }
                      for (var liftId in data.lifts) {
                           draw(data.lifts[liftId], liftId)
                      }

                      //speed.value = data.speed
                      //eventfrequency.value = data.eventSpeed

                   }

                 /*  } else if (l.command == 'UpdateLiftSystem') {
                       var data = l.data;
                       cycle = data.cycle
                       cycleBadge.innerHTML = data.cycle.toString()
                       if (cycle > biggestCycle) {
                            biggestCycle = cycle
                       }
                       for (var liftId in data.lifts) {
                            draw(data.lifts[liftId], liftId)
                       }
                    }
                    */
               };

           


        var setspeed = function() {
          $.post("/api/speed", {
            "speed": sessvars.speed
          })
        }

        var seteventfrequency = function() {
          $.post("/api/eventfrequency", {
            "eventfrequency": sessvars.eventfrequency
          })
        }

        var resetstats = function() {
          $.post("/api/resetstats", {
          })
        }

        $('#speedsubmit').click(function() {
          sessvars.speed=$('#speed').val()
          if (sessvars.speed >= 0) {
              setspeed()
          }
        })

        $('#eventfrequencysubmit').click(function() {
          sessvars.eventfrequency=$('#eventfrequency').val()
          if (sessvars.eventfrequency >= 0) {
              seteventfrequency()
          }
        })

        $('#resetstats').click(function() {
          resetstats()
        })


        $('#stepback').click(function() {

            if (cycle > 1) {
                cycle = cycle - 1
                $.get( "/api/rewind", function( data ) {
                      for (var liftId in data.lifts) {
                          draw(data.lifts[liftId], liftId)
                      }
                })

                cycleBadge.innerHTML = cycle.toString()

                $('#stepforward').disabled = false
            } else {
                $('#stepback').disabled = true
            }
        })


        $('#stepforward').click(function() {
            cycle = cycle + 1

            if (cycle > 1) {
                $('#stepback').disabled=false
            }

            $.get( "/api/fastforward", function( data ) {
                  for (var liftId in data.lifts) {
                      draw(data.lifts[liftId], liftId)
                  }
            })

            cycleBadge.innerHTML = cycle.toString()
            if (cycle >= biggestCycle) {
                $('#stepforward').disabled = true
            }
        })

        $('#pause').click(function() {
          pause = document.getElementById("pause");
          pause.style.display = 'none'
          pause.hidden = true

          play = document.getElementById("play");
          play.hidden = false
          play.style.display = 'block'
          $.post("/api/pause", {
          })
        })

        $('#play').click(function() {
          pause = document.getElementById("pause");
          pause.hidden = false
          pause.style.display = 'block'

          play = document.getElementById("play");
          play.hidden = true
          play.style.display = 'none'

          $.post("/api/unpause", {
          })
        })



//        var updates = new WebSocket("ws://" + window.location.host + "/updates");

/*
        updates.onopen = function () {
              console.log("Websocket connection enstablished");
        };

        updates.onmessage = function(lift) {
            var l = JSON.parse(lift.data)
            console.log(JSON.stringify(l, null, 4))
            if (l.command == 'UpdateLift') {
                var data = l.data;
                draw(data, data.liftId) 
            } else if (l.command == 'UpdateLiftSystem') {
                var data = l.data;
                cycle = data.cycle
                cycleBadge.innerHTML = data.cycle.toString()
                if (cycle > biggestCycle) {
                    biggestCycle = cycle
                }
                for (var liftId in data.lifts) {
                    draw(data.lifts[liftId], liftId)
                }
            }
        };
        */


        function preloadimages(arr){
            var arr=(typeof arr!="object")? [arr] : arr //force arr parameter to always be an array
            for (var i=0; i<arr.length; i++){
                  images[i]=new Image()
                  images[i].src=arr[i]
            }
        }

        preloadimages(['/static/img/stickman.gif', '/static/img/stopsign.gif', '/static/img/uparrow.gif', '/static/img/downarrow.gif', '/static/img/movinguparrow.gif', '/static/img/movingdownarrow.gif']);


        function draw(data, id) {
  
           var canvas = document.getElementById(id)
           console.log("draw: " +  JSON.stringify(data, null, 4))
           var floor = data.floor

           if (canvas != null && canvas.getContext) {

               var ctx = canvas.getContext("2d");

               // The Header
               drawHeader(data, id)

               // The car
               drawLiftCar(ctx, data, id, floor)

               // The stop or man at each floor
               drawFloorStops(ctx, data)

               // The Floor indicator
               drawFloorIndicator(ctx, data, floor)

               // The floor panel
               drawFloorPanel(ctx, data)
   
           } 
        }


      function drawHeader(data, id) {
           console.log("drawHeader: " +  JSON.stringify(data, null, 4))
           var canvas = document.getElementById(id+"_header")
           var ctx = canvas.getContext("2d");
           if (ctx != null) {
                ctx.font="bolder 10px sans-serif"
                ctx.clearRect(1, 1, 94, 70)
                ctx.textAlign = "left"
                ctx.textBaseline = "middle"

                var x = 2
                var y = 6
                var flr = 1

                ctx.fillText("Occupants: " + data.occupants, x,  y)

                var averageExtraFloors = data.totalExtraFloors / data.totalRides
                if (data.totalRides == 0) {
                    averageExtraFloors = 0
                }
                //ctx.clearRect(1, 20, 94, 70)
//1                ctx.fillText("Avg Extra Floors: " + averageExtraFloors.toPrecision(2), x,  y)

                //ctx.clearRect(1, 20, 94, 70)
                var totalOccupants = data.totalRides

                var y = 18
                ctx.fillText("Total Riders: " + totalOccupants, x,  y)
            }

        }



      function drawLiftCar(ctx, data, id, floor) {

           var lift_xoffset = 70
           var stops_xoffset = 82


            if (id in lastFloors) {
                ctx.clearRect(lift_xoffset - 1, 500 - (10 * lastFloors[id]) - 1, 14, 14);
            }

            if (data.status == DoorOpening || data.status == DoorClosing) {
                 ctx.strokeStyle = "#FF0000"
                 ctx.strokeRect(lift_xoffset, 500 - 10 * floor, 10, 10)
                 ctx.clearRect(stops_xoffset, 500-(10*floor), 12, 12)
            } else if (data.status == MovingUp || data.status == MovingDown) {
                 ctx.strokeStyle = "#000000"
                 ctx.fillStyle = "#00FF00"
                 ctx.fillRect(lift_xoffset, 500 - 10 * floor, 10, 10)
                 ctx.strokeRect(lift_xoffset, 500 - 10 * floor, 10, 10)
            } else if (data.status == Idle) {
                 ctx.strokeStyle = "#00FF00"
                 ctx.strokeRect(lift_xoffset, 500 - 10 * floor, 10, 10)
            }

            ctx.strokeStyle = "#000000"
            ctx.fillStyle = "#000000"
            ctx.font="10px Arial";
            ctx.textAlign = "center"
            ctx.textBaseline="bottom";
            ctx.fillText(data.occupants.toString(), lift_xoffset+5, 500-(10*floor)+10)

            lastFloors[id] = floor
      }

      function  drawFloorStops(ctx, data) {

           var stops_xoffset = 82

            if (data.stops != null) {
                for (var fl=1; fl<=50; fl++) {
                	if (fl.toString() in data.stops) {
						for (var i=0; i<data.stops[fl].length; i++) {
							var flr = parseInt(fl);
							var s = data.stops[fl][i]
							if (images.length == 0) {
							   ctx.fillRect(stops_xoffset + (i*14), 500-10*flr, 12, 12) 

							} else if (s.stopType == Dropoff) {
							   ctx.drawImage(images[1], stops_xoffset + (i*14), 500-10*flr, 12, 12)
							} else if (s.stopType == Pickup) {
							   ctx.drawImage(images[0], stops_xoffset + (i*14), 500-10*flr, 12, 12)
						   }
						}
					} else {
					   ctx.clearRect(stops_xoffset + (i*14), 500-10*flr, 12, 12) 
					}
                }
            }

      }

      function drawFloorIndicator(ctx, data, floor) {

            ctx.strokeStyle = "#000000"
            ctx.fillStyle = "#000000"

            ctx.font="20px Arial";
            ctx.clearRect(2, 2, 30, 30)
            ctx.textAlign = "center"
            ctx.textBaseline="bottom";
            ctx.fillText(floor.toString(), 17, 28);

            ctx.strokeStyle = "#909090"
            ctx.strokeRect(2, 2, 30, 30)

            if (images.length > 0) {
				if (data.direction == Up) {
                    if (data.status == MovingUp) {
     					ctx.drawImage(images[4], 38, 4, 26, 26);
                    } else {
     					ctx.drawImage(images[2], 38, 4, 26, 26);
                    }
				} else if (data.direction == Down) {
                    if (data.status == MovingDown) {
					     ctx.drawImage(images[5], 38, 4, 26, 26);
                    } else {
     					ctx.drawImage(images[3], 38, 4, 26, 26);
                    }
				} else {
					ctx.clearRect(38, 4, 26, 26);
				}
            }
			ctx.strokeStyle = "#909090"
			ctx.strokeRect(36, 2, 30, 30)

        }


      function drawFloorPanel(ctx, data) {

            var floorpanel_yoffset = 36
            ctx.font="800 9px Arial"
            ctx.clearRect(0, 36, 44, 158)
            ctx.strokeStyle = "#909090"
            ctx.strokeRect(0, 36, 44, 158)

            ctx.textAlign = "center"
            ctx.textBaseline = "middle"

            ctx.strokeStyle = "#000000"
            ctx.fillStyle = "#000000"
            var x = 8
            var y = floorpanel_yoffset + 8
            var flr = 1
            for (row = 1; row <= 17; ++row) {
                for (col = 1; col <= 3 ; ++col) {
                      flr = (row) + ((col-1) * 17)
                      if (flr <= 51) {
                          if (isStop(data, flr)) {
                               ctx.fillStyle = "#FF0000"
                          } else {
                               ctx.fillStyle = "#000000"
                          }
                          ctx.fillText(flr.toString(), x + ((col-1) * 13), y + ((row-1) * 9))
                      }
                }
            }
      }

      function isStop(data, floor) {

          if (data.stops != null && data.stops[floor]) {
               return true
          }
          return false
      }

