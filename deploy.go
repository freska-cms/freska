package main

import (
	"log"
	"os"
	"path/filepath"

	"github.com/freska-cms/assets"
)

// RunDeploy builds the assets, builds the server,
// and then runs the script at ./bin/deploy if it exists
func RunDeploy(args []string) {

	// Build our app assets and update secrets/assets.json
	buildAssets()

	// Build deploy server
	buildDeployServer()

	deploy := filepath.Join("bin", "deploy")

	_, err := os.Stat(deploy)
	if err != nil {
		log.Printf("Could not find deploy script at %s", deploy)
		return
	}

	// Default to development
	mode := ModeDevelopment
	if len(args) == 3 {
		mode = args[2]
	}

	log.Printf("Running deploy from " + deploy)
	result, err := runCommand(deploy, mode)
	if err != nil {
		log.Printf("Error running deploy %s", err)
		return
	}

	log.Printf(string(result))
}

// buildAssets compiles the app assets before a deploy, so that they're available for production use
func buildAssets() {
	log.Printf("Compiling assets...")
	err := assets.New(true).Compile("src", "public")
	if err != nil {
		log.Fatalf("#error compiling assets %s", err)
	}
}
