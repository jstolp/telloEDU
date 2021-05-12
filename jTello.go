package main

// jtello - MIT license - https://github.com/asticode/go-jtello
import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"strconv"
	"sync"
	"time"
)

// Defaults for Tello Edu
var (
	defaultTimeout = 5 * time.Second
	cmdAddr        = "192.168.10.1:8889"
	respAddr       = ":8889"
	stateAddr      = ":8890"
	//videoAddr      = ":11111" unimplented
)

// Events
const (
	LandEvent    = "land"
	StateEvent   = "state"
	TakeOffEvent = "take.off"
)

// Flip directions
const ( // add 4 more? :D
	FlipBack    = "b"
	FlipForward = "f"
	FlipLeft    = "l"
	FlipRight   = "r"
)

// ErrNotConnected is the error thrown when trying to send a cmd while not connected to the drone
var ErrNotConnected = errors.New("jtello: not connected")

// Drone represents an object capable of interacting with the SDK
type Drone struct {
	cancel    context.CancelFunc
	cmdConn   *net.UDPConn
	cmds      map[*cmd]bool
	ctx       context.Context
	e         *Eventer
	lr        string
	mc        *sync.Mutex // Locks cmds
	ms        *sync.Mutex // Locks s
	msc       *sync.Mutex // Locks sendCmd
	ol        *sync.Once  // Limits Close()
	oo        *sync.Once  // Limits Connect()
	rc        *sync.Cond
	s         *State
	stateConn *net.UDPConn
}

// New creates a new Drone
func New(l StdLogger) *Drone {
	return &Drone{
		cmds: make(map[*cmd]bool),
		e:    NewEventer(EventerOptions{}),
		mc:   &sync.Mutex{},
		msc:  &sync.Mutex{},
		ms:   &sync.Mutex{},
		ol:   &sync.Once{},
		oo:   &sync.Once{},
		rc:   sync.NewCond(&sync.Mutex{}),
		s:    &State{},
	}
}

// State returns the drone's state
func (d *Drone) State() State {
	d.ms.Lock()
	defer d.ms.Unlock()
	return *d.s
}

// On adds an event handler
func (d *Drone) On(name string, h EventerHandler) {
	d.e.On(name, h)
}

// Close closes the drone properly
func (d *Drone) Close() {
	// Make sure to execute this only once
	d.ol.Do(func() {
		// Cancel context
		if d.cancel != nil {
			d.cancel()
		}

		// Reset once
		d.oo = &sync.Once{}

		// Stop and reset eventer
		d.e.Stop()
		d.e.Reset()

		// Reset cmds
		d.cmds = make(map[*cmd]bool)

		// Close connections
		if d.cmdConn != nil {
			d.cmdConn.Close()
		}
		if d.stateConn != nil {
			d.stateConn.Close()
		}
		//if d.videoConn != nil { // DO NOT DO VIDEO
		//d.videoConn.Close()
		//}
	})
}

// Start starts to the drone
func (d *Drone) Start() (err error) {
	// Make sure to execute this only once
	d.oo.Do(func() {
		// Create context
		d.ctx, d.cancel = context.WithCancel(context.Background())

		// Reset once
		d.ol = &sync.Once{}

		// Start eventer
		go d.e.Start(d.ctx)

		// Handle state
		if err = d.handleState(); err != nil {
			err = fmt.Errorf("jtello: handling state failed: %w", err)
			return
		}

		// do not Handle video

		// Handle commands
		if err = d.handleCmds(); err != nil {
			err = fmt.Errorf("jtello: handling commands failed: %w", err)
			return
		}
	})
	return
}

func (d *Drone) handleState() (err error) {
	// Create laddr
	var laddr *net.UDPAddr
	if laddr, err = net.ResolveUDPAddr("udp", stateAddr); err != nil {
		err = fmt.Errorf("jtello: creating laddr failed: %w", err)
		return
	}

	// Listen
	if d.stateConn, err = net.ListenUDP("udp", laddr); err != nil {
		err = fmt.Errorf("jtello: listening failed: %w", err)
		return
	}

	// Read state
	go d.readState()
	return
}

func dLn(message string) {
	log.Printf("[CTRL] %v \n", message)
}

