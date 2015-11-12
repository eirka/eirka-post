package utils

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/techjanitor/pram-libs/config"
)

var Services *Capabilities

// controls capabilities for internal utils
type Capabilities struct {

	// utils
	Utils struct {
		Akismet bool
	}

	// storage capabilities
	Storage struct {
		Amazon bool
		Google bool
	}
}

func init() {

	Services = &Capabilities{}

	if config.Settings.Amazon.Key != "" {
		Services.Storage.Amazon = true
	}

	if config.Settings.Google.Key != "" {
		Services.Storage.Google = true
	}

	if config.Settings.Akismet.Key != "" {
		Services.Utils.Akismet = true
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
