package main

import (
	"fmt"
	"time"

	"github.com/hybridgroup/gobot"
	"github.com/hybridgroup/gobot/platforms/ble"
)

func main() {
	gbot := gobot.NewGobot()

	bleAdaptor := ble.NewBLEAdaptor("ble", "0BC3340A-A54C-4AE5-9536-12470A730A95")
	bleAdaptor.Debug = true
	battery := ble.NewBLEBatteryDriver(bleAdaptor, "battery")

	work := func() {
		fmt.Println("Working...")

		gobot.After(3*time.Second, func() {
			fmt.Println(battery.GetBatteryLevel())
		})
	}

	robot := gobot.NewRobot("bleBot",
		[]gobot.Connection{bleAdaptor},
		[]gobot.Device{battery},
		work,
	)

	gbot.AddRobot(robot)

	gbot.Start()
}
