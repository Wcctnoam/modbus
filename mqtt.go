package modbus

import (
	"fmt"
	"os"
	"strconv"
	"time"

	MQTT "github.com/eclipse/paho.mqtt.golang"
)

// MQTTClient interface
type MQTTClient interface {
	SetMQTTPub()
	StartMQTTSub(choke chan [2]string)
}

// mqttSettings for MQTT client
type mqttSettings struct {
	topic  string
	broker string
	id     string
	user   string
	passwd string
	action string
	qos    int
}

// NewMqttClient - get new mqtt client with specified settings
func NewMqttClient(topic string, broker string, id string, user string, passwd string, action string, qos int) MQTTClient {
	return &mqttSettings{topic: topic, broker: broker, id: id, user: user, passwd: passwd, action: action, qos: qos}
}

// Test client for publishing
func (mq *mqttSettings) SetMQTTPub() {

	opts := MQTT.NewClientOptions()
	//opts.AddBroker("tcp://eu.thethings.network:1883")
	opts.AddBroker("tcp://172.18.0.2:1883")
	opts.SetClientID("modpubtest456")
	// opts.SetUsername("sdf654sdf")
	// opts.SetPassword("ttn-account-v2.VzKrXNILq_3NUBtVaGgH2baGYSm60I7Blr6HMAd8VeE")

	client := MQTT.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		panic(token.Error())
	}
	fmt.Println("Sample Publisher Started")

	ch1 := make(chan string)
	ch2 := make(chan string)

	go func() {
		tmp := 1
		for i := 0; i < 10; i++ {
			fmt.Println("---- doing publish ----")
			payload := strconv.Itoa(tmp)
			token := client.Publish("/modbus/Node1/volt1", byte(0), false, payload)
			token.Wait()
			time.Sleep(500 * time.Millisecond)
			tmp++
		}

		ch1 <- "done 1"
	}()

	go func() {
		tmp := 1
		for i := 0; i < 10; i++ {
			fmt.Println("---- doing publish ----")
			payload := strconv.Itoa(tmp)
			token := client.Publish("/modbus/Node1/volt4", byte(0), false, payload)
			token.Wait()
			time.Sleep(800 * time.Millisecond)
			tmp++
		}

		ch2 <- "done 2"
	}()

	fmt.Println(<-ch1)
	fmt.Println(<-ch2)

	client.Disconnect(250)
	fmt.Println("Sample Publisher Disconnected")
}

/**
* StartMQTTSub
* @param chanBridge channel for creating pipe between mqtt and smart meter storage
 */
func (mq *mqttSettings) StartMQTTSub(chanBridge chan [2]string) {

	// Set mqtt settings
	opts := MQTT.NewClientOptions()
	opts.AddBroker(mq.broker)
	opts.SetClientID(mq.id)
	opts.SetUsername(mq.user)
	opts.SetPassword(mq.passwd)
	// opts.SetCleanSession(*cleansess)
	// if *store != ":memory:" {
	// 	opts.SetStore(MQTT.NewFileStore(*store))
	// }

	// channel for incoming patopics and payloads
	choke := make(chan [2]string)

	// Set handler
	opts.SetDefaultPublishHandler(func(client MQTT.Client, msg MQTT.Message) {
		choke <- [2]string{msg.Topic(), string(msg.Payload())}
	})

	// Create new client and check if it was succefull
	client := MQTT.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		panic(token.Error())
	}

	// Subscribe for specific topic
	if token := client.Subscribe(mq.topic, byte(mq.qos), nil); token.Wait() && token.Error() != nil {
		fmt.Println(token.Error())
		os.Exit(1)
	}

	// Run
	// it should run forever (TODO maybe with stopping channel?)
	receiveCount := 0
	num := 20
	for receiveCount < num {
		incoming := <-choke
		fmt.Printf("RECEIVED TOPIC: %s MESSAGE: %s\n", incoming[0], incoming[1])
		receiveCount++
		// Send incoming to bridge pipe to store this data
		chanBridge <- incoming
	}

	client.Disconnect(250)
	fmt.Println("Sample Subscriber Disconnected")

}

// package main

// import (
// 	"flag"
// 	"fmt"
// 	"os"

// 	MQTT "github.com/eclipse/paho.mqtt.golang"
// )

// /*
// Options:
//  [-help]                      Display help
//  [-a pub|sub]                 Action pub (publish) or sub (subscribe)
//  [-m <message>]               Payload to send
//  [-n <number>]                Number of messages to send or receive
//  [-q 0|1|2]                   Quality of Service
//  [-clean]                     CleanSession (true if -clean is present)
//  [-id <clientid>]             CliendID
//  [-user <user>]               User
//  [-password <password>]       Password
//  [-broker <uri>]              Broker URI
//  [-topic <topic>]             Topic
//  [-store <path>]              Store Directory

// */

// func main() {
// 	topic := flag.String("topic", "", "The topic name to/from which to publish/subscribe")
// 	broker := flag.String("broker", "tcp://iot.eclipse.org:1883", "The broker URI. ex: tcp://10.10.1.1:1883")
// 	password := flag.String("password", "", "The password (optional)")
// 	user := flag.String("user", "", "The User (optional)")
// 	id := flag.String("id", "testgoid", "The ClientID (optional)")
// 	cleansess := flag.Bool("clean", false, "Set Clean Session (default false)")
// 	qos := flag.Int("qos", 0, "The Quality of Service 0,1,2 (default 0)")
// 	num := flag.Int("num", 1, "The number of messages to publish or subscribe (default 1)")
// 	payload := flag.String("message", "", "The message text to publish (default empty)")
// 	action := flag.String("action", "", "Action publish or subscribe (required)")
// 	store := flag.String("store", ":memory:", "The Store Directory (default use memory store)")
// 	flag.Parse()

// 	if *action != "pub" && *action != "sub" {
// 		fmt.Println("Invalid setting for -action, must be pub or sub")
// 		return
// 	}

// 	if *topic == "" {
// 		fmt.Println("Invalid setting for -topic, must not be empty")
// 		return
// 	}

// 	fmt.Printf("Sample Info:\n")
// 	fmt.Printf("\taction:    %s\n", *action)
// 	fmt.Printf("\tbroker:    %s\n", *broker)
// 	fmt.Printf("\tclientid:  %s\n", *id)
// 	fmt.Printf("\tuser:      %s\n", *user)
// 	fmt.Printf("\tpassword:  %s\n", *password)
// 	fmt.Printf("\ttopic:     %s\n", *topic)
// 	fmt.Printf("\tmessage:   %s\n", *payload)
// 	fmt.Printf("\tqos:       %d\n", *qos)
// 	fmt.Printf("\tcleansess: %v\n", *cleansess)
// 	fmt.Printf("\tnum:       %d\n", *num)
// 	fmt.Printf("\tstore:     %s\n", *store)

// }
