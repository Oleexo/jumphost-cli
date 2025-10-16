package basic

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/Oleexo/jumphost-cli/internal/models"
	"github.com/Oleexo/jumphost-cli/internal/ui"
)

type Display struct {
	identity ui.Identity
}

func NewDisplay() *Display {
	return &Display{}
}

func (d *Display) SetIdentity(identity ui.Identity) {
	d.identity = identity
}

func (d *Display) SelectJumphostInstance(f func() ([]models.JumphostInstance, error)) (
	models.JumphostInstance,
	bool,
	error) {
	instances, err := f()
	if err != nil {
		return models.JumphostInstance{}, false, err
	}

	if len(instances) == 0 {
		fmt.Println("No jumphost instances found")
		return models.JumphostInstance{}, false, nil
	}

	if len(instances) == 1 {
		return instances[0], true, nil
	}

	fmt.Println("\nAvailable Jumphost Instances:")
	for i, instance := range instances {
		fmt.Printf("%d. %s (%s)\n", i+1, instance.Name, instance.InstanceID)
	}

	choice := d.getChoice(len(instances))
	if choice == -1 {
		return models.JumphostInstance{}, false, nil
	}

	return instances[choice], true, nil
}

func (d *Display) SelectService(services []models.Service) (models.Service, bool) {
	if len(services) == 0 {
		fmt.Println("No services available")
		return models.Service{}, false
	}

	if len(services) == 1 {
		return services[0], true
	}

	fmt.Println("\nAvailable Services:")
	for i, service := range services {
		fmt.Printf("%d. %s\n", i+1, service.Name())
	}

	choice := d.getChoice(len(services))
	if choice == -1 {
		return models.Service{}, false
	}

	return services[choice], true
}

func (d *Display) SelectTarget(
	service models.Service,
	loader func() ([]models.ConnectionParams, error)) (models.ConnectionParams, bool) {
	targets, err := loader()
	if err != nil {
		d.PrintError("Failed to load targets", err)
		return models.ConnectionParams{}, false
	}

	if len(targets) == 0 {
		fmt.Println("No targets available")
		return models.ConnectionParams{}, false
	}

	if len(targets) == 1 {
		return targets[0], true
	}

	fmt.Printf("\nAvailable %s Targets:\n", service.Name())
	for i, target := range targets {
		fmt.Printf("%d. %s\n", i+1, target.DisplayName())
	}

	choice := d.getChoice(len(targets))
	if choice == -1 {
		return models.ConnectionParams{}, false
	}

	return targets[choice], true
}

func (d *Display) SelectRegion(regions []models.Region) models.Region {
	if len(regions) == 0 {
		fmt.Println("No regions available")
		return models.Region{}
	}

	if len(regions) == 1 {
		return regions[0]
	}

	fmt.Println("\nAvailable Regions:")
	for i, region := range regions {
		fmt.Printf("%d. %s\n", i+1, region.Name)
	}

	choice := d.getChoice(len(regions))
	if choice == -1 {
		return models.Region{}
	}

	return regions[choice]
}

func (d *Display) SelectLocalPort(port int) int {
	fmt.Printf("\nDefault local port: %d\n", port)
	fmt.Print("Enter a different port or press Enter to use default: ")

	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)

	if input == "" {
		return port
	}

	newPort, err := strconv.Atoi(input)
	if err != nil || newPort < 1 || newPort > 65535 {
		fmt.Println("Invalid port, using default")
		return port
	}

	return newPort
}

func (d *Display) Loading(message string, loader func() (any, error)) (any, error) {
	fmt.Printf("%s...\n", message)
	return loader()
}

func (d *Display) StartJumphost(jumphost models.JumphostInstance, info models.ConnectionParams) error {
	fmt.Println("\n========================================")
	fmt.Println("Starting Jumphost Connection")
	fmt.Println("========================================")
	fmt.Printf("Jumphost: %s (%s)\n", jumphost.Name, jumphost.InstanceID)
	fmt.Printf("Target: %s\n", info.DisplayName())
	fmt.Printf("Local Port: %d\n", info.LocalPort)
	fmt.Printf("Remote Endpoint: %s:%d\n", info.Hostname(), info.Port())
	fmt.Println("========================================")
	fmt.Println("\nPress Ctrl+C to stop the connection")

	return jumphost.Start(context.Background(), info)
}

func (d *Display) PrintErrorf(format string, a ...any) {
	fmt.Fprintf(os.Stderr, "ERROR: "+format+"\n", a...)
}

func (d *Display) Print(message string) {
	fmt.Println(message)
}

func (d *Display) PrintError(message string, err error) {
	fmt.Fprintf(os.Stderr, "ERROR: %s: %v\n", message, err)
}

// getChoice prompts the user to select an option and returns the zero-based index
func (d *Display) getChoice(max int) int {
	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Printf("\nSelect an option (1-%d) or 'q' to quit: ", max)
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)

		if input == "q" || input == "Q" {
			return -1
		}

		choice, err := strconv.Atoi(input)
		if err == nil && choice >= 1 && choice <= max {
			return choice - 1
		}

		fmt.Printf("Invalid input. Please enter a number between 1 and %d\n", max)
	}
}
