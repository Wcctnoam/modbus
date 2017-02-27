package modbus

import (
	"encoding/binary"
	"encoding/json"
	"log"
	"math"
	"os"
	"strconv"
)

// SmartMeter public interface
type SmartMeter interface {
	// Mapping between UnitID (modbus) and NodeID (mqtt)
	GetNodeID(unitID int) (nodeID string, errHandler ErrorHandler)

	// Get the right topic (mqtt) for specified unitID (modbus) and reg address (modbus)
	GetTopic(unitID int, regAddr uint16) (topic string, errHandler ErrorHandler)

	// Get value type for specified unitID (modbus) and reg address (modbus)
	GetValueType(unitID int, regAddr uint16) (valueType int, errHandler ErrorHandler)

	// Get actual value for RHRegister function
	GetRHRegisterValue(data []byte, unitID int) (value []byte, errHandler ErrorHandler)

	// Check if requested register length (regsNum) for specified reg address (modbus) and unitID (modbus) is correct
	CheckRegsLength(unitID int, regsNum uint16, regAddr uint16) (tag bool, errHandler ErrorHandler)

	// Write values to smart meter storage structure (typically from MQTT)
	WriteValues(topics string, value string)
}

// Structure including sm storage and mapping, implements SmartMeter interace
type smartMeter struct {
	// Storage for smart meter values, typically MQTT (hash map in the form map["nodeID/regNum"] = "string value")
	smValuesMap map[string]string

	// Mapping tables \\
	// map[unitID] = MappingUnitTable (see @MappingUnitTable)
	mappUnitTable map[int]MappingUnitTable
	// existing types of smart meter, see @MappingAllTypeTable and mappUnitTable.smType
	smTypes []MappingAllTypeTable
}

// MappingAllTypeTable specifies type of smart meter, it's a hashmap specifying topic (mqtt) and value type (modbus) for each register (modbus reg num)
type MappingAllTypeTable struct {
	// map[registerNum] = MappingTypeTable
	mType map[int]MappingTypeTable
}

// MappingTypeTable specifies topic (mapping between modbus reg num and mqtt topic)
type MappingTypeTable struct {
	topic   string
	valType int
}

// MappingUnitTable specifies nodeID (for mqtt topic) and smart meter type for each unitID (see @smartMeter.mappUnitTable)
type MappingUnitTable struct {
	nodeID string
	// Index to @smartMeter.smTypes table, specifies type of smart meter
	smType int
}

/*-------------------------*\
-----JSON TMP STRUCTURES-----
----------------------------*/

// MappingJSONRegisters - see example conf.json.comment file
type MappingJSONRegisters struct {
	Numbers []int
	Topics  []string
	// Selected value type, see @ValueType consts
	ValueTypes []int
}

// MappingJSONTable - see example conf.json.comment file
type MappingJSONTable struct {
	UnitID []int
	NodeID []string
	Type   []int
	Types  []MappingJSONRegisters
}

/*-------------------------*\
-----JSON STRUCTURES END----
----------------------------*/

// ValueTypes for registers, see above
const (
	ValueTypeFLOAT    = 1
	ValueTypeSIGNED   = 2
	ValueTypeUNSIGNED = 3
)

/**
* NewSmartMeter set smart meter configuration
* @param config string path to config file, see @conf.json as example file
* return &smartMeter
 */
