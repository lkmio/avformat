package libflv

import (
	"github.com/lkmio/avformat/libbufio"
	"math"
)

func ReadAMF0String(buffer libbufio.BytesReader) (string, error) {
	size, err := buffer.ReadUint16()
	if err != nil {
		return "", err
	}

	bytes, err := buffer.ReadBytes(int(size))
	if err != nil {
		return "", err
	}

	return string(bytes), nil
}

func ReadAMF0LongString(buffer libbufio.BytesReader) (string, error) {
	size, err := buffer.ReadUint32()
	if err != nil {
		return "", err
	}

	bytes, err := buffer.ReadBytes(int(size))
	if err != nil {
		return "", err
	}

	return string(bytes), nil
}

func ReadObjectProperties(buffer libbufio.BytesReader) (*AMF0Object, error) {
	object := &AMF0Object{}
	for buffer.ReadableBytes() >= 3 {
		endMark, _ := buffer.ReadUint24()
		if uint32(AMF0DataTyeObjectEnd) == endMark {
			return object, nil
		}

		_ = buffer.SeekBack(3)
		key, err := ReadAMF0String(buffer)
		if err != nil {
			return nil, err
		}

		value, err := ReadAMF0Element(buffer)
		if err != nil {
			return nil, err
		}

		object.AddProperty(key, value)
	}

	return object, nil
}

func ReadAMF0Element(buffer libbufio.BytesReader) (AMF0Element, error) {
	marker, err := buffer.ReadUint8()
	if err != nil {
		return nil, err
	}

	switch DataType(marker) {
	case AMF0DataTypeNumber:
		number, err := buffer.ReadUint64()
		if err != nil {
			return nil, err
		}

		return AMF0Number(math.Float64frombits(number)), nil
	case AMF0DataTypeBoolean:
		value, err := buffer.ReadUint8()
		if err != nil {
			return nil, err
		}

		if 0 == value {
			return AMF0Boolean(false), nil
		} else {
			return AMF0Boolean(true), nil
		}
	case AMF0DataTypeString:
		amf0String, err := ReadAMF0String(buffer)
		if err != nil {
			return nil, err
		}

		return AMF0String(amf0String), nil
	case AMF0DataTypeObject:
		return ReadObjectProperties(buffer)
	case AMF0DataTypeMovieClip:
		println("skip reserved field MovieClip")
		return nil, nil
	case AMF0DataTypeNull:
		return AMF0Null{}, nil
	case AMF0DataTypeUnDefined:
		return AMF0Undefined{}, nil
	case AMF0DataTypeReference:
		// 引用元素索引
		index, err := buffer.ReadUint32()
		if err != nil {
			return nil, err
		}

		return AMF0Reference(index), nil
	case AMF0DataTypeECMAArray:
		// count *(object-property)
		_, err := buffer.ReadUint32()
		if err != nil {
			return nil, err
		}
		//for i := 0; i < count; i++ {
		//
		//}

		object, err := ReadObjectProperties(buffer)
		if err != nil {
			return nil, err
		}

		return &AMF0ECMAArray{object}, nil
	case AMF0DataTypeStrictArray:
		// array-count *(value-type)
		var array []AMF0Element
		count, err := buffer.ReadUint32()
		if err != nil {
			return nil, err
		}

		for i := 0; i < int(count); i++ {
			element, err := ReadAMF0Element(buffer)
			if err != nil {
				return nil, err
			}

			array = append(array, element)
		}

		return AMF0StrictArray(array), nil
	case AMF0DataTypeDate:

		zone, err := buffer.ReadUint16()
		if err != nil {
			return nil, err
		}

		date, err := buffer.ReadUint64()
		if err != nil {
			return nil, err
		}

		return AMF0Date{zone, math.Float64frombits(date)}, nil
	case AMF0DataTypeLongString:
		longString, err := ReadAMF0LongString(buffer)
		if err != nil {
			return nil, err
		}

		return AMF0LongString(longString), nil
	case AMF0DataTypeUnsupported, AMF0DataTypeRecordSet:
		return nil, nil
	case AMF0DataTypeXMLDocument:
		// The XML document type is always encoded as a long UTF-8 string.
		longString, err := ReadAMF0LongString(buffer)
		if err != nil {
			return nil, err
		}

		return AMF0XMLDocument{AMF0LongString(longString)}, nil
	case AMF0DataTypeTypedObject:
		className, err := ReadAMF0String(buffer)
		if err != nil {
			return nil, err
		}

		properties, err := ReadObjectProperties(buffer)
		return AMF0TypedObject{className, properties}, nil
	case AMF0DataTypeSwitchTOAMF3:
		return nil, nil
	}

	return nil, nil
}
