package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"
)

var groundOnline bool = true
var towerOnline bool = false
var droneOnline bool = false
var flying bool = false

var d Drone
var l log.Logger
var w Worker

func droneMssgLn(message string) {
	droneOnline = true
	log.Printf("%v [HELI]: %v \n", time.Now().Format(".9999"), message)
}

func ctrlMssgLn(message string) {
	log.Printf("%v [CTRL]: %v \n", time.Now().Format(".9999"), message)
}

func grndMssgLn(message string) {
	log.Printf("%v [GRND]: %v \n", time.Now().Format(".9999"), message)
}

func towerMssgLn(message string) {
	towerOnline = true
	log.Printf("%v [T0wr]: %v \n", time.Now().Format(".9999"), message)
}

// Registered
func init() {
	// Create logger
	l := log.New(log.Writer(), log.Prefix(), log.Flags())

	// Create the drone (CONNECT!)
	d := New(l)

	groundOnline = true
	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		fmt.Println("[[SYS]] Received SIGTERM from User... Running cleanup.")
		// Run Cleanup

		for flying == true {
			if err := d.Land(); err != nil {
				droneMssgLn("PH-JSX, FAILED land.!")
				l.Println(fmt.Errorf("main: taking off failed: %w", err))
				d.Land()
				return
			} else {
				flying = false
				droneMssgLn("PH-JSX, Switching to ground!")
			}
		}
		if flying {
			d.Land()
		} else {
			towerOnline = false
		}
		groundOnline = false
		fmt.Println("[[SYS]] Ground Offline. End of Service. Goodbye!")
		os.Exit(0)
	}()
}

func ground(activeDrone *Drone) {

	grndMssgLn(" Active DRONE: PH-JSX")
}

