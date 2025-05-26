package models

import (
	"time"
)

type SecurityGroup struct {
	ID            string              `json:"id" db:"id"`
	Name          string              `json:"name" db:"name"`
	Description   string              `json:"description" db:"description"`
	VPCID         string              `json:"vpc_id" db:"vpc_id"`
	UserID        string              `json:"user_id" db:"user_id"`
	InboundRules  []SecurityGroupRule `json:"inbound_rules" db:"-"`
	OutboundRules []SecurityGroupRule `json:"outbound_rules" db:"-"`
	CreatedAt     time.Time           `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time           `json:"updated_at" db:"updated_at"`
}

type SecurityGroupRule struct {
	ID              string `json:"id" db:"id"`
	SecurityGroupID string `json:"security_group_id" db:"security_group_id"`
	Direction       string `json:"direction" db:"direction"` // inbound, outbound
	Protocol        string `json:"protocol" db:"protocol"`   // tcp, udp, icmp, all
	FromPort        int    `json:"from_port" db:"from_port"`
	ToPort          int    `json:"to_port" db:"to_port"`
	Source          string `json:"source" db:"source"` // CIDR block or security group ID
	Description     string `json:"description" db:"description"`
}
