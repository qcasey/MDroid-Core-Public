// Package pybus interfaces between MDroid-Core and the pyBus programs
package pybus

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/qcasey/MDroid-Core/format/response"
	"github.com/gorilla/mux"
	"github.com/qcasey/MDroid-Core/format"
	"github.com/qcasey/MDroid-Core/mserial"
	"github.com/qcasey/MDroid-Core/sessions"
	"github.com/qcasey/MDroid-Core/settings"
	"github.com/rs/zerolog/log"
)

// PushQueue adds a directive to the pybus queue
// msg can either be a directive (e.g. 'openTrunk')
// or a Python formatted list of three byte strings: src, dest, and data
// e.g. '["50", "68", "3B01"]'
func PushQueue(command string) {

	//
	// First, interrupt with some special cases
	//
	switch command {
	case "rollWindowsUp":
		go PushQueue("popWindowsUp")
		go PushQueue("popWindowsUp")
		return
	case "rollWindowsDown":
		go PushQueue("popWindowsDown")
		go PushQueue("popWindowsDown")
		return
	}

	// Send request to pybus server
	resp, err := http.Get(fmt.Sprintf("http://localhost:8080/%s", command))
	if err != nil {
		log.Error().Msgf("Failed to request %s from pybus: \n %s", command, err.Error())
		return
	}
	defer resp.Body.Close()

	log.Debug().Msgf("Added %s to the Pybus Queue", command)
}

// StartRoutine handles incoming requests to the pybus program, will add routines to the queue
func StartRoutine(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)

	src, srcOK := params["src"]
	dest, destOK := params["dest"]
	data, dataOK := params["data"]

	if srcOK && destOK && dataOK && len(src) == 2 && len(dest) == 2 && len(data) > 0 {
		go PushQueue(fmt.Sprintf(`["%s", "%s", "%s"]`, src, dest, data))
	} else if params["command"] != "" {
		// Some commands need special timing functions
		go PushQueue(params["command"])
	} else {
		response.WriteNew(&w, r, response.JSONResponse{Output: "Invalid command", OK: false})
		return
	}
	response.WriteNew(&w, r, response.JSONResponse{Output: "OK", OK: true})
}

// repeatCommand endlessly, helps with request functions
func repeatCommand(command string, sleepSeconds int) {
	log.Info().Msgf("Running Pybus command %s every %d seconds", command, sleepSeconds)
	for {
		// Only push repeated pybus commands when powered, otherwise the car won't sleep
		if sessions.GetBoolDefault("KEY_DETECTED", false) {
			PushQueue(command)
		}
		time.Sleep(time.Duration(sleepSeconds) * time.Second)
	}
}

func waitUntilOnline() {
	log.Info().Msg("Waiting for pybus to come online...")
	for {
		if _, err := http.Get("http://localhost:8080/requestIgnitionStatus"); err == nil {
			break
		}
		time.Sleep(time.Millisecond * 100)
	}
}

