package main

import (
	"flag"
	"log"
	"strings"
	"time"

	"github.com/foxconn4tech/modbus"
)

func main() {
	// Coammand line arguments
	// Address and port for modbus server (listening)
	addr := flag.String("ip", "", "The modbus server listening addr, i.e. 127.0.0.1")
	port := flag.Int("port", 0, "The port for listening")
	// Config json file specifying smart meter mappings, see @conf.json file
	configFile := flag.String("config", "", "The json config file")
	flag.Parse()

	// Check parameters
	if *addr == "" {
		log.Println("The server address is empty, use -ip setting")
		return
	}

	if *port == 0 {
		log.Println("The port for listening was not set, use -port setting")
		return
	}

	if *configFile == "" {
		log.Println("The config file is not specified, use -config setting")
		return
	}

	if modbus.LoggerEnable == true {
		log.Println("Loading config file...")
	}

	// Create smart meter with settings according to config file
	smartMeter := modbus.NewSmartMeter(*configFile)
	if smartMeter == nil {
		log.Println("Config file was not succefully loaded")
		return
	}

	if modbus.LoggerEnable == true {
		log.Println(smartMeter)
	}

	// Channel for communication among mqtt client and smart meter storage
	// process: incoming mqqt message -> send it to this channel -> channel sends value to smart meter -> smart meter stores this value
	chanBridge := make(chan [2]string)

	// New MQTT client
	// TODO make it configurable //"tcp://iot.eclipse.org:1883"
	//mqttClient := modbus.NewMqttClient("/modbus/#", "tcp://eu.thethings.network:1883", "testmodid123", "sdf654sdf", "ttn-account-v2.VzKrXNILq_3NUBtVaGgH2baGYSm60I7Blr6HMAd8VeE", "sub", 0)
	mqttClient := modbus.NewMqttClient("/modbus/#", "tcp://172.18.0.2:1883", "testmodid123", "", "", "sub", 0)

	// Start client (pub and sub)
	go mqttClient.SetMQTTPub()
	go mqttClient.StartMQTTSub(chanBridge)

	// Start function that is waiting for incoming request through channel and then stores it
	go func() {
		for {
			incoming := <-chanBridge

			if modbus.LoggerEnable == true {
				log.Println("Writing values: ", incoming)
			}

			// Topics are usually in this format "/root/level1/level2"
			topics := strings.Split(incoming[0], "/")
			topicsNum := len(topics)

			// Check if the length of topics (ie. numbur of levels separated by "/") is enough
			if topicsNum < 3 {
				log.Printf("Invalid topic length (%d)\n", topicsNum)
				continue
			}

			// Last two levels is nodeID and final topic (ie reg num): ".../NodeID/RegNum"
			nodeID := topics[topicsNum-2]
			topic := topics[topicsNum-1]

			//
			if strings.HasPrefix(nodeID, "Node") {
				smartMeter.WriteValues(nodeID+"/"+topic, incoming[1])
			} else {
				log.Println("Invalid node id for received topic")
			}
		}
	}()

	// Initialize and start modbus TCP server
	server := modbus.NewTCPServer(*port, *addr, smartMeter)
	if server == nil {
		log.Println("Server was not succesfully initialize")
		return
	}
	log.Println("Server starts.................")
	server.ServerStart()

	//TMP waiting feature
	time.Sleep(10 * time.Second)
}