func NewSmartMeter(config string) SmartMeter {

	// Get config
	file, _ := os.Open(config)
	decoder := json.NewDecoder(file)
	var mapp MappingJSONTable
	// Read JSON config into MappingJSONTable structure
	err := decoder.Decode(&mapp)
	file.Close()
	// Check decodig status
	if err != nil {
		log.Println("Config error (file was not succefully decoded): ", err)
		os.Exit(1)
	}

	// Length of this two arraus must be the same, se @conf.json file
	if len(mapp.Type) != len(mapp.Types) {
		log.Println("Invalid config file, type length != types length!")
		os.Exit(1)
	}

	if LoggerEnable {
		log.Println("Converting config file to smart meter structure...")
	}

	// Convert to SmartMeter

	smartMeterNum := len(mapp.UnitID) // Number of devices, ie. number of mappings
	// Create sm mapp for unitIDs
	smMap := make(map[int]MappingUnitTable)
	for index := 0; index < smartMeterNum; index++ {
		mappUnitTable := MappingUnitTable{mapp.NodeID[index], mapp.Type[index]}
		smMap[mapp.UnitID[index]] = mappUnitTable
	}

	typesNum := len(mapp.Types)
	// Create sm mapp for smart meter types
	smTypes := make([]MappingAllTypeTable, typesNum)
	for index := 0; index < typesNum; index++ {
		smTypes[index].mType = make(map[int]MappingTypeTable)

		// For each type create mapp for registers
		typesLen := len(mapp.Types[index].Numbers)
		for t := 0; t < typesLen; t++ {
			smTypes[index].mType[mapp.Types[index].Numbers[t]] = MappingTypeTable{mapp.Types[index].Topics[t], mapp.Types[index].ValueTypes[t]}
		}
	}

	// Return smart meters
	return &smartMeter{smValuesMap: make(map[string]string), mappUnitTable: smMap, smTypes: smTypes}
}

func (sm *smartMeter) checkUnitID(unitID int) (errHandler ErrorHandler) {

	_, flag := sm.mappUnitTable[unitID]
	if flag == false {
		log.Println("Bad unit ID, not present")
		errHandler.ExceptionCode = ExceptionCodeBadUnitID
		return errHandler
	}
	return errHandler
}

func (sm *smartMeter) checkRegAddress(unitID int, regAddr uint16) (errHandler ErrorHandler) {

	errHandler = sm.checkUnitID(unitID)
	if errHandler.ExceptionCode != ExceptionCodeSuccess {
		return errHandler
	}

	//TODO first check sm type
	_, flag := sm.smTypes[sm.mappUnitTable[unitID].smType].mType[int(regAddr)]
	if flag == false {
		log.Println("Bad register address, not supported")
		errHandler.ExceptionCode = ExceptionCodeIllegalDataAddress
	}

	return errHandler
}

/**
* GetNodeID
* @param unitID unit ID from modbus request
* @return nodeID string right nodeID for unitID
 */
func (sm *smartMeter) GetNodeID(unitID int) (nodeID string, errHandler ErrorHandler) {

	// Check unitID (if it exists)
	errHandler = sm.checkUnitID(unitID)
	if errHandler.ExceptionCode != ExceptionCodeSuccess {
		return "", errHandler
	}

	// If everything is ok, get nodeID
	return sm.mappUnitTable[unitID].nodeID, errHandler
}

/**
* GetTopic
* @param unitID unit ID from modbus request
* @param regAddr register address from modbus request
* @return topic string right topic for unitID and specific register address
 */
func (sm *smartMeter) GetTopic(unitID int, regAddr uint16) (topic string, errHandler ErrorHandler) {

	// Check reg address
	errHandler = sm.checkRegAddress(unitID, regAddr)
	if errHandler.ExceptionCode != ExceptionCodeSuccess {
		return "", errHandler
	}

	// If everything is ok, you can access and return topic
	return sm.smTypes[sm.mappUnitTable[unitID].smType].mType[int(regAddr)].topic, errHandler
}

/**
* GetValueType
* @param unitID unit ID from modbus request
* @param regAddr register address from modbus request
* @return valueType int right value type for unitID and specific register address
 */
func (sm *smartMeter) GetValueType(unitID int, regAddr uint16) (valueType int, errHandler ErrorHandler) {

	// Check reg address
	errHandler = sm.checkRegAddress(unitID, regAddr)
	if errHandler.ExceptionCode != ExceptionCodeSuccess {
		return -1, errHandler
	}

	// If everything is ok, you can access and return value type
	return sm.smTypes[sm.mappUnitTable[unitID].smType].mType[int(regAddr)].valType, errHandler
}

/**
* GetValueType
* @param unitID unit ID from modbus request
* @param regNum requested register length/number from modbus request
* @param regAddr register address from modbus request
* @return tag bool true if it is ok, else false
 */
