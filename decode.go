package bert

import (
	"bytes";
	"encoding/binary";
	"io";
	"io/ioutil";
	"os";
	"reflect";
	"strconv";
)

var ErrBadMagic os.Error = &Error{"bad magic"}
var ErrUnknownType os.Error = &Error{"unknown type"}

func read1(r io.Reader) (int, os.Error) {
	bits, err := ioutil.ReadAll(io.LimitReader(r, 1));
	if err != nil {
		return 0, err
	}

	ui8 := uint8(bits[0]);
	return int(ui8), nil;
}

func read2(r io.Reader) (int, os.Error) {
	bits, err := ioutil.ReadAll(io.LimitReader(r, 2));
	if err != nil {
		return 0, err
	}

	ui16 := binary.BigEndian.Uint16(bits);
	return int(ui16), nil;
}

func read4(r io.Reader) (int, os.Error) {
	bits, err := ioutil.ReadAll(io.LimitReader(r, 4));
	if err != nil {
		return 0, err
	}

	ui32 := binary.BigEndian.Uint32(bits);
	return int(ui32), nil;
}

func readSmallInt(r io.Reader) (int, os.Error) {
	return read1(r)
}

func readInt(r io.Reader) (int, os.Error)	{ return read4(r) }

func readFloat(r io.Reader) (float, os.Error) {
	bits, err := ioutil.ReadAll(io.LimitReader(r, 31));
	if err != nil {
		return 0, err
	}

	// Atof doesn't like trailing 0s
	var i int;
	for i = 0; i < len(bits); i++ {
		if bits[i] == 0 {
			break
		}
	}

	return strconv.Atof(string(bits[0:i]));
}

func readAtom(r io.Reader) (Atom, os.Error) {
	str, err := readString(r);
	return Atom(str), err;
}

func readSmallTuple(r io.Reader) (Term, os.Error) {
	size, err := read1(r);
	if err != nil {
		return nil, err
	}

	tuple := make([]Term, size);

	for i := 0; i < size; i++ {
		term, err := readTag(r);
		if err != nil {
			return nil, err
		}
		switch a := term.(type) {
		case Atom:
			if a == BertAtom {
				return readComplex(r)
			}
		}
		tuple[i] = term;
	}

	return tuple, nil;
}

func readNil(r io.Reader) ([]Term, os.Error) {
	_, err := ioutil.ReadAll(io.LimitReader(r, 1));
	if err != nil {
		return nil, err
	}
	list := make([]Term, 0);
	return list, nil;
}

func readString(r io.Reader) (string, os.Error) {
	size, err := read2(r);
	if err != nil {
		return "", err
	}

	str, err := ioutil.ReadAll(io.LimitReader(r, int64(size)));
	if err != nil {
		return "", err
	}

	return string(str), nil;
}

func readList(r io.Reader) ([]Term, os.Error) {
	size, err := read4(r);
	if err != nil {
		return nil, err
	}

	list := make([]Term, size);

	for i := 0; i < size; i++ {
		term, err := readTag(r);
		if err != nil {
			return nil, err
		}
		list[i] = term;
	}

	read1(r);

	return list, nil;
}

func readBin(r io.Reader) ([]uint8, os.Error) {
	size, err := read4(r);
	if err != nil {
		return []uint8{}, err
	}

	bytes, err := ioutil.ReadAll(io.LimitReader(r, int64(size)));
	if err != nil {
		return []uint8{}, err
	}

	return bytes, nil;
}

func readComplex(r io.Reader) (Term, os.Error) {
	term, err := readTag(r);

	if err != nil {
		return term, err
	}

	switch kind := term.(type) {
	case Atom:
		switch kind {
		case NilAtom:
			return nil, nil
		case TrueAtom:
			return true, nil
		case FalseAtom:
			return false, nil
		}
	}

	return term, nil;
}

func readTag(r io.Reader) (Term, os.Error) {
	tag, err := read1(r);
	if err != nil {
		return nil, err
	}

	switch tag {
	case SmallIntTag:
		return readSmallInt(r)
	case IntTag:
		return readInt(r)
	case SmallBignumTag:
		return nil, ErrUnknownType
	case LargeBignumTag:
		return nil, ErrUnknownType
	case FloatTag:
		return readFloat(r)
	case AtomTag:
		return readAtom(r)
	case SmallTupleTag:
		return readSmallTuple(r)
	case LargeTupleTag:
		return nil, ErrUnknownType
	case NilTag:
		return readNil(r)
	case StringTag:
		return readString(r)
	case ListTag:
		return readList(r)
	case BinTag:
		return readBin(r)
	}

	return nil, ErrUnknownType;
}

func DecodeFrom(r io.Reader) (Term, os.Error) {
	version, err := read1(r);

	if err != nil {
		return nil, err
	}

	// check protocol version
	if version != VersionTag {
		return nil, ErrBadMagic
	}

	return readTag(r);
}

func Decode(data []byte) (Term, os.Error)	{ return DecodeFrom(bytes.NewBuffer(data)) }

func UnmarshalFrom(r io.Reader, val interface{}) (err os.Error) {
	result, _ := DecodeFrom(r);

	value := reflect.NewValue(val).(*reflect.PtrValue).Elem();

	switch v := value.(type) {
	case *reflect.StructValue:
		slice := reflect.NewValue(result).(*reflect.SliceValue);
		for i := 0; i < slice.Len(); i++ {
			e := slice.Elem(i).(*reflect.InterfaceValue).Elem();
			v.Field(i).SetValue(e);
		}
	}

	return nil;
}

func Unmarshal(data []byte, val interface{}) (err os.Error) {
	return UnmarshalFrom(bytes.NewBuffer(data), val)
}

func UnmarshalRequest(r io.Reader) (Request, os.Error) {
	var req Request;

	size, err := read4(r);
	if err != nil {
		return req, err
	}

	err = UnmarshalFrom(io.LimitReader(r, int64(size)), &req);

	return req, err;
}
