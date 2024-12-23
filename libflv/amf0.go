package libflv

import "github.com/lkmio/avformat/libbufio"

//@https://en.wikipedia.org/wiki/Action_Message_Format
//@https://rtmp.veriskope.com/pdf/amf0-file-format-specification.pdf

type DataType byte

const (
	AMF0DataTypeNumber       = DataType(0x00)
	AMF0DataTypeBoolean      = DataType(0x01)
	AMF0DataTypeString       = DataType(0x02)
	AMF0DataTypeObject       = DataType(0x03)
	AMF0DataTypeMovieClip    = DataType(0x04) // 预留字段
	AMF0DataTypeNull         = DataType(0x05)
	AMF0DataTypeUnDefined    = DataType(0x06)
	AMF0DataTypeReference    = DataType(0x07)
	AMF0DataTypeECMAArray    = DataType(0x08)
	AMF0DataTyeObjectEnd     = DataType(0x09)
	AMF0DataTypeStrictArray  = DataType(0x0A)
	AMF0DataTypeDate         = DataType(0x0B)
	AMF0DataTypeLongString   = DataType(0x0C)
	AMF0DataTypeUnsupported  = DataType(0x0D)
	AMF0DataTypeRecordSet    = DataType(0x0E) // 预留字段
	AMF0DataTypeXMLDocument  = DataType(0x0F)
	AMF0DataTypeTypedObject  = DataType(0x10)
	AMF0DataTypeSwitchTOAMF3 = DataType(0x11) // 切换到AMF3
)

type AMF0 struct {
	elements []AMF0Element
}

func (a *AMF0) Marshal(dst []byte) (int, error) {
	return MarshalElements(a.elements, dst)
}

func (a *AMF0) Unmarshal(data []byte) error {
	buffer := libbufio.NewBytesReader(data)

	for buffer.ReadableBytes() > 0 {
		element, err := ReadAMF0Element(buffer)
		if err != nil {
			return err
		}

		a.elements = append(a.elements, element)
	}

	return nil
}

func (a *AMF0) Size() int {
	return len(a.elements)
}

func (a *AMF0) Get(index int) AMF0Element {
	return a.elements[index]
}

func (a *AMF0) Add(value AMF0Element) {
	a.elements = append(a.elements, value)
}

func (a *AMF0) AddString(str string) {
	a.elements = append(a.elements, AMF0String(str))
}
func (a *AMF0) AddNumber(number float64) {
	a.elements = append(a.elements, AMF0Number(number))
}
