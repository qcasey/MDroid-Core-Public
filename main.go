package main

import (
	"github.com/gorilla/mux"
	//bluetooth "github.com/qcasey/MDroid-Bluetooth"
	"github.com/qcasey/MDroid-Core-Public/db"
	"github.com/qcasey/MDroid-Core-Public/mqtt"
	"github.com/qcasey/MDroid-Core-Public/mserial"
	"github.com/qcasey/MDroid-Core-Public/pybus"
	"github.com/qcasey/MDroid-Core-Public/sessions/gps"
	"github.com/qcasey/MDroid-Core-Public/sessions/system"
)

func main() {
	// Run through the config file and retrieve some settings
	configMap := parseConfig()

	// Init router
	router := mux.NewRouter()

	gps.Mod.Setup(configMap)
	gps.Mod.SetRoutes(router)
	system.Mod.Setup(configMap)
	system.Mod.SetRoutes(router)

	// Set default routes (including session)
	SetDefaultRoutes(router)

	// Setup conventional modules
	// TODO: More modular handling of modules
	mserial.Mod.Setup(configMap)
	mserial.Mod.SetRoutes(router)
	//bluetooth.Mod.Setup(configMap)
	//bluetooth.Mod.SetRoutes(router)
	pybus.Mod.Setup(configMap)
	pybus.Mod.SetRoutes(router)
	db.Mod.Setup(configMap)
	mqtt.Mod.Setup(configMap)

	// Connect bluetooth device on startup
	//bluetooth.Connect()

	Start(router)
}
