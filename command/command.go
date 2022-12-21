package command

import (
	"strings"
	"log"
)

type Command struct {
	Source string
	SourceID byte
	Name string
	Arguments []string
}

func CanRun(source string, command string) bool {
	return true
}

func Parse(source string, sourceID byte, command string) Command {
	if command[0] != '/' {
		log.Fatalln("Parsed command does not start with \"/\"! Something impossible has happened!")
	}

	parsed := strings.Split(command[1:], " ")

	return Command{source, sourceID, parsed[0], parsed[1:]}
}