func (sm *smartMeter) CheckRegsLength(unitID int, regsNum uint16, regAddr uint16) (tag bool, errHandler ErrorHandler) {

	// Get and check value type
	valType, errHandler := sm.GetValueType(unitID, regAddr)
	if errHandler.ExceptionCode != ExceptionCodeSuccess {
		return false, errHandler
	}

	// According value type check register length
	switch valType {
	case ValueTypeFLOAT:
	case ValueTypeSIGNED:
	case ValueTypeUNSIGNED:
		// These values should be all 4 bytes
		if regsNum != 4 {
			errHandler.ExceptionCode = ExceptionCodeIllegalDataValue
			return false, errHandler
		}
		break
	default:

	}

	return true, errHandler
}

/**
* GetRHRegisterValue
* @param data byte array from modbus request (including reg address and requested length)
* @param unitID unit id from modbus request
* @return value []byte value for specific register
 */
func (sm *smartMeter) GetRHRegisterValue(data []byte, unitID int) (value []byte, errHandler ErrorHandler) {

	if LoggerEnable {
		log.Println("Getting RHRegs values from smart meter...")
	}

	// Get length of Data
	dataLength := len(data)

	// Data for RHRegs = address (2B) + regNum (2B)
	if dataLength != 4 {
		log.Println("Bad data length for RHRegs function")
		errHandler.ExceptionCode = ExceptionCodeIllegalDataValue
		return nil, errHandler
	}

	// Read values from data buffer, see above (addr + regs count)
	regAddr := binary.BigEndian.Uint16(data)
	regsNum := binary.BigEndian.Uint16(data[2:])

	// Check requested number of registers to read (including register address)
	t, errHandler := sm.CheckRegsLength(unitID, regsNum, regAddr)

	if errHandler.ExceptionCode != ExceptionCodeSuccess {
		return nil, errHandler
	}
	if t == false {
		log.Println("Invalid registers number")
		errHandler.ExceptionCode = ExceptionCodeIllegalDataValue
		return nil, errHandler
	}

	// We do not have to check errHandler, because we check it above in CheckRegsLength function
	nodeID, _ := sm.GetNodeID(unitID)
	topic, _ := sm.GetTopic(unitID, regAddr)

	if LoggerEnable {
		log.Printf("Get nodeID (%s) and topicID (%s)\n", nodeID, topic)
	}

	valueType, _ := sm.GetValueType(unitID, regAddr) // We do not have to check errHandler, because we check it above in CheckRegsLength function
	valueString, flag := sm.smValuesMap[nodeID+"/"+topic]
	if flag == false {
		log.Println("Values for this topic are not present in the buffer")
		errHandler.ExceptionCode = ExceptionCodeGatewayTargetDeviceFailedToRespond //TODO is that the right response?
		return nil, errHandler
		//TODO bad topics
	}

	if LoggerEnable {
		log.Printf("Get value type (%d) and value string (%s)\n", valueType, valueString)
		log.Println("Parsing string value...")
	}

	// TODO
	switch valueType {
	case ValueTypeFLOAT:
		valueFloat, err := strconv.ParseFloat(valueString, 32)
		if err != nil {
			log.Printf("Parsing float value from %s was not succesfull %s", valueString, err.Error())
			errHandler.ExceptionCode = ExceptionCodeCreationError
			//TODO maybe response error?
			return nil, errHandler
		}
		valueBits := math.Float32bits(float32(valueFloat))
		value = make([]byte, 4)
		binary.LittleEndian.PutUint32(value, valueBits)
		if LoggerEnable {
			log.Printf("Parsed float value %f from string %s, converted to bits %d and finally to bytes %s", valueFloat, valueString, valueBits, value)
		}
		break
	case ValueTypeSIGNED:
	case ValueTypeUNSIGNED:
		//TODO
	default:
		break
	}

	return value, errHandler
}

/**
* WriteValues
* @param topics topics identifier and key for storing
* @param value string value to store
 */
func (sm *smartMeter) WriteValues(topics string, value string) {
	if LoggerEnable {
		log.Printf("Writing value %s for topic %s\n", value, topics)
	}

	sm.smValuesMap[topics] = value
}
