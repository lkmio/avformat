package libflv

//@https://en.wikipedia.org/wiki/Action_Message_Format
//@https://rtmp.veriskope.com/pdf/amf0-file-format-specification.pdf

import (
	"github.com/yangjiechina/avformat/libbufio"
	"math"
)

type dataType byte

const (
	AMF0DataTypeNumber       = dataType(0x00)
	AMF0DataTypeBoolean      = dataType(0x01)
	AMF0DataTypeString       = dataType(0x02)
	AMF0DataTypeObject       = dataType(0x03)
	AMF0DataTypeMovieClip    = dataType(0x04)
	AMF0DataTypeNull         = dataType(0x05)
	AMF0DataTypeUnDefined    = dataType(0x06)
	AMF0DataTypeReference    = dataType(0x07)
	AMF0DataTypeECMAArray    = dataType(0x08)
	AMF0DataTyeObjectEnd     = dataType(0x09)
	AMF0DataTypeStrictArray  = dataType(0x0A)
	AMF0DataTypeDate         = dataType(0x0B)
	AMF0DataTypeLongString   = dataType(0x0C)
	AMF0DataTypeUnsupported  = dataType(0x0D)
	AMF0DataTypeRecordSet    = dataType(0x0E)
	AMF0DataTypeXMLDocument  = dataType(0x0F)
	AMF0DataTypeTypedObject  = dataType(0x10)
	AMF0DataTypeSwitchTOAMF3 = dataType(0x11)
)

func ReadAMF0String(buffer libbufio.ByteBuffer) (string, error) {
	if err := buffer.PeekCount(2); err != nil {
		return "", err
	}
	count := int(buffer.ReadUInt16())
	if err := buffer.PeekCount(count); err != nil {
		return "", err
	}
	return string(buffer.ReadBytesWithShallowCopy(count)), nil
}

func ReadAMF0LongString(buffer libbufio.ByteBuffer) (string, error) {
	if err := buffer.PeekCount(4); err != nil {
		return "", err
	}
	count := int(buffer.ReadUInt32())
	if err := buffer.PeekCount(count); err != nil {
		return "", err
	}
	return string(buffer.ReadBytesWithShallowCopy(count)), nil
}

func ReadAMF0(buffer libbufio.ByteBuffer) (interface{}, error) {
	if err := buffer.PeekCount(1); err != nil {
		return nil, err
	}
	t := buffer.ReadUInt8()

	switch dataType(t) {
	case AMF0DataTypeNumber:
		if err := buffer.PeekCount(8); err != nil {
			return nil, err
		}
		return math.Float64frombits(buffer.ReadUInt64()), nil
	case AMF0DataTypeBoolean:
		if err := buffer.PeekCount(1); err != nil {
			return nil, err
		}
		return buffer.ReadUInt8(), nil
	case AMF0DataTypeString:
		return ReadAMF0String(buffer)
	case AMF0DataTypeObject:
		m := make(map[string]interface{}, 5)

		for buffer.ReadableBytes() > 3 {
			if buffer.PeekUInt24() == uint32(AMF0DataTyeObjectEnd) {
				buffer.Skip(3)
				return m, nil
			}
			key, err := ReadAMF0String(buffer)
			if err != nil {
				return nil, err
			}
			value, err := ReadAMF0(buffer)
			if err != nil {
				return nil, err
			}
			m[key] = value
		}
		return m, nil
	case AMF0DataTypeMovieClip, AMF0DataTypeNull, AMF0DataTypeUnDefined, AMF0DataTypeReference:
		//reserved
		return nil, nil
	case AMF0DataTypeECMAArray, AMF0DataTypeStrictArray:
		if err := buffer.PeekCount(4); err != nil {
			return nil, err
		}
		count := int(buffer.ReadUInt32())
		m := make(map[string]interface{}, count)
		for buffer.ReadableBytes() > 3 {
			if int24 := buffer.PeekUInt24(); int24 == uint32(AMF0DataTyeObjectEnd) {
				buffer.Skip(3)
				return m, nil
			}
			key, err := ReadAMF0String(buffer)
			if err != nil {
				return nil, err
			}
			value, err := ReadAMF0(buffer)
			if err != nil {
				return nil, err
			}
			m[key] = value
		}
		return m, nil
	case AMF0DataTypeDate:
		if err := buffer.PeekCount(8); err != nil {
			return nil, err
		}

		uInt64 := buffer.ReadUInt64()
		//LocalDateTimeOffset
		if err := buffer.PeekCount(2); err != nil {
			return nil, err
		}
		_ = buffer.ReadUInt16()
		return math.Float64frombits(uInt64), nil
	case AMF0DataTypeLongString:
		return ReadAMF0LongString(buffer)
	case AMF0DataTypeUnsupported, AMF0DataTypeRecordSet:
		return nil, nil
	case AMF0DataTypeXMLDocument:
		//var count int
		if err := buffer.PeekCount(4); err != nil {
			return nil, err
		}

		uInt32 := buffer.ReadUInt32()
		count := int(uInt32)
		bytes := make([]byte, count)
		buffer.ReadBytes(bytes)
		return bytes, nil
	case AMF0DataTypeTypedObject, AMF0DataTypeSwitchTOAMF3:
		return nil, nil
	}

	return nil, nil
}

func DoReadAMF0FromBuffer(buffer libbufio.ByteBuffer) ([]interface{}, error) {

	var result []interface{}
	for buffer.ReadableBytes() > 3 {
		if int24 := buffer.PeekUInt24(); int24 == uint32(AMF0DataTyeObjectEnd) {
			buffer.Skip(3)
			return result, nil
		}

		value, err := ReadAMF0(buffer)
		if err != nil {
			return nil, err
		}
		result = append(result, value)
	}

	return result, nil
}

func DoReadAMF0(data []byte) ([]interface{}, error) {
	return DoReadAMF0FromBuffer(libbufio.NewByteBuffer(data))
}
