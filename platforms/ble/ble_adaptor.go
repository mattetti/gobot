package ble

import (
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/hybridgroup/gobot"
	"github.com/hybridgroup/gobot/platforms/ble/option"
	"github.com/paypal/gatt"
)

var (
	_                    gobot.Adaptor = (*BLEAdaptor)(nil)
	ErrConnectionTimeout               = errors.New("Connection to device timed out")
)

// Represents a Connection to a BLE Peripheral
type BLEAdaptor struct {
	name       string
	uuid       string
	device     gatt.Device
	peripheral gatt.Peripheral
	//sp        io.ReadWriteCloser
	connected bool
	//connect   func(string) (io.ReadWriteCloser, error)
	doneChan chan bool
	err      error
	// Debug flag toggles the printing out of debug messages.
	Debug bool
	// ConnectionTimeout is the time waited for the device we try to connect to.
	ConnectionTimeout time.Duration
}

// NewBLEAdaptor returns a new BLEAdaptor given a name and uuid
func NewBLEAdaptor(name, uuid string) *BLEAdaptor {
	uuid = strings.ToLower(uuid)
	uuid = strings.Replace(uuid, "-", "", -1)
	return &BLEAdaptor{
		name:              name,
		uuid:              uuid,
		connected:         false,
		doneChan:          make(chan bool),
		ConnectionTimeout: (30 * time.Second),
		// connect: func(port string) (io.ReadWriteCloser, error) {
		// 	return serial.OpenPort(&serial.Config{Name: port, Baud: 115200})
		// },
	}
}

func (b *BLEAdaptor) Name() string                { return b.name }
func (b *BLEAdaptor) UUID() string                { return b.uuid }
func (b *BLEAdaptor) Peripheral() gatt.Peripheral { return b.peripheral }

// Connect initiates a connection to the BLE peripheral. Returns true on successful connection.
func (b *BLEAdaptor) Connect() (errs []error) {
	var err error
	b.device, err = gatt.NewDevice(option.DefaultClientOptions...)
	if err != nil {
		log.Fatalf("Failed to open BLE device, err: %s\n", err)
		errs = append(errs, err)
		return errs
	}

	// Register handlers.
	b.device.Handle(
		gatt.PeripheralDiscovered(b.onDiscovered),
		gatt.PeripheralConnected(b.onConnected),
		gatt.PeripheralDisconnected(b.onDisconnected),
	)

	b.device.Init(b.onStateChanged)

	// we wait for a device to be connected or we timeout.
	select {
	case <-b.doneChan:
		return []error{b.err}
	case <-time.After(b.ConnectionTimeout):
		return []error{ErrConnectionTimeout}
	}

}

// Reconnect attempts to reconnect to the BLE peripheral. If it has an active connection
// it will first close that connection and then establish a new connection.
// Returns true on Successful reconnection
func (b *BLEAdaptor) Reconnect() (errs []error) {
	if b.connected {
		b.Disconnect()
	}
	return b.Connect()
}

// Disconnect terminates the connection to the BLE peripheral. Returns true on successful disconnect.
func (b *BLEAdaptor) Disconnect() (errs []error) {
	// if a.connected {
	// 	if err := a.sp.Close(); err != nil {
	// 		return []error{err}
	// 	}
	// 	a.connected = false
	// }
	return
}

// Finalize finalizes the BLEAdaptor
func (b *BLEAdaptor) Finalize() (errs []error) {
	return b.Disconnect()
}

// ReadCharacteristic returns bytes from the BLE device for the
// requested service and characteristic
func (b *BLEAdaptor) ReadCharacteristic(sUUID string, cUUID string) (data chan []byte, err error) {
	//defer b.peripheral.Device().CancelConnection(b.peripheral)
	fmt.Println("ReadCharacteristic")
	if !b.connected {
		log.Fatalf("Cannot read from BLE device until connected")
		return
	}

	c := make(chan []byte)
	f := func(p gatt.Peripheral, e error) {
		b.performRead(c, sUUID, cUUID)
	}

	b.device.Handle(
		gatt.PeripheralConnected(f),
	)

	b.peripheral.Device().Connect(b.peripheral)

	return c, nil
}

