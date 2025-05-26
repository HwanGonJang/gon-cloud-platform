package network

import (
	"fmt"
	"net"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// OVSManager interface defines methods for managing Open vSwitch
type OVSManager interface {
	// Bridge management
	CreateBridge(name string, cidr string) error
	DeleteBridge(name string) error
	ListBridges() ([]Bridge, error)
	BridgeExists(name string) (bool, error)

	// Port management
	AddPort(bridgeName, portName, portType string) error
	DeletePort(bridgeName, portName string) error
	ListPorts(bridgeName string) ([]Port, error)

	// Flow management
	AddFlow(bridgeName string, flow Flow) error
	DeleteFlow(bridgeName string, flow Flow) error
	ListFlows(bridgeName string) ([]Flow, error)

	// VLAN management
	SetPortVLAN(bridgeName, portName string, vlan int) error
	GetPortVLAN(bridgeName, portName string) (int, error)

	// Utility methods
	GetBridgeInfo(name string) (*BridgeInfo, error)
	SetController(bridgeName, controller string) error
}

// Bridge represents an OVS bridge
type Bridge struct {
	Name     string    `json:"name"`
	UUID     string    `json:"uuid"`
	DataPath string    `json:"datapath"`
	Ports    []string  `json:"ports"`
	Created  time.Time `json:"created"`
}

// Port represents an OVS port
type Port struct {
	Name      string            `json:"name"`
	UUID      string            `json:"uuid"`
	Interface string            `json:"interface"`
	VLAN      int               `json:"vlan"`
	Type      string            `json:"type"`
	Options   map[string]string `json:"options"`
}

// Flow represents an OVS flow rule
type Flow struct {
	Priority    int    `json:"priority"`
	Match       string `json:"match"`
	Actions     string `json:"actions"`
	Table       int    `json:"table"`
	IdleAge     int    `json:"idle_age"`
	HardAge     int    `json:"hard_age"`
	Cookie      string `json:"cookie"`
	PacketCount int64  `json:"packet_count"`
	ByteCount   int64  `json:"byte_count"`
}

// BridgeInfo contains detailed information about a bridge
type BridgeInfo struct {
	Name       string            `json:"name"`
	UUID       string            `json:"uuid"`
	DataPath   string            `json:"datapath"`
	Controller string            `json:"controller"`
	Ports      []Port            `json:"ports"`
	Options    map[string]string `json:"options"`
	Status     string            `json:"status"`
}

// ovsManager implements OVSManager interface
type ovsManager struct {
	timeout time.Duration
}

// NewOVSManager creates a new OVS manager instance
func NewOVSManager() OVSManager {
	return &ovsManager{
		timeout: 30 * time.Second,
	}
}

// CreateBridge creates a new OVS bridge with specified CIDR
func (m *ovsManager) CreateBridge(name string, cidr string) error {
	// Validate bridge name
	if err := m.validateBridgeName(name); err != nil {
		return fmt.Errorf("invalid bridge name: %w", err)
	}

	// Check if bridge already exists
	exists, err := m.BridgeExists(name)
	if err != nil {
		return fmt.Errorf("failed to check bridge existence: %w", err)
	}
	if exists {
		return fmt.Errorf("bridge %s already exists", name)
	}

	// Parse CIDR to get network info
	_, ipNet, err := net.ParseCIDR(cidr)
	if err != nil {
		return fmt.Errorf("invalid CIDR: %w", err)
	}

	// Create bridge
	cmd := exec.Command("ovs-vsctl", "add-br", name)
	if err := m.runCommand(cmd); err != nil {
		return fmt.Errorf("failed to create bridge: %w", err)
	}

	// Set bridge as up
	cmd = exec.Command("ip", "link", "set", "dev", name, "up")
	if err := m.runCommand(cmd); err != nil {
		// Try to cleanup the bridge
		m.DeleteBridge(name)
		return fmt.Errorf("failed to bring bridge up: %w", err)
	}

	// Configure bridge IP (use first IP in the range as bridge IP)
	bridgeIP := m.getFirstIP(ipNet)
	if bridgeIP != "" {
		cmd = exec.Command("ip", "addr", "add", fmt.Sprintf("%s/%d", bridgeIP, m.getPrefixLength(ipNet)), "dev", name)
		if err := m.runCommand(cmd); err != nil {
			// Log warning but don't fail
			// In production, you might want to use a proper logger
			fmt.Printf("Warning: failed to set bridge IP: %v\n", err)
		}
	}

	// Set bridge options for better performance and security
	bridgeOptions := map[string]string{
		"stp_enable":                "false",
		"mcast_snooping":            "false",
		"rstp_enable":               "false",
		"protocols":                 "OpenFlow13",
		"fail_mode":                 "secure",
		"other_config:forward-bpdu": "false",
	}

	for key, value := range bridgeOptions {
		cmd = exec.Command("ovs-vsctl", "set", "bridge", name, fmt.Sprintf("%s=%s", key, value))
		if err := m.runCommand(cmd); err != nil {
			fmt.Printf("Warning: failed to set bridge option %s: %v\n", key, err)
		}
	}

	return nil
}

// DeleteBridge removes an OVS bridge
func (m *ovsManager) DeleteBridge(name string) error {
	// Check if bridge exists
	exists, err := m.BridgeExists(name)
	if err != nil {
		return fmt.Errorf("failed to check bridge existence: %w", err)
	}
	if !exists {
		return nil // Bridge doesn't exist, nothing to do
	}

	// Delete bridge
	cmd := exec.Command("ovs-vsctl", "del-br", name)
	if err := m.runCommand(cmd); err != nil {
		return fmt.Errorf("failed to delete bridge: %w", err)
	}

	return nil
}

// ListBridges returns all OVS bridges
func (m *ovsManager) ListBridges() ([]Bridge, error) {
	cmd := exec.Command("ovs-vsctl", "list-br")
	output, err := m.runCommandWithOutput(cmd)
	if err != nil {
		return nil, fmt.Errorf("failed to list bridges: %w", err)
	}

	bridgeNames := strings.Fields(strings.TrimSpace(output))
	bridges := make([]Bridge, 0, len(bridgeNames))

	for _, name := range bridgeNames {
		bridge, err := m.getBridgeDetails(name)
		if err != nil {
			// Log error but continue with other bridges
			fmt.Printf("Warning: failed to get details for bridge %s: %v\n", name, err)
			continue
		}
		bridges = append(bridges, *bridge)
	}

	return bridges, nil
}

// BridgeExists checks if a bridge exists
func (m *ovsManager) BridgeExists(name string) (bool, error) {
	cmd := exec.Command("ovs-vsctl", "br-exists", name)
	err := m.runCommand(cmd)
	if err == nil {
		return true, nil
	}

	// Check if it's a "bridge not found" error vs other errors
	if exitError, ok := err.(*exec.ExitError); ok {
		if exitError.ExitCode() == 2 {
			return false, nil // Bridge doesn't exist
		}
	}

	return false, fmt.Errorf("failed to check bridge existence: %w", err)
}

// AddPort adds a port to a bridge
func (m *ovsManager) AddPort(bridgeName, portName, portType string) error {
	// Check if bridge exists
	exists, err := m.BridgeExists(bridgeName)
	if err != nil {
		return fmt.Errorf("failed to check bridge existence: %w", err)
	}
	if !exists {
		return fmt.Errorf("bridge %s does not exist", bridgeName)
	}

	var cmd *exec.Cmd
	switch portType {
	case "internal":
		cmd = exec.Command("ovs-vsctl", "add-port", bridgeName, portName, "--", "set", "interface", portName, "type=internal")
	case "veth":
		cmd = exec.Command("ovs-vsctl", "add-port", bridgeName, portName)
	case "patch":
		return fmt.Errorf("patch ports require additional configuration, use AddPatchPort method")
	default:
		cmd = exec.Command("ovs-vsctl", "add-port", bridgeName, portName)
	}

	if err := m.runCommand(cmd); err != nil {
		return fmt.Errorf("failed to add port %s to bridge %s: %w", portName, bridgeName, err)
	}

	return nil
}

// DeletePort removes a port from a bridge
func (m *ovsManager) DeletePort(bridgeName, portName string) error {
	cmd := exec.Command("ovs-vsctl", "del-port", bridgeName, portName)
	if err := m.runCommand(cmd); err != nil {
		// Check if port doesn't exist
		if strings.Contains(err.Error(), "no port named") {
			return nil // Port doesn't exist, nothing to do
		}
		return fmt.Errorf("failed to delete port %s from bridge %s: %w", portName, bridgeName, err)
	}

	return nil
}

// ListPorts returns all ports on a bridge
func (m *ovsManager) ListPorts(bridgeName string) ([]Port, error) {
	cmd := exec.Command("ovs-vsctl", "list-ports", bridgeName)
	output, err := m.runCommandWithOutput(cmd)
	if err != nil {
		return nil, fmt.Errorf("failed to list ports for bridge %s: %w", bridgeName, err)
	}

	portNames := strings.Fields(strings.TrimSpace(output))
	ports := make([]Port, 0, len(portNames))

	for _, name := range portNames {
		port, err := m.getPortDetails(bridgeName, name)
		if err != nil {
			fmt.Printf("Warning: failed to get details for port %s: %v\n", name, err)
			continue
		}
		ports = append(ports, *port)
	}

	return ports, nil
}

// AddFlow adds a flow rule to a bridge
func (m *ovsManager) AddFlow(bridgeName string, flow Flow) error {
	flowSpec := m.buildFlowSpec(flow)
	cmd := exec.Command("ovs-ofctl", "add-flow", bridgeName, flowSpec)
	if err := m.runCommand(cmd); err != nil {
		return fmt.Errorf("failed to add flow to bridge %s: %w", bridgeName, err)
	}

	return nil
}

// DeleteFlow removes a flow rule from a bridge
func (m *ovsManager) DeleteFlow(bridgeName string, flow Flow) error {
	flowSpec := m.buildFlowMatchSpec(flow)
	cmd := exec.Command("ovs-ofctl", "del-flows", bridgeName, flowSpec)
	if err := m.runCommand(cmd); err != nil {
		return fmt.Errorf("failed to delete flow from bridge %s: %w", bridgeName, err)
	}

	return nil
}

// ListFlows returns all flow rules for a bridge
func (m *ovsManager) ListFlows(bridgeName string) ([]Flow, error) {
	cmd := exec.Command("ovs-ofctl", "dump-flows", bridgeName)
	output, err := m.runCommandWithOutput(cmd)
	if err != nil {
		return nil, fmt.Errorf("failed to list flows for bridge %s: %w", bridgeName, err)
	}

	return m.parseFlows(output), nil
}

// SetPortVLAN sets VLAN tag for a port
func (m *ovsManager) SetPortVLAN(bridgeName, portName string, vlan int) error {
	cmd := exec.Command("ovs-vsctl", "set", "port", portName, fmt.Sprintf("tag=%d", vlan))
	if err := m.runCommand(cmd); err != nil {
		return fmt.Errorf("failed to set VLAN %d for port %s: %w", vlan, portName, err)
	}

	return nil
}

// GetPortVLAN gets VLAN tag for a port
func (m *ovsManager) GetPortVLAN(bridgeName, portName string) (int, error) {
	cmd := exec.Command("ovs-vsctl", "get", "port", portName, "tag")
	output, err := m.runCommandWithOutput(cmd)
	if err != nil {
		return 0, fmt.Errorf("failed to get VLAN for port %s: %w", portName, err)
	}

	vlanStr := strings.TrimSpace(output)
	if vlanStr == "[]" || vlanStr == "" {
		return 0, nil // No VLAN tag
	}

	vlan, err := strconv.Atoi(vlanStr)
	if err != nil {
		return 0, fmt.Errorf("invalid VLAN value: %s", vlanStr)
	}

	return vlan, nil
}

// GetBridgeInfo returns detailed information about a bridge
func (m *ovsManager) GetBridgeInfo(name string) (*BridgeInfo, error) {
	exists, err := m.BridgeExists(name)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, fmt.Errorf("bridge %s does not exist", name)
	}

	// Get bridge UUID
	cmd := exec.Command("ovs-vsctl", "get", "bridge", name, "_uuid")
	uuidOutput, err := m.runCommandWithOutput(cmd)
	if err != nil {
		return nil, fmt.Errorf("failed to get bridge UUID: %w", err)
	}

	// Get datapath ID
	cmd = exec.Command("ovs-vsctl", "get", "bridge", name, "datapath_id")
	datapathOutput, err := m.runCommandWithOutput(cmd)
	if err != nil {
		datapathOutput = ""
	}

	// Get controller
	cmd = exec.Command("ovs-vsctl", "get", "bridge", name, "controller")
	controllerOutput, err := m.runCommandWithOutput(cmd)
	if err != nil {
		controllerOutput = ""
	}

	// Get ports
	ports, err := m.ListPorts(name)
	if err != nil {
		ports = []Port{}
	}

	return &BridgeInfo{
		Name:       name,
		UUID:       strings.TrimSpace(uuidOutput),
		DataPath:   strings.TrimSpace(datapathOutput),
		Controller: strings.TrimSpace(controllerOutput),
		Ports:      ports,
		Status:     "active",
	}, nil
}

