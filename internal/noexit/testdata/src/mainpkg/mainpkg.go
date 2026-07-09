package main

import (
	"log"
	"os"
)

func main() {
	log.Fatal("ok in main") // no diagnostic expected
	os.Exit(0)              // no diagnostic expected
}

func notMain() {
	log.Fatal("not ok") // want `avoid calling log.Fatal outside func main of package main`
	os.Exit(1)          // want `avoid calling os.Exit outside func main of package main`
}