func main() {

	// Create logger
	l2 := log.New(log.Writer(), "ctrl", log.Flags())
	// Create worker
	w := NewWorker(WorkerOptions{Logger: l2})

	fmt.Println("[[SYS]] Init READY for intterupts!")

	if !groundOnline {
		fmt.Printf("ERROR ground is not online (misconfig?)")
		os.Exit(1)
	} else {
		grndMssgLn("ONLINE.")
	}

	grndMssgLn(fmt.Sprintf("Try Connection To Tello %v", "PH-JSX-T1 Ultra-Light"))

	// Create logger
	l := log.New(log.Writer(), log.Prefix(), log.Flags())

	// Create the drone (CONNECT!)
	d := New(l)

	ground(d) // REGISTER DRONE WITH GROUND

	// Handle signals
	w.HandleSignals(TermSignalHandler(func() {
		// Make sure to land on term signal
		if err := d.Land(); err != nil {
			l.Println(fmt.Errorf("GRND: landing failed: %w", err))
			return
		}
	}))

	// NO VIDEO!

	// Handle take off event
	d.On(TakeOffEvent, func(interface{}) {
		ctrlMssgLn("PH-JSX-Tello 1 airborne")
		towerMssgLn(" I CAN SEE PH-JSX airborne")
		flying = true
	})

	// Start the drone
	if err := d.Start(); err != nil {
		grndMssgLn(fmt.Sprintf("PH-JSX unreachable: %w", err))
		droneOnline = false
		return
	}
	defer d.Close()

	towerMssgLn("Tower is online.")
	// Execute DRONE in a (single loose) task
	w.NewTask().Do(func() {
		// DO NOT Start video
		droneMssgLn("PH-JSX, Requesting ATIS Information!")
		// Take off

		// Log state
		l.Printf("DRONE: state is: %+v\n", d.State())
		droneMssgLn("PH-JSX, Requesting takeOff..!")

		if err := d.MissionModeOn(); err != nil {
			droneMssgLn("PH-JSX, Failed MON")
			l.Println(fmt.Errorf("main: mon: %w", err))
			return
		} else {
			droneMssgLn("PH-JSX, READY FOR MISSION!")
		}

		if !towerOnline {
			towerMssgLn("ERROR TOWER SHOULD BE ONLINE.")
			os.Exit(1)
		} else {
			towerMssgLn("PH-JSX: Cleared for take-off...")

		}
		if err := d.TakeOff(); err != nil {
			droneMssgLn("PH-JSX, Failed Takeoff")
			l.Println(fmt.Errorf("main: taking off failed: %w", err))

			return
		} else {
			flying = true
			droneMssgLn("PH-JSX, AIRBORNE!")
		}

		ctrlMssgLn("GO to Mission Pad 3, height FL20")
		if err := d.GoMissionPad(-60, 50, 200, 20, 3); err != nil {
			droneMssgLn("PH-JSX, mission failed! .. landing")
			l.Println(fmt.Errorf("main: mission failed: %w", err))
			d.Land()
			return
		} else {
			droneMssgLn("PH-JSX, Mission Complete!")
			ctrlMssgLn("RETURN TO PAD")
		}

		ctrlMssgLn("GO to Mission Pad 3 + 50 to land, FL5")
		if err := d.GoMissionPad(0, 0, 50, 20, 3); err != nil {
			droneMssgLn("PH-JSX, mission failed! .. landing")
			l.Println(fmt.Errorf("main: mission failed: %w", err))
			d.Land()
			return
		} else {
			droneMssgLn("PH-JSX, Mission Complete!")
			ctrlMssgLn("Cleared to LAND")
		}
		if err := d.Land(); err != nil {
			droneMssgLn("PH-JSX, FAILED land.!")
			l.Println(fmt.Errorf("main: taking off failed: %w", err))
			d.Land()
			return
		} else {
			flying = false
			droneMssgLn("PH-JSX, Switching to ground!")
		}

		if flying {
			d.Land()
			fmt.Println("[DRONE] Land")
			time.Sleep(1 * time.Second)
		}

		fmt.Println("[DRONE] Check Landed .... ")
		fmt.Println("check if landed properly...")

		// TODO check it!
		/* 		if !drone.GetFlightData().Flying {
		   			fmt.Println("[CTRL] Eagle has landed...")
		   			drone.ControlDisconnect()
		   			fmt.Println("[CTRL] Control disconnected")
		   			fmt.Println("[DRONE] Control disconnected")
		   			os.Exit(0)
		   		} else {
		   			fmt.Println("[DRONE] Sleep 1 && RETRY LANDING")
		   			time.Sleep(1 * time.Second)
		   			drone.Land()
		   			time.Sleep(3 * time.Second)
		   			if !drone.GetFlightData().Flying {
		   				fmt.Println("[DRONE] Safe on ground...")
		   				fmt.Println("[CTRL] Eagle has landed...")
		   				drone.ControlDisconnect()
		   				fmt.Println("[CTRL] Control disconnected")
		   				fmt.Println("[DRONE] Control disconnected")
		   				os.Exit(0)
		   			} else {
		   				// 3rd times a charm?
		   				fmt.Println("Watch out... 3 times ignoreing your land commmand?!? NOT GOOD")
		   				drone.Land()
		   			}
		   		}
		   		os.Exit(0) */

		if err := d.MissionModeOff(); err != nil {
			droneMssgLn("PH-JSX, Failed MON")
			l.Println(fmt.Errorf("main: mon: %w", err))
			return
		} else {
			droneMssgLn("PH-JSX, GOING OFFLINE!")
		}

		droneOnline = false
		ctrlMssgLn("GOING OFFLINE. Bye! End of service")
		// Stop worker
		w.Stop()
		d.stateConn.Close()
		d.cmdConn.Close()
		os.Exit(0)
	})

	fmt.Println("Start Main for loop. at 1500 hrz...")
	for {
		time.Sleep(1500 * time.Millisecond)
		//fmt.Print("you can exit now... \n")
		if !droneOnline {
			towerMssgLn("LostConnection with Drone!!! HELP...")
		}
	}
}

