package main

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	logger "github.com/d2r2/go-logger"
	"github.com/yosssi/gmq/mqtt"
	"github.com/yosssi/gmq/mqtt/client"

	"github.com/kubeedge/kubeedge/edge/pkg/devicetwin/dttype"
)

const (
	DeviceETPrefix            = "$hw/events/device/"
	TwinETUpdateSuffix        = "/twin/update"
	TwinETUpdateDetalSuffix   = "/twin/update/delta"
	DeviceETStateUpdateSuffix = "/state/update"
	TwinETCloudSyncSuffix     = "/twin/cloud_updated"
	TwinETGetResultSuffix     = "/twin/get/result"
	TwinETGetSuffix           = "/twin/get"

	DeviceID = "switch"
)

var lg = logger.NewPackageLogger("main",
	logger.DebugLevel,
	// logger.InfoLevel,
)

var onceClient sync.Once

//DeviceStateUpdate is the structure used in updating the device state
type DeviceStateUpdate struct {
	State string `json:"state,omitempty"`
}

//BaseMessage the base struct of event message
type BaseMessage struct {
	EventID   string `json:"event_id"`
	Timestamp int64  `json:"timestamp"`
}

//TwinValue the struct of twin value
type TwinValue struct {
	Value    *string        `json:"value, omitempty"`
	Metadata *ValueMetadata `json:"metadata,omitempty"`
}

//ValueMetadata the meta of value
type ValueMetadata struct {
	Timestamp int64 `json:"timestamp, omitempty"`
}

//TypeMetadata the meta of value type
type TypeMetadata struct {
	Type string `json:"type,omitempty"`
}

//TwinVersion twin version
type TwinVersion struct {
	CloudVersion int64 `json:"cloud"`
	EdgeVersion  int64 `json:"edge"`
}

//MsgTwin the struct of device twin
type MsgTwin struct {
	Expected        *TwinValue    `json:"expected,omitempty"`
	Actual          *TwinValue    `json:"actual,omitempty"`
	Optional        *bool         `json:"optional,omitempty"`
	Metadata        *TypeMetadata `json:"metadata,omitempty"`
	ExpectedVersion *TwinVersion  `json:"expected_version,omitempty"`
	ActualVersion   *TwinVersion  `json:"actual_version,omitempty"`
}

//DeviceTwinUpdate the struct of device twin update
type DeviceTwinUpdate struct {
	BaseMessage
	Twin map[string]*MsgTwin `json:"twin"`
}

type deviceState struct {
	Consumption int    `json:"consumption"`
	Linkquality int    `json:"linkquality"`
	Power       int    `json:"power"`
	State       string `json:"state"`
	Temperature int    `json:"temperature"`
}

func main() {
	// connect to Mqtt broker
	cli := connectToMqtt()

	InitCLient(cli)

	for {
		time.Sleep(time.Second * 2)
	}
	lg.Info("exited")
}

func InitCLient(cli *client.Client) {
	fmt.Println("init client ...")

	onceClient.Do(func() {
		topic := DeviceETPrefix + DeviceID + TwinETUpdateDetalSuffix
		err := cli.Subscribe(&client.SubscribeOptions{
			SubReqs: []*client.SubReq{
				&client.SubReq{
					TopicFilter: []byte(topic),
					QoS:         mqtt.QoS0,
					Handler: func(topicName, message []byte) {
						OperateUpdateDetalSub(cli, message)
					},
				},
			},
		})

		if err != nil {
			panic(err)
		}
	})

	onceClient.Do(func() {
		topic := "zigbee2mqtt/0x00158d0002c80b58"
		err := cli.Subscribe(&client.SubscribeOptions{
			SubReqs: []*client.SubReq{
				&client.SubReq{
					TopicFilter: []byte(topic),
					QoS:         mqtt.QoS0,
					Handler: func(topicName, message []byte) {
						OperateUpdateZigbeeSub(cli, message)
					},
				},
			},
		})

		if err != nil {
			panic(err)
		}
	})
}

func OperateUpdateZigbeeSub(cli *client.Client, msg []byte) {
	fmt.Printf("Receive msg %v\n\n", string(msg))
	current := &deviceState{}
	if err := json.Unmarshal(msg, current); err != nil {
		fmt.Printf("unmarshl receive msg to deviceState, error %v\n", err)
		return
	}

	// publish temperature status to mqtt broker
	publishToMqtt(cli, current.State)
}

func OperateUpdateDetalSub(cli *client.Client, msg []byte) {
	fmt.Printf("Receive msg %v\n\n", string(msg))
	current := &dttype.DeviceTwinUpdate{}
	if err := json.Unmarshal(msg, current); err != nil {
		fmt.Printf("unmarshl receive msg DeviceTwinUpdate{} to error %v\n", err)
		return
	}
	value := *(current.Twin["state"].Expected.Value)

	publishToMqttToZigbee(cli, value)
}

func connectToMqtt() *client.Client {
	cli := client.New(&client.Options{
		// Define the processing of the error handler.
		ErrorHandler: func(err error) {
			fmt.Println(err)
		},
	})
	defer cli.Terminate()

	// Connect to the MQTT Server.
	err := cli.Connect(&client.ConnectOptions{
		Network:  "tcp",
		Address:  "localhost:1883",
		ClientID: []byte("receive-client"),
	})

	if err != nil {
		panic(err)
	}
	return cli
}

func publishToMqttToZigbee(cli *client.Client, state string) {
	topic := "zigbee2mqtt/0x00158d0002c80b58/set"

	actualMap := map[string]string{"state": state}
	body, _ := json.Marshal(actualMap)

	cli.Publish(&client.PublishOptions{
		TopicName: []byte(topic),
		QoS:       mqtt.QoS0,
		Message:   body,
	})
}

func publishToMqtt(cli *client.Client, state string) {
	deviceTwinUpdate := "$hw/events/device/" + DeviceID + "/twin/update"

	updateMessage := createActualUpdateMessage(state)
	twinUpdateBody, _ := json.Marshal(updateMessage)

	cli.Publish(&client.PublishOptions{
		TopicName: []byte(deviceTwinUpdate),
		QoS:       mqtt.QoS0,
		Message:   twinUpdateBody,
	})
}

//createActualUpdateMessage function is used to create the device twin update message
func createActualUpdateMessage(actualValue string) DeviceTwinUpdate {
	var deviceTwinUpdateMessage DeviceTwinUpdate
	actualMap := map[string]*MsgTwin{"state": {Actual: &TwinValue{Value: &actualValue}, Metadata: &TypeMetadata{Type: "Updated"}}}
	deviceTwinUpdateMessage.Twin = actualMap
	return deviceTwinUpdateMessage
}
