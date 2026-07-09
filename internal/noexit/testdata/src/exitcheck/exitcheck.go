package exitcheck

import (
	"log"
	"os"
)

func helper() {
	log.Fatal("exit")  // want `avoid calling log.Fatal outside func main of package main`
	log.Fatalf("exit")  // want `avoid calling log.Fatalf outside func main of package main`
	log.Fatalln("exit") // want `avoid calling log.Fatalln outside func main of package main`
	os.Exit(1)          // want `avoid calling os.Exit outside func main of package main`
}