func (b *BLEAdaptor) performRead(c chan []byte, sUUID string, cUUID string) {
	fmt.Println("performRead")
	fmt.Printf("%x", b.Peripheral())
	s := b.getService(sUUID)
	characteristic := b.getCharacteristic(s, cUUID)

	val, err := b.peripheral.ReadCharacteristic(characteristic)
	if err != nil {
		fmt.Printf("Failed to read characteristic, err: %s\n", err)
		c <- []byte{}
	}

	c <- val
}

func (b *BLEAdaptor) getPeripheral() {

}

func (b *BLEAdaptor) getService(sUUID string) (service *gatt.Service) {
	fmt.Println("getService")
	ss, err := b.Peripheral().DiscoverServices(nil)
	fmt.Println(ss)
	fmt.Println("yo")
	fmt.Println(err)
	if err != nil {
		fmt.Printf("Failed to discover services, err: %s\n", err)
		return
	}

	fmt.Println("service")

	for _, s := range ss {
		msg := "Service: " + s.UUID().String()
		if len(s.Name()) > 0 {
			msg += " (" + s.Name() + ")"
		}
		fmt.Println(msg)

		id := strings.ToUpper(s.UUID().String())
		if strings.ToUpper(sUUID) != id {
			continue
		}

		msg = "Found Service: " + s.UUID().String()
		if len(s.Name()) > 0 {
			msg += " (" + s.Name() + ")"
		}
		fmt.Println(msg)
		return s
	}

	fmt.Println("getService: none found")
	return
}

func (b *BLEAdaptor) getCharacteristic(s *gatt.Service, cUUID string) (c *gatt.Characteristic) {
	fmt.Println("getCharacteristic")
	cs, err := b.Peripheral().DiscoverCharacteristics(nil, s)
	if err != nil {
		fmt.Printf("Failed to discover characteristics, err: %s\n", err)
		return
	}

	for _, char := range cs {
		id := strings.ToUpper(char.UUID().String())
		if strings.ToUpper(cUUID) != id {
			continue
		}

		msg := "  Found Characteristic  " + char.UUID().String()
		if len(char.Name()) > 0 {
			msg += " (" + char.Name() + ")"
		}
		msg += "\n    properties    " + char.Properties().String()
		fmt.Println(msg)
		return char
	}

	return nil
}

func (b *BLEAdaptor) onStateChanged(d gatt.Device, s gatt.State) {
	if b.Debug {
		fmt.Println("State:", s)
	}
	switch s {
	case gatt.StatePoweredOn:
		if b.Debug {
			fmt.Println("scanning...")
		}
		d.Scan([]gatt.UUID{}, false)
		return
	default:
		d.StopScanning()
	}
}

func (b *BLEAdaptor) onDiscovered(p gatt.Peripheral, a *gatt.Advertisement, rssi int) {
	isMatch := p.ID() == b.UUID()
	if b.Debug {
		fmt.Printf("Peripheral discovered ID: %s, Name: (%s), Match: %t\n", p.ID(), p.Name(), isMatch)
	}

	if !isMatch {
		return
	}

	b.connected = true
	b.peripheral = p

	// Stop scanning once we've got the peripheral we're looking for.
	p.Device().StopScanning()

	if b.Debug {
		fmt.Printf("\nPeripheral ID:%s, NAME:(%s)\n", p.ID(), p.Name())
		fmt.Println("\tLocal Name        =", a.LocalName)
		fmt.Println("\tTX Power Level    =", a.TxPowerLevel)
		fmt.Println("\tManufacturer Data =", a.ManufacturerData)
		fmt.Println("\tService Data      =", a.ServiceData)
		fmt.Println()
	}

	p.Device().Connect(p)
}

func (b *BLEAdaptor) onConnected(p gatt.Peripheral, err error) {
	if b.Debug {
		fmt.Println("Connected")
	}
	defer func() {
		close(b.doneChan)
		p.Device().CancelConnection(p)
	}()

	if err := p.SetMTU(500); err != nil {
		b.err = fmt.Errorf("Failed to set MTU, err: %s\n", err)
		return
	}

}

func (b *BLEAdaptor) onDisconnected(p gatt.Peripheral, err error) {
	fmt.Println("Disconnected")
}
