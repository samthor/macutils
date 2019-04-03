package main

import (
	"_desktop/macutils"
	"log"
)

func main() {
	keyCh, err := macutils.ListenForKey(macutils.KeyF12)
	if err != nil {
		log.Fatalf("couldn't listen for key: %v", err)
	}

	lw, err := macutils.SubscribeLogin()
	if err != nil {
		log.Fatalf("couldn't subscribe to login: %v", err)
	}

	for {
		select {
		case <-keyCh:
			log.Printf("F12 key down")

		case lu := <-lw.Change:
			log.Printf("got new user: %v", lu)

		case err := <-lw.Errors:
			log.Printf("WARN: got err in login: %v", err)
		}
	}
}
