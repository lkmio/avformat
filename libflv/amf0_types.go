package libflv

import (
	"encoding/binary"
	"math"
)

type AMF0Element interface {
	Type() DataType

	Marshal(dst []byte) (int, error)
}

type AMF0Null struct {
}

func (a AMF0Null) Type() DataType {
	return AMF0DataTypeNull
}

func (a AMF0Null) Marshal(dst []byte) (int, error) {
	return 0, nil
}

type AMF0Number float64

func (a AMF0Number) Type() DataType {
	return AMF0DataTypeNumber
}

func (a AMF0Number) Marshal(dst []byte) (int, error) {
	binary.BigEndian.PutUint64(dst, math.Float64bits(float64(a)))
	return 8, nil
}

type AMF0Boolean bool

func (a AMF0Boolean) Type() DataType {
	return AMF0DataTypeBoolean
}

func (a AMF0Boolean) Marshal(dst []byte) (int, error) {
	if a {
		dst[0] = 1
	} else {
		dst[0] = 0
	}

	return 1, nil
}

type AMF0Undefined struct {
}

func (a AMF0Undefined) Type() DataType {
	return AMF0DataTypeUnDefined
}

func (a AMF0Undefined) Marshal(dst []byte) (int, error) {
	return 0, nil
}

type AMF0Reference uint16

func (a AMF0Reference) Type() DataType {
	return AMF0DataTypeReference
}

func (a AMF0Reference) Marshal(dst []byte) (int, error) {
	binary.BigEndian.PutUint16(dst, uint16(a))
	return 2, nil
}

type AMF0ECMAArray struct {
	*AMF0Object
}

func (a AMF0ECMAArray) Type() DataType {
	return AMF0DataTypeECMAArray
}

type AMF0StrictArray []AMF0Element

func (s AMF0StrictArray) Type() DataType {
	return AMF0DataTypeStrictArray
}

func (s AMF0StrictArray) Marshal(dst []byte) (int, error) {
	return MarshalElements(s, dst)
}

type AMF0Date struct {
	zone uint16
	date float64
}

func (a AMF0Date) Type() DataType {
	return AMF0DataTypeDate
}

func (a AMF0Date) Marshal(dst []byte) (int, error) {
	binary.BigEndian.PutUint16(dst, a.zone)
	binary.BigEndian.PutUint16(dst[2:], a.zone)
	return 10, nil
}

type AMF0LongString string

func (a AMF0LongString) Type() DataType {
	return AMF0DataTypeLongString
}

func (a AMF0LongString) Marshal(dst []byte) (int, error) {
	length := uint32(len(a))
	binary.BigEndian.PutUint32(dst, length)
	copy(dst[4:], a)
	return int(4 + length), nil
}

type AMF0XMLDocument struct {
	AMF0LongString
}

func (a AMF0XMLDocument) Type() DataType {
	return AMF0DataTypeXMLDocument
}

type AMF0TypedObject struct {
	ClassName string
	*AMF0Object
}

func (a AMF0TypedObject) Type() DataType {
	return AMF0DataTypeTypedObject
}

func (a AMF0TypedObject) Marshal(dst []byte) (int, error) {
	n, err := AMF0String(a.ClassName).Marshal(dst)
	if err != nil {
		return 0, err
	}

	n2, err := a.AMF0Object.Marshal(dst[n:])
	if err != nil {
		return 0, err
	}

	return n + n2, nil
}

type AMF0String string

func (a AMF0String) Type() DataType {
	return AMF0DataTypeString
}

func (a AMF0String) Marshal(dst []byte) (int, error) {
	binary.BigEndian.PutUint16(dst, uint16(len(a)))
	copy(dst[2:], a)
	return 2 + len(a), nil
}

func MarshalElement(element AMF0Element, dst []byte) (int, error) {
	dst[0] = byte(element.Type())
	n, err := element.Marshal(dst[1:])
	if err != nil {
		return 0, err
	}

	return 1 + n, nil
}

func MarshalElements(elements []AMF0Element, dst []byte) (int, error) {
	var length int
	for _, element := range elements {
		n, err := MarshalElement(element, dst[length:])
		if err != nil {
			return 0, err
		}

		length += n
	}

	return length, nil
}
