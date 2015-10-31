package utils

import (
	"encoding/json"
	"fmt"
)

var Services *Capabilities

// controls capabilities for internal utils
type Capabilities struct {
	// storage capabilities
	Storage struct {
		Amazon bool
		Google bool
	}
}

func init() {

	if config.Settings.Amazon {
		Services.Storage.Amazon = true
	}

	if config.Settings.Google {
		Services.Storage.Google = true
	}
}

func (c Capabilities) Print() {

	// Marshal the structs into JSON
	output, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	fmt.Printf("%s\n", output)

}