func (d *Drone) readState() {
	for {
		// Check context
		if d.ctx.Err() != nil {
			return
		}

		// Read
		b := make([]byte, 2048)
		n, err := d.stateConn.Read(b)
		if err != nil {
			if d.ctx.Err() == nil {
				fmt.Printf("jtello: reading state failed: %w", err)
			}
			continue
		}

		// Create state
		s, err := newState(string(bytes.TrimSpace(b[:n])))
		if err != nil {
			fmt.Errorf("jtello: creating state failed: %w", err)

			continue
		}

		// Update state
		d.ms.Lock()
		*d.s = s
		d.ms.Unlock()

		// Dispatch
		d.e.Dispatch(StateEvent, s)
	}
}

// StateEventHandler returns the proper EventHandler for the State event
func StateEventHandler(f func(s State)) EventerHandler {
	return func(payload interface{}) {
		f(payload.(State))
	}
}

func (d *Drone) handleCmds() (err error) {
	// Create raddr
	var raddr *net.UDPAddr
	if raddr, err = net.ResolveUDPAddr("udp", cmdAddr); err != nil {
		err = fmt.Errorf("jtello: creating raddr failed: %w", err)
		return
	}

	// Create laddr
	var laddr *net.UDPAddr
	if laddr, err = net.ResolveUDPAddr("udp", respAddr); err != nil {
		err = fmt.Errorf("jtello: creating laddr failed: %w", err)
		return
	}

	// Dial
	if d.cmdConn, err = net.DialUDP("udp", laddr, raddr); err != nil {
		err = fmt.Errorf("jtello: dialing failed: %w", err)
		return
	}

	// Read responses
	go d.readResponses()

	// Command
	if err = d.command(); err != nil {
		err = fmt.Errorf("jtello: command failed: %w", err)
		return
	}
	return
}

func (d *Drone) readResponses() {
	for {
		// Check context
		if d.ctx.Err() != nil {
			return
		}

		// Read
		b := make([]byte, 2048)
		n, err := d.cmdConn.Read(b)
		if err != nil {
			if d.ctx.Err() == nil {
				fmt.Errorf("jtello: reading response failed: %w", err)
			}
			continue
		}

		// Log
		r := bytes.TrimSpace(b[:n])
		fmt.Printf("jtello: received resp '%s'", r)

		// Signal
		d.rc.L.Lock()
		d.lr = string(r)
		d.rc.Signal()
		d.rc.L.Unlock()
	}
}

type respHandler func(resp string) error

func defaultRespHandler(resp string) (err error) {
	// Check response
	if resp != "ok" {
		err = fmt.Errorf("jtello: invalid response: %w", errors.New(resp))
		return
	}
	return
}

func (d *Drone) respHandlerWithEvent(name string) respHandler {
	return func(resp string) (err error) {
		// Default
		if err = defaultRespHandler(resp); err != nil {
			return
		}

		// Dispatch
		d.e.Dispatch(name, nil)
		return
	}
}

type cmd struct {
	canceller bool
	cmd       string
	h         respHandler
	timeout   time.Duration
}

func (d *Drone) priorityCmd(cmd *cmd) (priority bool) {
	// Lock
	d.mc.Lock()
	defer d.mc.Unlock()

	// Check
	if cmd.canceller {
		priority = true
		for p := range d.cmds {
			if p.canceller {
				priority = false
				break
			}

			// Takeoff and land can't be sent at the same time
			if cmd.cmd == "land" && p.cmd == "takeoff" {
				priority = false
				break
			}
		}
	}
	return
}

