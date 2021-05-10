package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/SMerrony/tello"
)

const broadcast_addr = "255.255.255.255"

func ctrlMssgLn(message string) {
	log.Printf("[CTRL] %v \n", message)
}

func main() {
	log.Printf("Trying to connect... to PH-JSX-TB-Tello1")
	drone := new(tello.Tello)
	err := drone.ControlConnectDefault()
	if err != nil {
		log.Fatalf("%v", err)
	}

	flying := false
	ctrlMssgLn("Connected with Drone.")
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
	// drone.TakeOff() and...
	// emergecy stop after 50 ms?

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
	drone.ThrowTakeOff()
	time.Sleep(3 * time.Second)
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
		fmt.Println("CTRL >>> Cleared For Takeoff")
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
	drone.Anticlockwise()

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
