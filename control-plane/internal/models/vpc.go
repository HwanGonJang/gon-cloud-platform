package models

import (
	"time"
)

type VPC struct {
	ID          string    `json:"id" db:"id"`
	Name        string    `json:"name" db:"name"`
	CIDRBlock   string    `json:"cidr_block" db:"cidr_block"`
	Description *string   `json:"description" db:"description"`
	UserID      string    `json:"user_id" db:"user_id"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}

type Subnet struct {
	ID               string    `json:"id" db:"id"`
	VPCID            string    `json:"vpc_id" db:"vpc_id"`
	Name             string    `json:"name" db:"name"`
	CIDRBlock        string    `json:"cidr_block" db:"cidr_block"`
	AvailabilityZone string    `json:"availability_zone" db:"availability_zone"`
	IsPublic         bool      `json:"is_public" db:"is_public"`
	CreatedAt        time.Time `json:"created_at" db:"created_at"`
	UpdatedAt        time.Time `json:"updated_at" db:"updated_at"`
}

type RouteTable struct {
	ID        string    `json:"id" db:"id"`
	VPCID     string    `json:"vpc_id" db:"vpc_id"`
	Name      string    `json:"name" db:"name"`
	IsMain    bool      `json:"is_main" db:"is_main"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

type Route struct {
	ID              string `json:"id" db:"id"`
	RouteTableID    string `json:"route_table_id" db:"route_table_id"`
	DestinationCIDR string `json:"destination_cidr" db:"destination_cidr"`
	TargetType      string `json:"target_type" db:"target_type"` // igw, nat, instance
	TargetID        string `json:"target_id" db:"target_id"`
	Priority        int    `json:"priority" db:"priority"`
}

type InternetGateway struct {
	ID        string    `json:"id" db:"id"`
	VPCID     string    `json:"vpc_id" db:"vpc_id"`
	Name      string    `json:"name" db:"name"`
	State     string    `json:"state" db:"state"` // available, attached, detached
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}