func (d *Drone) sendCmd(cmd *cmd) (err error) {
	// No connection
	if d.cmdConn == nil {
		err = ErrNotConnected
		return
	}

	// In most cases we need to wait for the previous cmd to be done. But not when this is a priority cmd.
	// This is a priority cmd if cmd is a canceller and no other canceller is running
	priority := d.priorityCmd(cmd)

	// Add cmd
	d.mc.Lock()
	d.cmds[cmd] = true
	d.mc.Unlock()

	// Make sure to remove cmd
	defer func() {
		d.mc.Lock()
		delete(d.cmds, cmd)
		d.mc.Unlock()
	}()

	// Not a priority cmd
	if !priority {
		// Check context
		if err = d.ctx.Err(); err != nil {
			return
		}

		// Make sure not to send several cmds at the same time
		d.msc.Lock()
		defer d.msc.Unlock()
	}

	// Lock resp
	d.rc.L.Lock()
	defer d.rc.L.Unlock()

	// Log with fmt for
	fmt.Printf("ctrl: sending cmd %s \n", cmd.cmd)

	// Write
	if _, err = d.cmdConn.Write([]byte(cmd.cmd)); err != nil {
		err = fmt.Errorf("jtello: writing failed: %w", err)
		return
	}

	// No handler
	if cmd.h == nil {
		return
	}

	// Create context
	ctx, cancel := context.WithCancel(d.ctx)
	if cmd.timeout > 0 {
		ctx, cancel = context.WithTimeout(d.ctx, cmd.timeout)
	}
	defer cancel()

	// Handle context
	go func() {
		// Wait for context to be done
		<-ctx.Done()

		// Check error
		if d.ctx.Err() != context.Canceled && ctx.Err() != context.DeadlineExceeded {
			return
		}

		// Signal
		d.rc.L.Lock()
		d.rc.Signal()
		d.rc.L.Unlock()
	}()

	// Wait for response
	d.rc.Wait()

	// Check context
	if ctx.Err() != nil {
		err = ctx.Err()
		return
	}

	// Custom
	if err = cmd.h(d.lr); err != nil {
		err = fmt.Errorf("jtello: custom handler failed: %w", err)
		return
	}
	return
}

func (d *Drone) command() (err error) {
	// Send "command" cmd
	if err = d.sendCmd(&cmd{
		cmd:     "command",
		h:       defaultRespHandler,
		timeout: defaultTimeout,
	}); err != nil {
		err = fmt.Errorf("jtello: sending 'command' cmd failed: %w", err)
		return
	}
	return
}

/** COMMANDS LIST **/
// Emergency makes Tello stop all motors immediately
// This cmd doesn't seem to be receiving any response, that's why we don't provide any handler
func (d *Drone) Emergency() (err error) {
	// Send cmd
	if err = d.sendCmd(&cmd{
		canceller: true,
		cmd:       "emergency",
		timeout:   defaultTimeout,
	}); err != nil {
		err = fmt.Errorf("jtello: sending emergency cmd failed: %w", err)
		return
	}
	return
}

// TakeOff makes Tello auto takeoff
func (d *Drone) TakeOff() (err error) {
	// Send cmd
	if err = d.sendCmd(&cmd{
		cmd:     "takeoff",
		h:       d.respHandlerWithEvent(TakeOffEvent),
		timeout: 20 * time.Second,
	}); err != nil {
		err = fmt.Errorf("jtello: sending takeoff cmd failed: %w", err)
		return
	}
	return
}

// Land makes Tello auto land
func (d *Drone) Land() (err error) {
	// Send cmd
	if err = d.sendCmd(&cmd{
		canceller: true,
		cmd:       "land",
		h:         d.respHandlerWithEvent(LandEvent),
		timeout:   20 * time.Second,
	}); err != nil {
		err = fmt.Errorf("jtello: sending land cmd failed: %w", err)
		return
	}
	return
}

// Up makes Tello fly up with distance x cm
func (d *Drone) Up(x int) (err error) {
	// Send cmd
	if err = d.sendCmd(&cmd{
		cmd:     fmt.Sprintf("up %d", x),
		h:       defaultRespHandler,
		timeout: time.Minute,
	}); err != nil {
		err = fmt.Errorf("jtello: sending up cmd failed: %w", err)
		return
	}
	return
}

// Down makes Tello fly down with distance x cm
func (d *Drone) Down(x int) (err error) {
	// Send cmd
	if err = d.sendCmd(&cmd{
		cmd:     fmt.Sprintf("down %d", x),
		h:       defaultRespHandler,
		timeout: time.Minute,
	}); err != nil {
		err = fmt.Errorf("jtello: sending down cmd failed: %w", err)
		return
	}
	return
}

