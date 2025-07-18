package decode

import "reflect"

func makeDefaultMarshaller(rt reflect.Type) unmarshaller {
	kind := rt.Kind()
	switch kind {
	case reflect.String:
		return decodeString
	case reflect.Bool:

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:

	case reflect.Float32, reflect.Float64:

	case reflect.Map:

	case reflect.Struct:

	case reflect.Slice:

	case reflect.Array:

	case reflect.Pointer:

	case reflect.Interface:
	default:
		
	}
	return nil
}

func decodeString(first byte, rv reflect.Value, d *Decoder) error {
	
}
func decodeBool(first byte, rv reflect.Value, d *Decoder) error {

}
func decodeNumber(first byte, rv reflect.Value, d *Decoder) error {

}
/* func decodeString(first byte, rv reflect.Value, d *Decoder) error {

} */