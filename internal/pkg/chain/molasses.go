package chain

import (
//"fmt"
)

type Molasses struct {
	name     string
	producer Producer
}

func NewMolasses(p Producer) *Molasses {
	return &Molasses{name: "Molasses", producer: p}
}

func (m *Molasses) Name() string {
	return m.name
}

func (m *Molasses) Producer() Producer {
	return m.producer
}
