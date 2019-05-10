package aorm

type KeyStringSerial struct {
	ID string `gorm:"size:24;primary_key;AUTO_INCREMENT"`
}

func (p *KeyStringSerial) GetID() string {
	return p.ID
}

func (p *KeyStringSerial) SetID(value string) {
	p.ID = value
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
