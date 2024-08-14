package hclconv

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/alecthomas/hcl"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/pkg/errors"
)

type ListProxy struct {
	List []string
}

func ConvertDynamicAttributeToString(key string, elementValue attr.Value) (string, error) {
	switch elementValue := elementValue.(type) {
	case types.String:
		return elementValue.ValueString(), nil
	case types.Bool:
		return strconv.FormatBool(elementValue.ValueBool()), nil
	case types.Int64:
		return strconv.FormatInt(elementValue.ValueInt64(), 10), nil
	case types.Int32:
		return strconv.FormatInt(int64(elementValue.ValueInt32()), 10), nil
	case types.Float64:
		return strconv.FormatFloat(elementValue.ValueFloat64(), 'e', 4, 64), nil
	case types.Float32:
		return strconv.FormatFloat(float64(elementValue.ValueFloat32()), 'e', 4, 32), nil
	case types.Number:
		bigFloat := elementValue.ValueBigFloat()
		if bigFloat.IsInt() {
			return bigFloat.Text('f', 0), nil
		} else {
			return bigFloat.Text('e', 4), nil
		}
	case types.List:
		result, err := MarshalTFListToHcl(elementValue.Elements())
		if err != nil {
			return "", errors.Wrap(err, "could not convert list to hcl")
		}
		return result, nil
	case types.Set:
		result, err := MarshalTFListToHcl(elementValue.Elements())
		if err != nil {
			return "", errors.Wrap(err, "could not convert set to hcl")
		}
		return result, nil
	case types.Map:
		return "", errors.New(
			fmt.Sprintf("Maps are currently unsupported as variables (key %s)", key))
	case types.Object:
		return "", errors.New(
			fmt.Sprintf("Objects are currently unsupported as variables (key %s)", key))
	default:
		return "", errors.New(
			fmt.Sprintf("Unsupported type for variable %s in object: %s",
				key,
				reflect.TypeOf(elementValue).String()))
	}
}

func MarshalTFListToHcl(elements []attr.Value) (string, error) {
	convertedElements := make([]string, len(elements))
	for i, element := range elements {
		var err error
		convertedElements[i], err = ConvertDynamicAttributeToString(strconv.FormatInt(int64(i), 10), element)
		if err != nil {
			return "", errors.Wrap(err, fmt.Sprintf("could not convert element %d to HCL", i))
		}
	}
	byt, err := hcl.Marshal(&ListProxy{convertedElements})
	if err != nil {
		return "Failed to marshal elements to HCL", err
	}
	hclString := string(byt)
	// Remove the "List = " prefix
	return strings.TrimPrefix(hclString, "List = "), nil
}
