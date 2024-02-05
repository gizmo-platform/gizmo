# BEST Robotics Gizmo

Welcome, lets drive some robots!  The Gizmo is an open source and open
hardware platform developed in cooperation with the team behind [BEST
Robotics](https://bestrobotics.org) which you can use to build and
control robots.  The system is composed of multiple parts including a
[hardware device](https://github.com/bestrobotics/gizmo-hw) that
contains the control components, the Gizmo utilities (this repo), the
system software that runs on the hardware
[gizmo-fw](https://github.com/bestrobotics/gizmo-fw), and an [Arduino
library](https://github.com/bestrobotics/ArduinoGizmo) that lets you
build custom code to control your robot.

## Setup

First, you will need to install the dependencies on at least one
computer required to program the Raspbery Pi Pico processors that are
on the Gizmo.  To program them, install the [Arduino
environment](https://www.arduino.cc/en/software) for your platform,
and optionally the
[arduino-cli](https://github.com/arduino/arduino-cli/releases/tag/v0.35.2)
to enable the Gizmo utility to perform certain tasks for you
automatically.  If you're on Windows, you'll also need to install
[Python](https://www.python.org/downloads/windows/) and add it to your
`PATH` variable (the python installer will do this for you).

If you're on Linux, make sure you wind up with arduino-cli in your
`PATH`

Once you've installed the dependencies as described above, you can
install the Gizmo software itself from this repository.  If you are
not planning to develop the Gizmo software, grab a compiled version
from the Releases page linked in the right hand sidebar.  Grab the one
appropriate for your operating system and architecture.  If you intend
to develop the Gizmo software, fork and clone this repository
somewhere convenient.  You'll also need to clone the gizmo-fw
repository since the compatible firmware builds get "baked" into the
gizmo CLI.  The Gizmo tools can be compiled with any modern version of
Go.

## Installing the Gizmo System Software

Before you can use the Gizmo board, you need to install the system
software.  This is installed to the Raspberry Pi Pico on the left side
of the board when held with the USB ports pointing away from you.
This process involves generating a firmware image that you can install
onto the processor using a USB micro cable.  Begin by opening your
terminal of choice.  If using Windows, you'll need to be in the same
directory where you downloaded the Gizmo software file, and its
recommended to rename the `exe` file to just `gizmo.exe` so you don't
have to type the version number every time.

Start by running `gizmo firmware configure` and answering the
questions.  Your output will look similar to this:

```
$ gizmo firmware configure
? Address of the field server 192.168.16.10
? Network SSID RoboNet
? Network PSK (Input will be obscured) ********
? Team Number 1234
```

The Network SSID and PSK should be the network you want the robot to
connect to, and the IP address should be the IP you intend to have the
gizmo software running on to drive your robot.  By default the Gizmo
utility assumes this is the computer that you're currently using, and
fills in the right address as a default which you can accept by just
pressing enter on that prompt.

The survey questions will create a file called `gsscfg.json` in the
current directory.  Note that the IP address you give the wizard
should be stable, and static either via a fully static configuration
or static via a DHCP reservation.  Configuring a static address is
beyond the scope of this repo, but either your network administrator
or Google can help you with this.

### Automatic Build

Once the configuration has been created, the firmware image can be
compiled.  Run the following command in the same terminal window:

```
$ gizmo firmware build
2024-02-05T00:09:40.113-0600 [INFO]  field: Log level: level=info
2024-02-05T00:09:40.113-0600 [INFO]  field.factory: Performing Step: team=1234 step=Unpack
2024-02-05T00:09:40.113-0600 [INFO]  field.factory: Performing Step: team=1234 step=Configure
2024-02-05T00:09:40.113-0600 [INFO]  field.factory: Performing Step: team=1234 step=Compile
2024-02-05T00:10:00.474-0600 [INFO]  field.factory: Performing Step: team=1234 step=Export
2024-02-05T00:10:00.475-0600 [INFO]  field.factory: Performing Step: team=1234 step=Cleanup
```

Note that the firmware build can take several minutes depending on the
speed of your computer.

Assuming you don't get any errors, a file will be created in the
current directory called `gss_1234.uf2` where `1234` is the team
number provided to the configuration step previously.  Hold down the
white button on the left hand Raspberry Pi Pico (the "system"
processor) and plug in the USB cable.  Your computer will recognize
the device as a flash drive and mount it as removable storage.  Copy
the `uf2` file to this drive and it will eject itself.  If all has
gone to plan, you should see the green LED blink slowly a few times,
then very rapidly (possibly so rapidly you can't easily see it blink).
You're now ready to run the gizmo field software (see below) and
develop your own user code specific to your robot!

### Manual Build

Using this automatic build process depends on having the arduino-cli
installed and available.  If you do not wish to use this automated
process, you can still build the firmware manually by using this
command instead:

```
$ gizmo firmware build --extract-only --directory firmware
2024-02-05T00:21:05.485-0600 [INFO]  field: Log level: level=info
2024-02-05T00:21:05.485-0600 [INFO]  field.factory: Performing Step: team=1234 step=Unpack
2024-02-05T00:21:05.485-0600 [INFO]  field.factory: Performing Step: team=1234 step=Configure
```

This will create a directory called "firmware" that you can then open
in the Arduino GUI.  Compile and upload this project to the system
processor to complete installation of the Gizmo system software.

## Running a Field

The Gizmo software provides two options for running the field
management system (FMS).  The full featured version runs a multiple
field setup suitable for competitions, whereas the practice system
runs a field with only one color configured for one robot to drive
around.

### Running a Practice Mode Field

To run a practice mode field, just run `gizmo field practice <number>`
where `<number` is your team number.  In the above example for team
1234, the command will look like this:

```
$ gizmo field practice 1234
2024-02-05T00:26:14.772-0600 [INFO]  field: Log level: level=info
2024-02-05T00:26:14.824-0600 [INFO]  field.gamepad-controller: Successfully bound controller: fid=field1:practice jsid=0
2024-02-05T00:26:14.825-0600 [INFO]  field.web: HTTP is starting
2024-02-05T00:26:14.825-0600 [INFO]  field.mqtt: MQTT is starting
2024-02-05T00:26:14.826-0600 [INFO]  field.pusher: Connected to broker
2024-02-05T00:26:14.826-0600 [INFO]  field.pusher: Subscribed to topics
2024-02-05T00:26:15.825-0600 [INFO]  field.metrics: Connected to broker
2024-02-05T00:26:15.825-0600 [INFO]  field.tlm: Connected to broker
2024-02-05T00:26:15.826-0600 [INFO]  field.metrics: Subscribed to topics
2024-02-05T00:26:15.826-0600 [INFO]  field: Startup Complete!
```

Once you see `Startup Complete!` the FMS is configured and ready to
use.  If your robot is powered on, the 3 RGB LEDs at the bottom will
cycle through some colors.  RGB0 will turn green when your network
connection is established, RGB1 will turn white and blink when you are
successfully connected to the practice field, and RGB2 will change
colors based on how charged your battery is (green/yellow/red).

When you're done using the practice server, press `Control + c` to
terminate the server process.

### Running a Full Competition Field

Running a full competition field is slightly more complicated.  You'll
need all 4 gamepads plugged into the same computer, usually with USB
extension cables.  An additional step is involved where you configure
your field server using `gizmo field wizard`.  The configuration
process looks like this:

```
$ gizmo field wizard
? Address of the field server 192.168.16.10
? Select the number of fields present 1
? All gamepads are connected directly via USB Yes
Your event is configured as follows

You have 1 field(s)
All gamepads are connected directly
The server's IP is 192.168.16.10
? Does everything above look right? Yes
```

This will create a file in the same directory called `config.yml`.
After running the wizard, you can start the field server with the
command `gizmo field serve`.  This will have output similar to the
practice mode above, but will show all 4 gamepads being initialized
and bound.  Since the field server is a long lived program that needs
to be reconfigured between matches, the gizmo utility includes a
command to do this `gizmo field remap`.  Its also possible to have the
FMS follow a schedule in an external system, but that's beyond the
scope of this README.  Running `gizmo field remap` will prompt you
what teams you want to put on each field.  To leave a field empty,
enter a `-` instead of a number.  If you want to remap a field
quadrant without cahnging the team that's on it, press enter to accept
the team that's already there.  The command will look like this:

```
$ gizmo field remap
Enter new mapping
? field1:red 1234
? field1:blue -
? field1:green -
? field1:yellow -
```

The server will print out a line to its log to note that the field was
reconfigured and teams were moved.  That line looks like this:

```
2024-02-05T00:40:09.123-0600 [INFO]  field.web: Immediately remapped teams!: map=map[1234:field1:red]
```

Note that field remapping is instantaneous, and should not be done in
the middle of a match due to the possibility of disconnecting a team
on the field.
