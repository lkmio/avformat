package libflv

import (
	"encoding/binary"
	"github.com/lkmio/avformat/libbufio"
)

type AMF0Object struct {
	properties []*AMF0Property
}

func (a *AMF0Object) Type() DataType {
	return AMF0DataTypeObject
}

func (a *AMF0Object) Marshal(dst []byte) (int, error) {
	var length int
	for _, property := range a.properties {
		n, err := property.Marshal(dst[length:])
		if err != nil {
			return 0, err
		}

		length += n
	}

	libbufio.PutUint24(dst[length:], uint32(AMF0DataTyeObjectEnd))
	return length + 3, nil
}

func (a *AMF0Object) AddProperty(name string, value AMF0Element) {
	a.properties = append(a.properties, &AMF0Property{name, value})
}

func (a *AMF0Object) FindProperty(name string) *AMF0Property {
	for _, property := range a.properties {
		if property.Name == name {
			return property
		}
	}

	return nil
}

func (a *AMF0Object) AddStringProperty(name, value string) {
	a.AddProperty(name, AMF0String(value))
}

func (a *AMF0Object) AddNumberProperty(name string, value float64) {
	a.AddProperty(name, AMF0Number(value))
}

type AMF0Property struct {
	Name  string
	Value AMF0Element
}

func (a *AMF0Property) Marshal(dst []byte) (int, error) {
	length := len(a.Name)
	binary.BigEndian.PutUint16(dst, uint16(length))
	copy(dst[2:], a.Name)
	length += 2

	n, err := MarshalElement(a.Value, dst[length:])
	if err != nil {
		return 0, err
	}

	return length + n, nil
}
