package models

import (
	"time"
)

type Instance struct {
	ID             string            `json:"id" db:"id"`
	Name           string            `json:"name" db:"name"`
	InstanceType   string            `json:"instance_type" db:"instance_type"`
	ImageID        string            `json:"image_id" db:"image_id"`
	SubnetID       string            `json:"subnet_id" db:"subnet_id"`
	PrivateIP      string            `json:"private_ip" db:"private_ip"`
	PublicIP       string            `json:"public_ip" db:"public_ip"`
	State          string            `json:"state" db:"state"` // pending, running, stopping, stopped, terminated
	WorkerNodeID   string            `json:"worker_node_id" db:"worker_node_id"`
	UserID         string            `json:"user_id" db:"user_id"`
	KeyPair        string            `json:"key_pair" db:"key_pair"`
	SecurityGroups []string          `json:"security_groups" db:"-"`
	Tags           map[string]string `json:"tags" db:"-"`
	CreatedAt      time.Time         `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time         `json:"updated_at" db:"updated_at"`
}

type InstanceType struct {
	Name    string  `json:"name" db:"name"`
	CPU     int     `json:"cpu" db:"cpu"`
	Memory  int     `json:"memory" db:"memory"`   // MB
	Storage int     `json:"storage" db:"storage"` // GB
	Network string  `json:"network" db:"network"`
	Price   float64 `json:"price" db:"price"`
}

type Image struct {
	ID           string    `json:"id" db:"id"`
	Name         string    `json:"name" db:"name"`
	Description  string    `json:"description" db:"description"`
	OS           string    `json:"os" db:"os"`
	Version      string    `json:"version" db:"version"`
	Architecture string    `json:"architecture" db:"architecture"`
	IsPublic     bool      `json:"is_public" db:"is_public"`
	UserID       string    `json:"user_id" db:"user_id"`
	Size         int64     `json:"size" db:"size"` // bytes
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
}