// Left makes Tello fly left with distance x cm
func (d *Drone) Left(x int) (err error) {
	// Send cmd
	if err = d.sendCmd(&cmd{
		cmd:     fmt.Sprintf("left %d", x),
		h:       defaultRespHandler,
		timeout: time.Minute,
	}); err != nil {
		err = fmt.Errorf("jtello: sending left cmd failed: %w", err)
		return
	}
	return
}

// Right makes Tello fly right with distance x cm
func (d *Drone) Right(x int) (err error) {
	// Send cmd
	if err = d.sendCmd(&cmd{
		cmd:     fmt.Sprintf("right %d", x),
		h:       defaultRespHandler,
		timeout: time.Minute,
	}); err != nil {
		err = fmt.Errorf("jtello: sending right cmd failed: %w", err)
		return
	}
	return
}

// Forward makes Tello fly forward with distance x cm
func (d *Drone) Forward(x int) (err error) {
	// Send cmd
	if err = d.sendCmd(&cmd{
		cmd:     fmt.Sprintf("forward %d", x),
		h:       defaultRespHandler,
		timeout: time.Minute,
	}); err != nil {
		err = fmt.Errorf("jtello: sending forward cmd failed: %w", err)
		return
	}
	return
}

// Back makes Tello fly back with distance x cm
func (d *Drone) Back(x int) (err error) {
	// Send cmd
	if err = d.sendCmd(&cmd{
		cmd:     fmt.Sprintf("back %d", x),
		h:       defaultRespHandler,
		timeout: time.Minute,
	}); err != nil {
		err = fmt.Errorf("jtello: sending back cmd failed: %w", err)
		return
	}
	return
}

// RotateClockwise makes Tello rotate x degree clockwise
func (d *Drone) RotateClockwise(x int) (err error) {
	// Send cmd
	if err = d.sendCmd(&cmd{
		cmd:     fmt.Sprintf("cw %d", x),
		h:       defaultRespHandler,
		timeout: time.Minute,
	}); err != nil {
		err = fmt.Errorf("jtello: sending cw cmd failed: %w", err)
		return
	}
	return
}

// RotateCounterClockwise makes Tello rotate x degree counter-clockwise
func (d *Drone) RotateCounterClockwise(x int) (err error) {
	// Send cmd
	if err = d.sendCmd(&cmd{
		cmd:     fmt.Sprintf("ccw %d", x),
		h:       defaultRespHandler,
		timeout: time.Minute,
	}); err != nil {
		err = fmt.Errorf("jtello: sending ccw cmd failed: %w", err)
		return
	}
	return
}

// Flip makes Tello flip in the specified direction
// Check out Flip... constants for available flip directions
func (d *Drone) Flip(x string) (err error) {
	// Send cmd
	if err = d.sendCmd(&cmd{
		cmd:     fmt.Sprintf("flip %s", x),
		h:       defaultRespHandler,
		timeout: 20 * time.Second,
	}); err != nil {
		err = fmt.Errorf("jtello: sending flip cmd failed: %w", err)
		return
	}
	return
}

// Go makes Tello fly to x y z in speed (cm/s)
func (d *Drone) GoMissionPad(x, y, z, speed int, pad int) (err error) {	
	if err = d.sendCmd(&cmd{
		cmd:     fmt.Sprintf("go %d %d %d %d m%d", x, y, z, speed, pad),
		h:       defaultRespHandler,
		timeout: time.Minute,
	}); err != nil {
		err = fmt.Errorf("jtello: sending go mission cmd failed: %w", err)
		return
	}
	return
}

// Go makes Tello fly to x y z in speed (cm/s)
func (d *Drone) Go(x, y, z, speed int) (err error) {
	// Send cmd
	if err = d.sendCmd(&cmd{
		cmd:     fmt.Sprintf("go %d %d %d %d", x, y, z, speed),
		h:       defaultRespHandler,
		timeout: time.Minute,
	}); err != nil {
		err = fmt.Errorf("jtello: sending go cmd failed: %w", err)
		return
	}
	return
}

// Curve makes Tello fly a curve defined by the current and two given coordinates with speed (cm/s)
func (d *Drone) Curve(x1, y1, z1, x2, y2, z2, speed int) (err error) {
	// Send cmd
	if err = d.sendCmd(&cmd{
		cmd:     fmt.Sprintf("curve %d %d %d %d %d %d %d", x1, y1, z1, x2, y2, z2, speed),
		h:       defaultRespHandler,
		timeout: time.Minute,
	}); err != nil {
		err = fmt.Errorf("jtello: sending go cmd failed: %w", err)
		return
	}
	return
}

