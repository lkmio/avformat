package libflv

import (
	"github.com/lkmio/avformat/libbufio"
	"github.com/lkmio/avformat/utils"
	"math"
	"time"
)

const (
	AMF3DataTypeUndefined   = DataType(0x00)
	AMF3DataTypeNULL        = DataType(0x01)
	AMF3DataTypeFalse       = DataType(0x02)
	AMF3DataTypeTrue        = DataType(0x03)
	AMF3DataTypeInteger     = DataType(0x04)
	AMF3DataTypeDouble      = DataType(0x05)
	AMF3DataTypeString      = DataType(0x06)
	AMF3DataTypeXMLDocument = DataType(0x07)
	AMF3DataTypeDate        = DataType(0x08)
	AMF3DataTypeArray       = DataType(0x09)
	AMF3DataTypeObject      = DataType(0x0A)
	AMF3DataTypeXML         = DataType(0x0B)
	AMF3DataTypeByteArray   = DataType(0x0C)
	AMF3DataTypeVector      = DataType(0x0D)
	AMF3DataTypeDictionary  = DataType(0x0E)
)

type AMF3Reader struct {
	strRefTable []string
	objRefTable []string
}

func (d *AMF3Reader) readU29(buffer libbufio.BytesReader) (int, bool, error) {
	var integer int
	for i := 0; i < 4; i++ {
		uInt8, err := buffer.ReadUint8()
		if err != nil {
			return 0, false, err
		}

		integer <<= i * 7
		if i == 3 {
			integer <<= 1
			integer |= int(uInt8)
		} else {
			integer |= int(uInt8 & 0x7)
		}

		if uInt8>>7 == 0 {
			break
		}
	}

	return integer, integer&0x1 == 0, nil
}

func (d *AMF3Reader) findString(index int) (string, error) {
	if index >= len(d.strRefTable) {
		return "", utils.NewSliceBoundsOutOfRangeError(len(d.strRefTable), index)
	}
	return d.strRefTable[index], nil
}

func (d *AMF3Reader) findObject(index int) (string, error) {
	if index >= len(d.objRefTable) {
		return "", utils.NewSliceBoundsOutOfRangeError(len(d.objRefTable), index)
	}
	return d.objRefTable[index], nil
}

func (d *AMF3Reader) ReadObjectFromTable(buffer libbufio.BytesReader) (string, int, bool, error) {
	u29, ref, err := d.readU29(buffer)
	if err != nil {
		return "", -1, false, err
	}

	if ref {
		object, err := d.findObject(u29 >> 1)
		return object, u29, err == nil, err

	}

	return "", u29, false, err
}

func (d *AMF3Reader) readRef(buffer libbufio.BytesReader, readCache func(int) (string, error)) (string, error) {
	u29, ref, err := d.readU29(buffer)
	if err != nil {
		return "", err
	}
	if ref {
		if object, err := readCache(u29 >> 1); err != nil {
			return "", err
		} else {
			return object, nil
		}
	}

	u29 >>= 1
	dst := make([]byte, u29)
	//if readBytes := buffer.ReadBytes(dst); readBytes != u29 {
	//	dst = dst[:u29]
	//}

	return string(dst), err
}

func (d *AMF3Reader) ReadAMF3String(buffer libbufio.BytesReader) (string, error) {
	str, err := d.readRef(buffer, d.findString)
	if err != nil {
		d.strRefTable = append(d.strRefTable, str)
	}
	return str, err
}

func (d *AMF3Reader) ReadAMF3Object(buffer libbufio.BytesReader) (string, error) {
	str, err := d.readRef(buffer, d.findObject)
	if err != nil {
		d.objRefTable = append(d.objRefTable, str)
	}
	return str, err
}

func (d *AMF3Reader) ReadAMF3FromBuffer(buffer libbufio.BytesReader) (interface{}, error) {
	mark, err := buffer.ReadUint8()
	if err != nil {
		return nil, err
	}

	switch DataType(mark) {
	case AMF3DataTypeUndefined:
		return "undefined", nil
	case AMF3DataTypeNULL:
		return "null", nil
	case AMF3DataTypeFalse:
		return "false", nil
	case AMF3DataTypeTrue:
		return "true", nil
	case AMF3DataTypeInteger:
		if u29, _, err := d.readU29(buffer); err != nil {
			return nil, err
		} else {
			return u29, nil
		}
	case AMF3DataTypeDouble:
		value, err := buffer.ReadUint64()
		if err != nil {
			return nil, err
		}

		return math.Float64frombits(value), nil
	case AMF3DataTypeString:
		ref, err := d.ReadAMF3String(buffer)
		if err != nil {
			return nil, err

		}
		return ref, nil
	case AMF3DataTypeXMLDocument, AMF3DataTypeXML, AMF3DataTypeByteArray:
		object, err := d.ReadAMF3Object(buffer)
		if err != nil {
			return nil, err

		}
		return object, nil
	case AMF3DataTypeDate:
		object, _, b, err := d.ReadObjectFromTable(buffer)
		if err != nil {
			return nil, err
		} else if b {
			return object, nil
		}

		f64, err := buffer.ReadUint64()
		if err != nil {
			return nil, err
		}

		dateTime := time.Unix(int64(math.Float64frombits(f64)/1000), 0).UTC().String()
		d.strRefTable = append(d.objRefTable, dateTime)
		return dateTime, nil
	case AMF3DataTypeArray, AMF3DataTypeObject:
		object, _, b, err := d.ReadObjectFromTable(buffer)
		if err != nil {
			return nil, err
		} else if b {
			return object, nil
		}

		_, err = d.ReadAMF3String(buffer)
		if err != nil {
			return nil, err
		}

		objs := make(map[string]interface{}, 5)
		if err = d.DoReadAMF3(buffer, objs); err != nil {
			return nil, err
		}
		return objs, err
	case AMF3DataTypeVector:

		break
	case AMF3DataTypeDictionary:
		break
	}

	return nil, nil
}

func (d *AMF3Reader) DoReadAMF3(buffer libbufio.BytesReader, dst map[string]interface{}) error {
	key, err := d.ReadAMF3String(buffer)
	if err != nil {
		return err
	}

	for buffer.ReadableBytes() > 3 {
		value, err := ReadAMF0Element(buffer)
		if err != nil {
			return err
		}
		dst[key] = value
	}

	return err
}
