package main

import ()

type ClientCommandType int

const (
	_ ClientCommandType = iota
	UpdateLift
	UpdateLiftSystem
)

var ClientCommandTypes = [...]string{
	"None",
	"UpdateLift",
	"UpdateLiftSystem",
}

func (b ClientCommandType) String() string {
	return ClientCommandTypes[b]
}

type ClientCommand struct {
	Command string      `json:"command"`
	Data    interface{} `json:"data"`
}

func NewCommand(typ ClientCommandType, data interface{}) *ClientCommand {
	return &ClientCommand{typ.String(), data}
}
