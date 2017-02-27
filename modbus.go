package modbus

import (
	"net"
)

// FunctionCodes
const (
	// Bit access
	FuncCodeReadDiscreteInputs = 2
	FuncCodeReadCoils          = 1

	// 16-bit access
	FuncCodeReadInputRegisters   = 4
	FuncCodeReadHoldingRegisters = 3
)

// ExceptionCode
const (
	// DO NOT CHANGE THIS VALUE
	ExceptionCodeSuccess                            = 0
	ExceptionCodeIllegalFunction                    = 1
	ExceptionCodeIllegalDataAddress                 = 2
	ExceptionCodeIllegalDataValue                   = 3
	ExceptionCodeServerDeviceFailure                = 4
	ExceptionCodeAcknowledge                        = 5
	ExceptionCodeServerDeviceBusy                   = 6
	ExceptionCodeMemoryParityError                  = 8
	ExceptionCodeGatewayPathUnavailable             = 10
	ExceptionCodeGatewayTargetDeviceFailedToRespond = 11

	// custom exceptions
	ExceptionCodeCreationError = 0xE1
	ExceptionCodeBadUnitID     = 0xE2
)

// MaxADULength for modbus tcp
const MaxADULength = 260

// LoggerEnable - set if logger should be enable
const LoggerEnable = true

// ErrorHandler for handling modbus errors
type ErrorHandler struct {
	FunctionCode  byte
	ExceptionCode byte
}

// ADUUnit structure for storing incoming requests
type ADUUnit struct {
	transactionID uint16
	protocolID    uint16
	length        uint16
	unitID        byte
	functionCode  byte

	data []byte
}

// Server interface
type Server interface {

	// Start Server
	ServerStart() (err error)

	HandleClient(c net.Conn)

	// Default reading operation (handle request)
	Read(c net.Conn) (err error)

	// Write data
	Write(c net.Conn, data []byte) (err error)

	// Parse received request from client
	ParseRequest(adu []byte, aduUnit *ADUUnit) (errHandler ErrorHandler)

	// Fault function for error handling and logging
	Fault(errHandler *ErrorHandler, detail string)

	CreateResponse(aduUnit *ADUUnit) (response []byte, errHandler ErrorHandler)

	/******************************\
	|* MODBUS OUTCOMING RESPONSES *|
	\******************************/
	//
	ResponseRHRegisters(aduUnit *ADUUnit) (response []byte, errHandler ErrorHandler)
}