/*
func backupOFmain() {
	log.Printf("Try Connection To Tello %v", "PH-JSX-T1 Ultra-Light")
	drone := new(jtello.Tello)
	err := drone.ControlConnectDefault()
	if err != nil {
		log.Fatalf("%v", err)
	}

	flying := false

	ctrlMssgLn("Connected with Drone.")

	log.Printf("CONNECTED TO:  %v", "PH-JSX-T1 Ultra-Light")
	fmt.Println("I made connection with the Drone")
	fmt.Printf("[CTRL] %v Percent BATTERY, Checking... \n", drone.GetFlightData().BatteryPercentage)
	time.Sleep(5 * time.Second)
	//fmt.Printf("SSID: %s\n", drone.GetFlightData().SSID)
	//fmt.Printf("Version: %s\n", drone.GetFlightData().Version)
	fmt.Printf("[DRONE] BATTERY CHECK %v Percent \n", drone.GetFlightData().BatteryPercentage)
	fmt.Printf("[CTRL] %v Percent BATTERY, Checking... \n", drone.GetFlightData().BatteryPercentage)

	if 20 >= drone.GetFlightData().BatteryPercentage {
		fmt.Printf("[CTRL] Flight request .. %v battery is too low DENIED... \n", drone.GetFlightData().BatteryPercentage)
		fmt.Printf("BAT-perc (CRIT! <20perc ): %v\n", drone.GetFlightData().BatteryPercentage)
		log.Fatal("Cannot fly low battery!!!")
		os.Exit(1)
	} else {
		fmt.Printf("[CTRL] PH-JSX-Tello1... with %v battery. Cleared for takeOff in 3 sec. \n", drone.GetFlightData().BatteryPercentage)
		fmt.Println("StartTime: ", time.Now().Format(time.RFC3339Nano))
		fmt.Printf("nSpeed? %v\n", drone.GetFlightData().NorthSpeed)
		log.Printf("3 seconds wait to Throw TakeOff... \n")
	}

	ctrlMssgLn("Pre-Flight Checks...")
	fmt.Println("[DRONE] Cleared for take off 3...")
	time.Sleep(time.Second)
	fmt.Println("[DRONE] Cleared for take off 2...")
	time.Sleep(time.Second)
	fmt.Println("[DRONE] Cleared for take off 1...")
	time.Sleep(time.Second)
	fmt.Println("[CTRL] Throwing Take off? you have 3 sec...")
	log.Printf("[CTRL] ThrowTakeOff 3 sec...\n")
	drone.ThrowTakeOff()
	start_time := time.Now()
	time.Sleep(3 * time.Second)
	ctrlMssgLn("Trow Takeoff")
	drone.ThrowTakeOff()
	time.Sleep(3 * time.Second)
	ctrlMssgLn("3sec after Trow Takeoff END. Check if Flying?")
	fmt.Println("[DRONE] Ready to Fly!")

	if drone.GetFlightData().Flying {
		log.Printf("[CTRL] TB-Tello == GO. CONFIRMED AIRBORNE!!! \n")
		start_time = time.Now()
		// Check if drone is flying...
		fmt.Println("[CTRL] PH-JSX-Tello1 is Airbone!")
		fmt.Println("[DRONE] Confirmed")
		flying = true
	} else {
		flying = false
		// not flying so let's takeOff
		log.Printf("[CTRL] Normal TakeOff in 3 seconds...\n")
		time.Sleep(3 * time.Second)
		log.Printf("CTRL: Did not perform Throw Takeoff... \n")
		fmt.Println("CTRL >>> Takeoff")
		drone.TakeOff()
		log.Printf("[CTRL] Take-off!!! check if Airbourne?...\n")
		time.Sleep(1 * time.Second)
		start_time = time.Now()
		fmt.Println("[DRONE] TakeOff")
	}

	fmt.Println("[CTRL] Sleep 1")
	time.Sleep(1 * time.Second)
	fmt.Println("[CTRL] Tower, is PH-JSX-Tello1 airborne?")

	if drone.GetFlightData().Flying || flying {
		flying = true
		log.Printf("[CTRL] AIRBORNE! Confirmed Normal Take-off")
		fmt.Println("[DRONE] TWR, PH-JSX-Tello 1 airborne!")
		fmt.Println("[CRTL] PH-JSX-Tello1 Contact Departure")
		fmt.Println("[DRONE] PH-JSX-Tello1, Roger, Wilco!")
	} else {
		fmt.Println("[CRTL] PH-JSX-Tello1 Abort Flight")
		log.Fatal("expected to be airborne...")
	}

	fmt.Println("[CTRL] Sleep 1")
	time.Sleep(1 * time.Second)
	fmt.Println("[CTRL] PH-JSX-Tello1, CHECK MISSION PAD 1/3.")

	fmt.Println("[DRONE] increate to FL 10")
	drone.AutoFlyToHeight(5)
	fmt.Println("[CTRL] Sleep 2")
	time.Sleep(1 * time.Second)
	fmt.Println("[CTRL] Sleep 1")
	time.Sleep(1 * time.Second)

	drone.Hover()
	fmt.Println("[CTRL] Do a backflip!")
	time.Sleep(1 * time.Second)
	drone.BackFlip()
	fmt.Println("[DRONE] BackFLip! Right Back.")
	time.Sleep(2 * time.Second)
	fmt.Println("[CTRL] Request Palm Land...")
	drone.PalmLand()
	fmt.Println("[DRONE] Request Palm Land... 3... 2... 1...")
	time.Sleep(3 * time.Second)
	fmt.Println("[DRONE] END palm land")
	if flying {
		time.Sleep(1 * time.Second)
		fmt.Println("CTRL >>> LAND")
		drone.Land()
		fmt.Println("[DRONE] Land")
		fmt.Println("[CTRL] Sleep 3")
		time.Sleep(1 * time.Second)
		fmt.Println("[CTRL] Sleep 2")
		time.Sleep(1 * time.Second)
		fmt.Println("[CTRL] Sleep 1")
		time.Sleep(1 * time.Second)

		drone.Land()
		fmt.Println("[DRONE] Land")
		time.Sleep(1 * time.Second)
	} else {
		land_time := time.Now()
		fmt.Println("Land Time: ", land_time.Format(time.RFC3339Nano))
		fmt.Println("[DRONE] Eagle has landed...")
		drone.ControlDisconnect()
		fmt.Println("[DRONE] Control disconnected")
		total_flight := land_time.Sub(start_time)
		fmt.Printf("[CTRL] You flight was: %v time", total_flight)
		fmt.Println("[DRONE] Welcome to BenneBronx RotteGat Airport...")
		os.Exit(0)
	}

	fmt.Println("CTRL >>> LAND")
	fmt.Println("[DRONE] Sleep 1")
	time.Sleep(1 * time.Second)

	if flying {
		drone.Land()
		fmt.Println("[DRONE] Land")
		time.Sleep(1 * time.Second)
	}

	fmt.Println("[DRONE] Check Landed .... ")
	fmt.Println("check if landed properly...")

	if !drone.GetFlightData().Flying {
		fmt.Println("[CTRL] Eagle has landed...")
		drone.ControlDisconnect()
		fmt.Println("[CTRL] Control disconnected")
		fmt.Println("[DRONE] Control disconnected")
		os.Exit(0)
	} else {
		fmt.Println("[DRONE] Sleep 1 && RETRY LANDING")
		time.Sleep(1 * time.Second)
		drone.Land()
		time.Sleep(3 * time.Second)
		if !drone.GetFlightData().Flying {
			fmt.Println("[DRONE] Safe on ground...")
			fmt.Println("[CTRL] Eagle has landed...")
			drone.ControlDisconnect()
			fmt.Println("[CTRL] Control disconnected")
			fmt.Println("[DRONE] Control disconnected")
			os.Exit(0)
		} else {
			// 3rd times a charm?
			fmt.Println("Watch out... 3 times ignoreing your land commmand?!? NOT GOOD")
			drone.Land()
		}
	}
	os.Exit(0)
}
*/
