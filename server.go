package modbus

import (
	"encoding/binary"
	"io"
	"log"
	"net"
	"strconv"
	"time"
)

type server struct {
	port int
	addr string
	sm   SmartMeter
}

func (s *server) ServerStart() (err error) {

	// Create server
	p := strconv.Itoa(s.port)
	ln, err := net.Listen("tcp", s.addr+":"+p)

	// Check if it was succesufully created
	if err != nil {
		log.Println("Not possible to create server: ", err.Error())
		return err
	}

	// Run it
	for {
		if LoggerEnable {
			log.Println("Waiting for new request")
		}

		// Wait for request
		conn, err := ln.Accept()
		if err != nil {
			// Handle error
			log.Println("Accept error: ", err.Error())
			continue
		}

		if LoggerEnable {
			log.Println("Handle request")
		}
		// Asynchronously handle request
		go s.HandleClient(conn)
	}

	//return err
}

func (s *server) HandleClient(c net.Conn) {

	defer c.Close()

	for {
		if s.Read(c) != nil {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func (s *server) Read(c net.Conn) (err error) {

	// Buffer to store received data
	data := make([]byte, MaxADULength)

	if LoggerEnable {
		log.Println("Reading available data...")
	}

	// Read available data
	n, err := c.Read(data)

	// Check for errors
	if err != nil {
		log.Println("Read error:", err.Error())
		if err != io.EOF {
			if n == MaxADULength {
				log.Println("Read error (packet length is bigger than max ADU length):", err.Error())
			} else {
				log.Println("Read error:", err.Error())
			}
			return err
		}
	}

	if LoggerEnable {
		log.Println("Data length: ", n)
		log.Println("Data: ", data[:n])
	}

	if LoggerEnable {
		log.Println("Parsing request...")
	}

	// Get request (parse it)
	request := ADUUnit{}
	errHandler := s.ParseRequest(data[:n], &request)
	if errHandler.ExceptionCode != ExceptionCodeSuccess {
		//TODO check error
		log.Println("Parse error:", errHandler)
		//TODO write and send error to client?
		// TMP
		return
	}

	if LoggerEnable {
		log.Println("Parsed request: ", request)
	}

	// Create response
	response, errHandler := s.CreateResponse(&request)
	if errHandler.ExceptionCode != ExceptionCodeSuccess {
		//TODO check error
		log.Println("Create error: ", errHandler)
		//TODO send error?
		return
	}

	if LoggerEnable {
		log.Println("Created response: ", response)
		log.Println("Sending response...")
	}

	err = s.Write(c, response)
	if err != nil {
		//TODO check error
		log.Println("Write error:", err.Error())
	}

	return
}

// Write
func (s *server) Write(c net.Conn, data []byte) (err error) {

	//TODO
	n, err := c.Write(data)

	// Check error
	if err != nil {
		log.Println("Sending error: ", err.Error())
		log.Println("Number of sent bytes: ", n)
		return err
	}

	if LoggerEnable {
		log.Println("Number of sent bytes: ", n)
	}

	return err
}

// Parse received request from client
func (s *server) ParseRequest(adu []byte, aduUnit *ADUUnit) (errHandler ErrorHandler) {
	errHandler.ExceptionCode = ExceptionCodeSuccess

	aduLength := len(adu)

	// Minimal length = MBAP + functioncode, ie. 8 bytes
	if aduLength < 8 {
		log.Println("ADU is too short")
		errHandler.ExceptionCode = ExceptionCodeIllegalDataValue
		return errHandler
	}

	// Read MBAP
	aduUnit.transactionID = binary.BigEndian.Uint16(adu)
	aduUnit.protocolID = binary.BigEndian.Uint16(adu[2:])
	aduUnit.unitID = adu[6] //TODO check unit ID

	// Read function code
	aduUnit.functionCode = adu[7]

	// Check if function code is in supported modbus range (later, we check, if we support this function)
	if aduUnit.functionCode < 1 || aduUnit.functionCode > 43 {
		log.Println("Function code is out of range: ", aduUnit.functionCode)
		errHandler.FunctionCode = aduUnit.functionCode
		errHandler.ExceptionCode = ExceptionCodeIllegalFunction
		return errHandler
	}

	// Read data
	aduUnit.data = adu[8:]

	// Get expected "rest" length
	length := binary.BigEndian.Uint16(adu[4:])
	// This should apply: MBAP + functioncode + data == len(adu) == defined length + MBAP - 1 (unitID is included in defined length) == defined length + 6
	if (6 + length) != uint16(len(adu)) {
		log.Println("ADU has invalid specification of length")
		errHandler.FunctionCode = aduUnit.functionCode
		errHandler.ExceptionCode = ExceptionCodeIllegalDataValue
		return errHandler
	}

	if LoggerEnable {
		log.Println("ADU Array: ", adu)
		log.Println("ADU Unit:  ", aduUnit)
	}

	return errHandler
}

func (s *server) CreateResponse(aduUnit *ADUUnit) (response []byte, errHandler ErrorHandler) {

	switch aduUnit.functionCode {
	case FuncCodeReadHoldingRegisters:
		// Get number of registers to read
		aduUnit.length = binary.BigEndian.Uint16(aduUnit.data[2:])

		// Check if requested registers length is in defined range
		if aduUnit.length < 1 || aduUnit.length > 125 {
			log.Println("ADU is too short")
			errHandler.ExceptionCode = ExceptionCodeIllegalDataValue
			return nil, errHandler
		}
		response, errHandler = s.ResponseRHRegisters(aduUnit)

		if errHandler.ExceptionCode != ExceptionCodeSuccess {
			errHandler.FunctionCode = FuncCodeReadHoldingRegisters
			return nil, errHandler
		}

		if response == nil {
			log.Println("No response created")
			errHandler.ExceptionCode = ExceptionCodeCreationError
			errHandler.FunctionCode = FuncCodeReadHoldingRegisters
			return nil, errHandler
		}
		//TODO check ErrorHandler
	default:
		log.Println("Unsupported function")
		errHandler.ExceptionCode = ExceptionCodeIllegalFunction
		errHandler.FunctionCode = FuncCodeReadHoldingRegisters
		return nil, errHandler
	}

	return response, errHandler
}

func (s *server) ResponseRHRegisters(aduUnit *ADUUnit) (response []byte, errHandler ErrorHandler) {

	if LoggerEnable {
		log.Println("Responsing RHRegisters request...")
	}

	value, errHandler := s.sm.GetRHRegisterValue(aduUnit.data, int(aduUnit.unitID))

	if errHandler.ExceptionCode != ExceptionCodeSuccess {
		log.Println("Unable to create reponse")
		return nil, errHandler
	}

	if LoggerEnable {
		log.Println("Value buffer: ", value)
	}

	// Number of bytes for register values (x2 because it is 16bit registers)
	numOfRegs := aduUnit.length * 2
	// Data length = registers values + function code (1B) + registers values length (1B) + unit ID (1B)
	dataLength := numOfRegs + 3
	// MBAP - unit ID + data length == 7 - 1 + data length
	finalSize := 6 + dataLength

	if LoggerEnable {
		log.Println("Final size: ", finalSize)
	}

	// Create response buffer
	response = make([]byte, finalSize)

	/*----------------------------------------------------------------------*\
	| transaction ID | protocol ID | length | unit ID | function code | data |
	--------------------------------------------------------------------------
	|      2 B       |     2 B     |  2 B   |   1 B   |     1 B       | N B  |
	\*----------------------------------------------------------------------*/
	response[0] = byte(aduUnit.transactionID >> 8)
	response[1] = byte(aduUnit.transactionID)
	response[2] = byte(aduUnit.protocolID >> 8)
	response[3] = byte(aduUnit.protocolID)
	response[4] = byte(dataLength >> 8)
	response[5] = byte(dataLength)
	response[6] = aduUnit.unitID
	response[7] = FuncCodeReadHoldingRegisters
	response[8] = byte(numOfRegs)

	// TODO fill right data (now 0s)
	i := 0
	for index := uint16(9); index < finalSize; index++ {
		response[index] = value[i]
		i++
		if i == 4 {
			break
		}
	}

	if LoggerEnable {
		log.Println("RHRegisters response: ", response)
	}

	return response, errHandler
}

func (s *server) Fault(errHandler *ErrorHandler, detail string) {
	log.Println("fault")
}

// NewTCPServer ...
func NewTCPServer(port int, addr string, sm SmartMeter) Server {
	return &server{port: port, addr: addr, sm: sm}
}
