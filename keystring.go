package aorm

type KeyStringSerial struct {
	ID string `gorm:"size:24;primary_key" serial:"yes"`
}

func (p *KeyStringSerial) GetID() string {
	return p.ID
}

type KeyString struct {
	ID string `gorm:"size:24;primary_key"`
}

func (p *KeyString) GetID() string {
	return p.ID
}

func (p *KeyString) SetID(value string) {
	p.ID = value
}