// SetController sets the OpenFlow controller for a bridge
func (m *ovsManager) SetController(bridgeName, controller string) error {
	cmd := exec.Command("ovs-vsctl", "set-controller", bridgeName, controller)
	if err := m.runCommand(cmd); err != nil {
		return fmt.Errorf("failed to set controller for bridge %s: %w", bridgeName, err)
	}

	return nil
}

// Helper methods

// validateBridgeName validates bridge name format
func (m *ovsManager) validateBridgeName(name string) error {
	if len(name) == 0 {
		return fmt.Errorf("bridge name cannot be empty")
	}
	if len(name) > 15 {
		return fmt.Errorf("bridge name cannot exceed 15 characters")
	}

	// Check for valid characters (alphanumeric, hyphen, underscore)
	validName := regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)
	if !validName.MatchString(name) {
		return fmt.Errorf("bridge name can only contain alphanumeric characters, hyphens, and underscores")
	}

	return nil
}

// runCommand executes a command with timeout
func (m *ovsManager) runCommand(cmd *exec.Cmd) error {
	// Set timeout
	timer := time.AfterFunc(m.timeout, func() {
		cmd.Process.Kill()
	})
	defer timer.Stop()

	return cmd.Run()
}

// runCommandWithOutput executes a command and returns output
func (m *ovsManager) runCommandWithOutput(cmd *exec.Cmd) (string, error) {
	// Set timeout
	timer := time.AfterFunc(m.timeout, func() {
		cmd.Process.Kill()
	})
	defer timer.Stop()

	output, err := cmd.Output()
	return string(output), err
}