// SetSticks sends RC control via four channels
// All values are between -100 and 100
// lr: left/right
// fb: forward/backward
// ud: up/down
// y: yawn
// This cmd doesn't seem to be receiving any response, that's why we don't provide any handler
func (d *Drone) SetSticks(lr, fb, ud, y int) (err error) {
	// Send cmd
	if err = d.sendCmd(&cmd{
		cmd:     fmt.Sprintf("rc %d %d %d %d", lr, fb, ud, y),
		timeout: defaultTimeout,
	}); err != nil {
		err = fmt.Errorf("jtello: sending rc cmd failed: %w", err)
		return
	}
	return
}

// SetWifi sets Wi-Fi with SSID password
// I couldn't make this work (it returned 'error' even though the SSID was changed but the password was not)
// If anyone manages to make it work, create an issue in github, I'm really interested in how you managed that :D
// OK, lets stry! :D
func (d *Drone) SetWifi(ssid, password string) (err error) {
	// Send cmd
	if err = d.sendCmd(&cmd{
		cmd:     fmt.Sprintf("wifi %s %s", ssid, password),
		h:       defaultRespHandler,
		timeout: defaultTimeout,
	}); err != nil {
		err = fmt.Errorf("jtello: sending wifi cmd failed: %w", err)
		return
	}
	return
}

// Wifi returns the Wifi SNR
func (d *Drone) Wifi() (snr int, err error) {
	// Send cmd
	// It returns "100.0"
	if err = d.sendCmd(&cmd{
		cmd: "wifi?",
		h: func(resp string) (err error) {
			// Parse
			if snr, err = strconv.Atoi(resp); err != nil {
				err = fmt.Errorf("jtello: atoi %s failed: %w", resp, err)
				return
			}
			return
		},
		timeout: defaultTimeout,
	}); err != nil {
		err = fmt.Errorf("jtello: sending wifi? cmd failed: %w", err)
		return
	}
	return
}

// SetSpeed sets speed to x cm/s
func (d *Drone) SetSpeed(x int) (err error) {
	// Send cmd
	if err = d.sendCmd(&cmd{
		cmd:     fmt.Sprintf("speed %d", x),
		h:       defaultRespHandler,
		timeout: defaultTimeout,
	}); err != nil {
		err = fmt.Errorf("jtello: sending speed cmd failed: %w", err)
		return
	}
	return
}

// Speed returns the current speed (cm/s)
func (d *Drone) Speed() (x int, err error) {
	// Send cmd
	// It returns "100.0"
	if err = d.sendCmd(&cmd{
		cmd: "speed?",
		h: func(resp string) (err error) {
			// Parse
			var f float64
			if f, err = strconv.ParseFloat(resp, 64); err != nil {
				err = fmt.Errorf("jtello: parsing float %s failed: %w", resp, err)
				return
			}

			// Set speed
			x = int(f)
			return
		},
		timeout: defaultTimeout,
	}); err != nil {
		err = fmt.Errorf("jtello: sending speed? cmd failed: %w", err)
		return
	}
	return
}

// Mission Mode OFF (USE ILS / VP) ia Mission Pads (tello edu)
func (d *Drone) MissionModeOn() (err error) { 
	// Send cmd
	if err = d.sendCmd(&cmd{
		cmd:     "mon",
		h:       defaultRespHandler, // mission handle?
		timeout: 4 * time.Second,
	}); err != nil {
		err = fmt.Errorf("jtello: sending mON cmd failed: %w", err)
		return
	}
	return
}

// Mission Mode OFF (USE ILS / VP) via Mission Pads (tello edu)
func (d *Drone) MissionModeOff() (err error) {
	// Send cmd
	if err = d.sendCmd(&cmd{
		cmd:     "moff",
		h:       defaultRespHandler, // mission handler?
		timeout: 4 * time.Second,
	}); err != nil {
		err = fmt.Errorf("jtello: sending mOFF cmd failed: %w", err)
		return
	}
	return
}
