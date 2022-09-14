package config

import (
	"log"
	"strings"
	"strconv"
)

type Config struct {
	Data map[string]string
}

func (config Config) Exists(key string) bool {
	_, exists := config.Data[key]
	return exists
}

func (config Config) GetString(key string) string {
	data, exists := config.Data[key]
	
	if !exists {
		log.Fatalln("Failed to read an option from server.properties: The option", key, "does not exist.")
	}
	
	return data
}

func (config Config) GetNumber(key string) int {
	data, exists := config.Data[key]
	
	if !exists {
		log.Fatalln("Failed to read an option from server.properties: The option", key, "does not exist.")
	}
	
	number, err := strconv.ParseInt(data, 10, 32)
	
	if err != nil {
		log.Fatalln("Failed to read an option from server.properties: The option", key, "contains an invalid number value.")
	}
	
	return int(number)
}

func (config Config) GetBoolean(key string) bool {
	data, exists := config.Data[key]
	
	if !exists {
		log.Fatalln("Failed to read an option from server.properties: The option", key, "does not exist.")
	}
	
	boolean := false
	
	if data == "true" {
		boolean = true
	} else if data == "false" {
		boolean = false
	} else {
		log.Fatalln("Failed to read an option from server.properties: The option", key, "contains an invalid boolean value.")
	}
	
	return boolean
}

func ParseConfig(data string) Config {
	lines := strings.Split(data, "\n")
	
	config := make(map[string]string)
	
	for i := 0; i < len(lines); i++ {
		line := lines[i]
		
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		
		if !strings.Contains(line, "=") {
			log.Fatalln("Failed to parse server.properties: Line", i + 1, "does not contain the \"=\" character.")
		}
		
		parsedLine := strings.Split(line, "=")
		optionName := parsedLine[0]
		optionValue := parsedLine[1]
		
		config[optionName] = optionValue
	}
	
	return Config{config}
}