// getFirstIP returns the first usable IP in a network
func (m *ovsManager) getFirstIP(ipNet *net.IPNet) string {
	ip := ipNet.IP.Mask(ipNet.Mask)
	// Increment by 1 to get first usable IP
	for i := len(ip) - 1; i >= 0; i-- {
		ip[i]++
		if ip[i] != 0 {
			break
		}
	}
	return ip.String()
}

// getPrefixLength returns the prefix length of a network
func (m *ovsManager) getPrefixLength(ipNet *net.IPNet) int {
	ones, _ := ipNet.Mask.Size()
	return ones
}

// getBridgeDetails gets detailed information about a bridge
func (m *ovsManager) getBridgeDetails(name string) (*Bridge, error) {
	// Get UUID
	cmd := exec.Command("ovs-vsctl", "get", "bridge", name, "_uuid")
	uuidOutput, err := m.runCommandWithOutput(cmd)
	if err != nil {
		return nil, err
	}

	// Get datapath ID
	cmd = exec.Command("ovs-vsctl", "get", "bridge", name, "datapath_id")
	datapathOutput, err := m.runCommandWithOutput(cmd)
	if err != nil {
		datapathOutput = ""
	}

	// Get ports
	cmd = exec.Command("ovs-vsctl", "list-ports", name)
	portsOutput, err := m.runCommandWithOutput(cmd)
	if err != nil {
		portsOutput = ""
	}

	ports := strings.Fields(strings.TrimSpace(portsOutput))

	return &Bridge{
		Name:     name,
		UUID:     strings.TrimSpace(uuidOutput),
		DataPath: strings.TrimSpace(datapathOutput),
		Ports:    ports,
		Created:  time.Now(), // This would need to be retrieved from OVS DB for accuracy
	}, nil
}

