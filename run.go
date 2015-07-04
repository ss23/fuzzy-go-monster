package main

import "fmt"
import "net"
import "os"
import "flag" // Cause all the cool kids use flags, not just argv
import "io/ioutil"
import "time"

// Create our semaphore (I don't know enough Go to be sure this is the 'best' way
var sem = make(chan int, 1)
var fuzzers = make([]Fuzzer, 0)

func main() {
	fmt.Println("Starting daemon...")

	// Do argument parsing
	pathPtr := flag.String("syncdir", "", "The sync directory your afl fuzzers are using")
	flag.Parse()

	if *pathPtr == "" {
		fmt.Println("syncdir is required")
		os.Exit(1)
	}

	// Verify that our path is actually a real path
	files, _ := ioutil.ReadDir(*pathPtr)
	for _, f := range files {
		fmt.Println("Added fuzzer: " + f.Name())
		fuzzy := Fuzzer{name: f.Name(), active: false}
		fuzzers = append(fuzzers, fuzzy) // Add the new fuzzer
	}

	if len(fuzzers) == 0 {
		fmt.Println("Look here, you silly sally. You don't have anything in your sync directory as yet. That's probably fine, but if it's not, you should fix it. Like, you know, if you were planning to resume a fuzz or something. Maybe you got the path wrong? I don't know, I don't judge, I'm just letting you know. Hey maybe it's your first time? You new around here? That's cool, I can dig it. In any case, I'll carry on under the assumption that you are new. So... yeah.")
	}

	// Begin listening for connections
	ln, err := net.Listen("tcp", ":8123")
	if err != nil {
		fmt.Println("Error creating socket server")
		os.Exit(1)
	}

	for {
		conn, err := ln.Accept()
		if err != nil {
			fmt.Println("Incoming connection caused error")
			continue
		}
		go handleConnection(conn)
	}
}

func handleConnection(conn net.Conn) {
	fmt.Println("Connection started...")

	// Get an exclusive lock on our fuzzers
	sem <- 1

	// Check whether there is a fuzzer they can use
	potential := false
	for key, f := range fuzzers {
		if !f.active {
			// A non-active fuzzer we can assign
			fuzzers[key].active = true
			conn.Write([]byte(f.name))
			potential = true
			break
		}
	}
	if !potential {
		// We need to create a new fuzzer + name for them
		newname := fmt.Sprintf("fuzzer%03d", len(fuzzers)+1)
		newfuzzer := Fuzzer{name: newname, active: true}
		fuzzers = append(fuzzers, newfuzzer)
		conn.Write([]byte(newname))
	}
	// Release the lock
	<-sem
}

type Fuzzer struct {
	name       string
	active     bool
	lastReport time.Time // eventually we be used as a "they haven't reported in for x time, assume they're inactive"
}