// ParseCommand is a list of pre-approved routes to PyBus for easier routing
// These GET requests can be used instead of knowing the implementation function in pybus
// and are actually preferred, since we can handle strange cases
func ParseCommand(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)

	if len(params["device"]) == 0 || len(params["command"]) == 0 {
		response.WriteNew(&w, r, response.JSONResponse{Output: "Error: One or more required params is empty", OK: false})
		return
	}

	// Format similarly to the rest of MDroid suite, removing plurals
	// Formatting allows for fuzzier requests
	device := strings.TrimSuffix(format.Name(params["device"]), "S")
	command := strings.TrimSuffix(format.Name(params["command"]), "S")

	// Parse command into a bool, make either "on" or "off" effectively
	isPositive, err := format.IsPositiveRequest(command)
	isPosErr := err != nil

	log.Info().Msgf("Attempting to send command %s to device %s", command, device)

	// If the car's ACC power isn't on, it won't be ready for requests. Wake it up first
	if !sessions.GetBoolDefault("ACC_POWER", false) {
		PushQueue("requestVehicleStatus") // this will be swallowed
	}

	// All I wanted was a moment or two to
	// See if you could do that switch-a-roo
	switch device {
	case "DOOR":
		if isPosErr {
			log.Error().Msg(err.Error())
			return
		}
		doorStatus, _ := sessions.Get("DOORS_LOCKED")
		if mserial.Writer != nil && isPositive && doorStatus.Value == "FALSE" ||
			mserial.Writer != nil && !isPositive && doorStatus.Value == "TRUE" {
			mserial.PushText("toggleDoorLocks")
		} else {
			log.Info().Msgf("Request to %s doors denied, door status is %s", command, doorStatus.Value)
		}
	case "WINDOW":
		if command == "POPDOWN" {
			PushQueue("popWindowsDown")
		} else if command == "POPUP" {
			PushQueue("popWindowsUp")
		} else if isPositive {
			PushQueue("rollWindowsUp")
		} else {
			PushQueue("rollWindowsDown")
		}
	case "TOP", "CONVERTIBLE_TOP":
		if isPosErr {
			log.Error().Msg(err.Error())
			return
		}
		if isPositive {
			PushQueue("convertibleTopUp")
		} else {
			PushQueue("convertibleTopDown")
		}
	case "TRUNK":
		PushQueue("openTrunk")
	case "HAZARD":
		if isPosErr {
			log.Error().Msg(err.Error())
			return
		}
		if isPositive {
			PushQueue("turnOnHazards")
		} else {
			PushQueue("turnOffAllExteriorLights")
		}
	case "FLASHER":
		if isPosErr {
			log.Error().Msg(err.Error())
			return
		}
		if isPositive {
			PushQueue("flashAllExteriorLights")
		} else {
			PushQueue("turnOffAllExteriorLights")
		}
	case "INTERIOR":
		if isPosErr {
			log.Error().Msg(err.Error())
			return
		}
		if isPositive {
			PushQueue("interiorLightsOff")
		} else {
			PushQueue("interiorLightsOn")
		}
	case "CLOWN", "NOSE":
		PushQueue("turnOnClownNose")
	case "MODE":
		PushQueue("pressMode")
	case "RADIO", "NAV", "STEREO":
		switch command {
		case "AM":
			PushQueue("pressAM")
		case "FM":
			PushQueue("pressFM")
		case "NEXT":
			PushQueue("pressNext")
		case "PREV":
			PushQueue("pressPrev")
		case "MODE":
			PushQueue("pressMode")
		case "NUM":
			PushQueue("pressNumPad")
		case "1":
			PushQueue("press1")
		case "2":
			PushQueue("press2")
		case "3":
			PushQueue("press3")
		case "4":
			PushQueue("press4")
		case "5":
			PushQueue("press5")
		case "6":
			PushQueue("press6")
		default:
			PushQueue("pressStereoPower")
		}
	case "AZMODAN", "CAMERA", "BOARD":
		if format.Name(command) == "AUTO" {
			settings.Set("BOARD", "POWER", "AUTO")
		} else if isPositive && !isPosErr {
			settings.Set("BOARD", "POWER", "ON")
			mserial.PushText("powerOnBoard")
		} else if !isPosErr {
			settings.Set("BOARD", "POWER", "OFF")
			mserial.PushText("powerOffBoard")
		}
	case "BRIGHTWING", "LTE":
		if format.Name(command) == "AUTO" {
			settings.Set("WIRELESS", "POWER", "AUTO")
		} else if isPositive && !isPosErr {
			settings.Set("WIRELESS", "POWER", "ON")
			mserial.PushText("powerOnWireless")
		} else if !isPosErr {
			settings.Set("WIRELESS", "POWER", "OFF")
			mserial.PushText("powerOffWireless")
		}
	default:
		log.Error().Msgf("Invalid device %s", device)
		response := response.JSONResponse{Output: fmt.Sprintf("Invalid device %s", device), OK: false}
		response.Write(&w, r)
		return
	}

	// Yay
	response.WriteNew(&w, r, response.JSONResponse{Output: device, OK: true})
}