// getPortDetails gets detailed information about a port
func (m *ovsManager) getPortDetails(bridgeName, portName string) (*Port, error) {
	// Get port UUID
	cmd := exec.Command("ovs-vsctl", "get", "port", portName, "_uuid")
	uuidOutput, err := m.runCommandWithOutput(cmd)
	if err != nil {
		return nil, err
	}

	// Get VLAN
	vlan, _ := m.GetPortVLAN(bridgeName, portName)

	// Get interface type
	cmd = exec.Command("ovs-vsctl", "get", "interface", portName, "type")
	typeOutput, err := m.runCommandWithOutput(cmd)
	if err != nil {
		typeOutput = "normal"
	}

	return &Port{
		Name:      portName,
		UUID:      strings.TrimSpace(uuidOutput),
		Interface: portName,
		VLAN:      vlan,
		Type:      strings.Trim(strings.TrimSpace(typeOutput), "\""),
		Options:   make(map[string]string),
	}, nil
}

// buildFlowSpec builds flow specification string
func (m *ovsManager) buildFlowSpec(flow Flow) string {
	spec := ""

	if flow.Table > 0 {
		spec += fmt.Sprintf("table=%d,", flow.Table)
	}

	if flow.Priority > 0 {
		spec += fmt.Sprintf("priority=%d,", flow.Priority)
	}

	if flow.Match != "" {
		spec += flow.Match + ","
	}

	if flow.Actions != "" {
		spec += "actions=" + flow.Actions
	}

	return strings.TrimSuffix(spec, ",")
}

