package tablewriter

import (
	"reflect"
	"strconv"
	"strings"
)

type column struct {
	reflect.StructField

	Header string
	MaxLen int
}

func newColumn(field reflect.StructField) column {
	c := column{
		StructField: field,

		Header: columnHeader(field.Name),
		MaxLen: getIntTag(field.Tag, "maxLen"),
	}
	return c
}

func columnHeader(fieldName string) string {
	return strings.ReplaceAll(strings.ToUpper(fieldName), "_", " ")
}

func getIntTag(tag reflect.StructTag, key string) int {
	v := tag.Get(key)
	if v == "" {
		return 0
	}
	i, err := strconv.Atoi(v)
	if err != nil {
		return 0
	}
	return i
}
