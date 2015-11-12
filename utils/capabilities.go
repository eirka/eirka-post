package utils

import (
	"fmt"
	"strings"

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
}

func CheckServices() {

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

	fmt.Println(strings.Repeat("*", 60))
	fmt.Printf("%-20v\n", "Available Services")
	fmt.Println(strings.Repeat("*", 60))
	fmt.Printf("%-20v\n\n", "Utils")
	fmt.Printf("%-20v%40v\n", "Akismet", Services.Utils.Akismet)
	fmt.Println(strings.Repeat("*", 60))
	fmt.Printf("%-20v\n\n", "Storage")
	fmt.Printf("%-20v%40v\n", "Amazon", Services.Storage.Amazon)
	fmt.Printf("%-20v%40v\n", "Google", Services.Storage.Google)
	fmt.Println(strings.Repeat("*", 60))

}