// buildFlowMatchSpec builds flow match specification for deletion
func (m *ovsManager) buildFlowMatchSpec(flow Flow) string {
	spec := ""

	if flow.Table > 0 {
		spec += fmt.Sprintf("table=%d,", flow.Table)
	}

	if flow.Priority > 0 {
		spec += fmt.Sprintf("priority=%d,", flow.Priority)
	}

	if flow.Match != "" {
		spec += flow.Match
	}

	return strings.TrimSuffix(spec, ",")
}

// parseFlows parses ovs-ofctl dump-flows output
func (m *ovsManager) parseFlows(output string) []Flow {
	lines := strings.Split(output, "\n")
	flows := make([]Flow, 0)

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "NXST_FLOW") {
			continue
		}

		flow := m.parseFlowLine(line)
		if flow != nil {
			flows = append(flows, *flow)
		}
	}

	return flows
}

// parseFlowLine parses a single flow line
func (m *ovsManager) parseFlowLine(line string) *Flow {
	// This is a simplified parser - in production you'd want more robust parsing
	flow := &Flow{}

	// Extract priority
	if idx := strings.Index(line, "priority="); idx >= 0 {
		priorityStr := line[idx+9:]
		if comma := strings.Index(priorityStr, ","); comma >= 0 {
			priorityStr = priorityStr[:comma]
		}
		if priority, err := strconv.Atoi(priorityStr); err == nil {
			flow.Priority = priority
		}
	}

	// Extract actions
	if idx := strings.Index(line, "actions="); idx >= 0 {
		flow.Actions = line[idx+8:]
	}

	return flow
}
