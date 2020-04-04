package main

import (
	"fmt"
	"strconv"

	//bluetooth "github.com/qcasey/MDroid-Bluetooth"
	"github.com/qcasey/MDroid-Core-Public/format"
	"github.com/qcasey/MDroid-Core-Public/sessions"
	"github.com/qcasey/MDroid-Core-Public/sessions/gps"
	"github.com/rs/zerolog/log"

	"github.com/qcasey/MDroid-Core-Public/settings"
)

func setupHooks() {
	settings.RegisterHook("AUTO_SLEEP", autoSleepSettings)
	settings.RegisterHook("AUTO_LOCK", autoLockSettings)
	settings.RegisterHook("ANGEL_EYES", angelEyesSettings)
	sessions.RegisterHookSlice(&[]string{"MAIN_VOLTAGE_RAW", "AUX_VOLTAGE_RAW"}, voltage)
	sessions.RegisterHook("AUX_CURRENT_RAW", auxCurrent)
	sessions.RegisterHook("ACC_POWER", accPower)
	sessions.RegisterHook("KEY_STATE", keyState)
	sessions.RegisterHook("LIGHT_SENSOR_REASON", lightSensorReason)
	sessions.RegisterHook("LIGHT_SENSOR_ON", lightSensorOn)
	sessions.RegisterHookSlice(&[]string{"SEAT_MEMORY_1", "SEAT_MEMORY_2", "SEAT_MEMORY_3"}, voltage)
	log.Info().Msg("Enabled session hooks")
}

//
// From here on out are the hook functions.
// We're taking actions based on the values or a combination of values
// from the session/settings post values.
//

// When angel eyes setting is changed
func angelEyesSettings(settingName string, settingValue string) {
	// Determine state of angel eyes
	evalAngelEyesPower(sessions.GetStringDefault("KEY_STATE", "FALSE"))
}

// When auto lock setting is changed
func autoLockSettings(settingName string, settingValue string) {
	accOn := sessions.GetBoolDefault("ACC_POWER", false)
	wifiOn := sessions.GetBoolDefault("WIFI_CONNECTED", true)

	// Determine state of auto lock
	evalAutoLock(sessions.GetStringDefault("KEY_STATE", "FALSE"), accOn, wifiOn)
}

// When auto Sleep setting is changed
func autoSleepSettings(settingName string, settingValue string) {
	accOn := sessions.GetBoolDefault("ACC_POWER", false)
	wifiOn := sessions.GetBoolDefault("WIFI_CONNECTED", true)

	// Determine state of auto Sleep
	evalAutoSleep(sessions.GetStringDefault("KEY_STATE", "FALSE"), accOn, wifiOn)
}

// When key state is changed in session
func keyState(hook *sessions.Data) {
	accOn := sessions.GetBoolDefault("ACC_POWER", false)
	wifiOn := sessions.GetBoolDefault("WIFI_CONNECTED", true)

	// Play / pause bluetooth media on key in/out
	/*
	if hook.Value != "FALSE" {
		go bluetooth.Play()
	} else {
		go bluetooth.Pause()
	}*/

	// Determine state of angel eyes, and main board
	evalAngelEyesPower(hook.Value)
	evalVideoPower(hook.Value, accOn, wifiOn)
	evalAutoLock(hook.Value, accOn, wifiOn)
}

// When light sensor is changed in session
func lightSensorOn(hook *sessions.Data) {
	// Determine state of angel eyes
	evalAngelEyesPower(sessions.GetStringDefault("KEY_STATE", "FALSE"))
}

// Convert main raw voltage into an actual number
func voltage(hook *sessions.Data) {
	voltageFloat, err := strconv.ParseFloat(hook.Value, 64)
	if err != nil {
		log.Error().Msgf("Failed to convert string %s to float", hook.Value)
		return
	}

	//sessions.SetValue(hook.Name[0:len(hook.Name)-4], fmt.Sprintf("%.3f", 0.15+(voltageFloat/1024)*24.4))
	sessions.SetValue(hook.Name[0:len(hook.Name)-4], fmt.Sprintf("%.3f", (voltageFloat/1024)*16.5))
}

// Modifiers to the incoming Current sensor value
func auxCurrent(hook *sessions.Data) {
	currentFloat, err := strconv.ParseFloat(hook.Value, 32)
	if err != nil {
		log.Error().Msgf("Failed to convert string %s to float", hook.Value)
		return
	}

	modifier := .06
	if currentFloat < .3 {
		modifier = .09
	} else if currentFloat > 1.5 {
		modifier = .3
	}

	realCurrent := currentFloat + modifier
	sessions.SetValue("AUX_CURRENT", fmt.Sprintf("%.3f", realCurrent))
}

// Trigger for booting boards/tablets
func accPower(hook *sessions.Data) {
	var accOn bool

	// Check incoming ACC power value is valid
	switch hook.Value {
	case "TRUE":
		accOn = true
	case "FALSE":
		accOn = false
	default:
		log.Error().Msgf("ACC Power Trigger unexpected value: %s", hook.Value)
		return
	}

	// Pull the necessary configuration data
	wifiOn := sessions.GetBoolDefault("WIFI_CONNECTED", true)
	keyIsIn := sessions.GetStringDefault("KEY_STATE", "FALSE")

	// Trigger video, and tablet based on ACC and wifi status
	go evalVideoPower(keyIsIn, accOn, wifiOn)
	go evalTabletPower(keyIsIn, accOn, wifiOn)
	go evalAutoLock(keyIsIn, accOn, wifiOn)
	go evalAutoSleep(keyIsIn, accOn, wifiOn)
}

// Alert me when it's raining and windows are down
func lightSensorReason(hook *sessions.Data) {
	keyPosition, kerr := sessions.Get("KEY_POSITION")
	doorsLocked, derr := sessions.Get("DOORS_LOCKED")
	windowsOpen, werr := sessions.Get("WINDOWS_OPEN")

	// Check if any of the above aren't set yet
	if kerr != nil || derr != nil || werr != nil {
		return
	}

	delta, err := format.CompareTimeToNow(doorsLocked.LastUpdate, gps.GetTimezone())
	if err != nil {
		log.Error().Msg(err.Error())
		return
	}

	if hook.Value == "RAIN" &&
		keyPosition.Value == "OFF" &&
		doorsLocked.Value == "TRUE" &&
		windowsOpen.Value == "TRUE" &&
		delta.Minutes() > 5 {
		sessions.SlackAlert("Windows are down in the rain, eh?")
	}
}

// Restart different machines when seat memory buttons are pressed
func seatMemory(hook *sessions.Data) {
	switch hook.Name {
	case "SEAT_MEMORY_1":
		sendServiceCommand("BOARD", "restart")
	case "SEAT_MEMORY_2":
		sendServiceCommand("WIRELESS", "restart")
	case "SEAT_MEMORY_3":
		sendServiceCommand("MDROID", "restart")
	}
}
