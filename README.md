# Tello EDU - Drone Golang Starter Template

Ryze TelloEDU Drone Golang Starter Template
For anyone looking to code some Golang, and have some fun with the Tello EDU.

* Requires a working knowledge of golang / gopath / go modules.
There are many great examples out there, why not build a [BattleSnake](https://docs.battlesnake.com/guides/getting-started) first?

## Disclaimer

Tello is a registered trademark of Ryze Tech.  The author of this package is in no way affiliated with Ryze, DJI, or Intel.  

The package has been developed by gathering together information from a variety of sources on the Internet;

Use this package at your own risk!!!  The author(s) is/are in no way responsible for any damage caused either to or by the drone when using this software.
NEVER FLY NEAR PEOPLE, ANIMALS OR ANY OTHER OBJECTS. Use a clear space! Watch your head ;-)

Big shout out to [SMerrony](https://godoc.org/github.com/SMerrony)! for (https://godoc.org/github.com/SMerrony/tello)[tello] & (https://godoc.org/github.com/SMerrony/telloterm)[telloterm]!!!

See [Tello Lib ImplementationChart.md](https://github.com/SMerrony/tello/blob/master/ImplementationChart.md) for full details of what functions are currently implemented.

## What do you need?

- Tello / Tello EDU drone
- PC with Wifi connection
- 2m<sup>3</sup> / 16ft<sup>3</sup/> of empty space for fying.

Optional (but fun)
- Joystick 
- PS4 Controller (DualShock 4)
- Mission Pads (*only works with for Tello EDU)


### Tello term

#### Install
```
go get github.com/SMerrony/telloterm
```
If you wish to use the video window you must have mplayer installed and on your PATH.

#### Run

```
$ telloterm -h
Usage of C:\Users\jstolp\go\bin\telloterm.exe:
  -cpuprofile file
        Write cpu profile to file
  -fdlog string
        Log some CSV flight data to this file
  -joyhelp

    TelloTerm Joystick Control Mapping

    Right Stick  Forward/Backward/Left/Right
    Left Stick   Up/Down/Turn
    Triangle     Takeoff
    X            Land
    Circle
    Square       Take Photo
    L1           Bounce (on/off)
    L2           Palm Land



  -jsid int
        ID number of joystick to use (see -jslist to get IDs) (default 999)
$ telloterm -jsid 1

  -jslist
        List attached joysticks

Joystick ID: 0: Name: Microsoft PC-joystick driver, Axes: 5, Buttons: 4 (TrustMaster joystick)
Joystick ID: 1: Name: Microsoft PC-joystick driver, Axes: 8, Buttons: 14 (DualShock PS4 controller)

  -jstest  -         Debug joystick mapping
  -jstype (DualShock4||HotasX) select joystick type
  -keyhelp
        TelloTerm Keyboard Control Mapping

        <Cursor Keys> Move Left/Right/Forward/Backward
        w|a|s|d       W: Up, S: Down, A: Turn Left, D: Turn Right
        <SPACE>       Hover (stop all movement)
        <HOME>        Set Home position or fly to Home position
        b             Bounce (toggle)
        t             Takeoff
        o             Throw Takeoff
        l             Land
        p             Palm Land
        0             360 degree smart video flight
        1|2|3|4       Flip Fwd/Back/Left/Right
        f             Take Picture (Foto)
        q/<Escape>    Quit
        r/<Ctrl-L>        Refresh Screen
        v             Start Video (mplayer) Window
        -             Slow (normal) flight mode
        +             Fast (sports) flight mode
        =             Switch between normal and wide video mode


  -x11
        Use '-vo x11' flag in case mplayer takes over entire window

```