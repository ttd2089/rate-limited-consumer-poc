package config

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParse(t *testing.T) {

	t.Run("returns error when target is slice", func(t *testing.T) {
		_, err := Parse[[]string](nil)
		assert.Error(t, err)
	})

	t.Run("returns error when target is map", func(t *testing.T) {
		// TODO: Support this case.
		_, err := Parse[map[string]string](nil)
		assert.Error(t, err)
	})

	t.Run("returns error when target is pointer to map", func(t *testing.T) {
		// TODO: Support this case.
		_, err := Parse[*map[string]string](nil)
		assert.Error(t, err)
	})

	t.Run("returns error when target is pointer to pointer to struct", func(t *testing.T) {
		_, err := Parse[**struct{}](nil)
		assert.Error(t, err)
	})

	t.Run("populates primative struct fields", func(t *testing.T) {
		type target struct {
			String  string  `config_key:"string"`
			Bool    bool    `config_key:"bool"`
			Int     int     `config_key:"int"`
			Uint    uint    `config_key:"uint"`
			Int8    int8    `config_key:"int8"`
			Int16   int16   `config_key:"int16"`
			Int32   int32   `config_key:"int32"`
			Int64   int64   `config_key:"int64"`
			Uint8   uint8   `config_key:"uint8"`
			Uint16  uint16  `config_key:"uint16"`
			Uint32  uint32  `config_key:"uint32"`
			Uint64  uint64  `config_key:"uint64"`
			Float32 float32 `config_key:"float32"`
			Float64 float64 `config_key:"float64"`
		}
		expected := target{
			String:  "forty-two",
			Bool:    true,
			Int:     -42,
			Uint:    42,
			Int8:    -13,
			Int16:   -1337,
			Int32:   -326000,
			Int64:   -8306623063,
			Uint8:   13,
			Uint16:  1337,
			Uint32:  326000,
			Uint64:  8306623063,
			Float32: 1.618,
			Float64: 0.5772156649,
		}
		config := StdMap(map[string]string{
			"string":  expected.String,
			"bool":    strconv.FormatBool(expected.Bool),
			"int":     strconv.Itoa(expected.Int),
			"uint":    strconv.FormatUint(uint64(expected.Uint), 10),
			"int8":    strconv.FormatInt(int64(expected.Int8), 10),
			"int16":   strconv.FormatInt(int64(expected.Int16), 10),
			"int32":   strconv.FormatInt(int64(expected.Int32), 10),
			"int64":   strconv.FormatInt(expected.Int64, 10),
			"uint8":   strconv.FormatUint(uint64(expected.Uint8), 10),
			"uint16":  strconv.FormatUint(uint64(expected.Uint16), 10),
			"uint32":  strconv.FormatUint(uint64(expected.Uint32), 10),
			"uint64":  strconv.FormatUint(expected.Uint64, 10),
			"float32": strconv.FormatFloat(float64(expected.Float32), 'f', -1, 32),
			"float64": strconv.FormatFloat(expected.Float64, 'f', -1, 64),
		})

		t.Run("on struct", func(t *testing.T) {
			actual, err := Parse[target](config)
			assert.NoError(t, err)
			assert.Equal(t, expected, actual)
		})

		t.Run("on pointer to struct", func(t *testing.T) {
			actual, err := Parse[*target](config)
			assert.NoError(t, err)
			assert.Equal(t, expected, *actual)
		})
	})
}
