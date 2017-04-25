package main

import (
	"flag"
	"fmt"
	"os"
)

var (
	// Propgram name
	NAME,
	// Program version
	VERSION,
	// Build date
	BUILDDATE string
	// Send log messages to the console
	CONSOLELOG = LevelDebug
	// Listen requests at
	SERVERADDRESS string
	// SBSS API server address
	SBSSAPISERVER string

	PrintVersion bool
)

func init() {
	if NAME == "" {
		NAME = "sbss-vbook"
	}

	flag.IntVar(&CONSOLELOG, "v", 0, "Console verbose output, default 0 - off, 7 - debug")
	flag.BoolVar(&PrintVersion, "V", false, "Print version")
	flag.StringVar(&SERVERADDRESS, "L", ":8080", "Listen http request at [:8080]")
	flag.StringVar(&SBSSAPISERVER, "A", "http://localhost", "LANBilling SBSS API server address")
}

// Покажи версию программы и заверши процесс
func showVersion(log *Log) {
	var str = fmt.Sprintf("SBSS vcard address book server (%s) %s, built %s", NAME, VERSION, BUILDDATE)

	if PrintVersion {
		fmt.Println(str)
		os.Exit(0)
	} else {
		if log != nil {
			log.Notice(str)
		}
	}
}
