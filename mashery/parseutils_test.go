package mashery

import (
	"errors"
	"github.com/aliakseiyanchuk/mashery-v3-go-client/transport"
	"github.com/hashicorp/vault/sdk/framework"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestCopyStringFieldIfDefined_WillCopy(t *testing.T) {
	d := framework.FieldData{
		Raw: map[string]interface{}{
			"a": "b",
			"i": 45,
		},
		Schema: map[string]*framework.FieldSchema{
			"a": {
				Type: framework.TypeString,
			},
			"c": {
				Type: framework.TypeString,
			},
			"i": {
				Type: framework.TypeInt,
			},
		},
	}

	var rv string
	copyStringFieldIfDefined(&d, "a", &rv)
	assert.Equal(t, "b", rv)

	var miss string
	copyStringFieldIfDefined(&d, "c", &miss)
	assert.True(t, len(miss) == 0)

	var typeCollision string
	copyStringFieldIfDefined(&d, "i", &typeCollision)
	// Nothing should happen
	assert.True(t, len(miss) == 0)
}

func TestCopyBooleanFieldIfDefined_WillCopy(t *testing.T) {
	d := framework.FieldData{
		Raw: map[string]interface{}{
			"a": true,
			"i": 45,
		},
		Schema: map[string]*framework.FieldSchema{
			"a": {
				Type: framework.TypeBool,
			},
			"i": {
				Type: framework.TypeInt,
			},
		},
	}

	var rv bool
	copyBooleanFieldIfDefined(&d, "a", &rv)
	assert.True(t, rv)

	var typeCollision bool
	copyBooleanFieldIfDefined(&d, "i", &typeCollision)
	// Nothing should happen
	assert.False(t, typeCollision)
}

func TestFormatOptionalSecretValue_WillCopy(t *testing.T) {
	assert.Equal(t, "", formatOptionalSecretValue(nil))
	assert.Equal(t, "", formatOptionalSecretValue([]byte{}))
	assert.Equal(t,
		"sha256:ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad",
		formatOptionalSecretValue([]byte("abc")))

}

func TestFormatTLSPinningOption(t *testing.T) {
	assert.Equal(t, "default", formatTLSPinningOption(TLSPinningDefault))
	assert.Equal(t, "system", formatTLSPinningOption(TLSPinningSystem))
	assert.Equal(t, "custom", formatTLSPinningOption(TLSPinningCustom))
	assert.Equal(t, "5", formatTLSPinningOption(5))
}

func TestFormatCertPin_Empty(t *testing.T) {
	assert.Equal(t, "", formatCertPin(transport.TLSCertChainPin{}))
}

func TestFormatCertPin_Filled(t *testing.T) {
	assert.Equal(t, "cn=CN, sn=534e, fp=4650", formatCertPin(transport.TLSCertChainPin{
		CommonName:   "CN",
		SerialNumber: []byte("SN"),
		Fingerprint:  []byte("FP"),
	}))

	assert.Equal(t, "cn=CN, fp=4650", formatCertPin(transport.TLSCertChainPin{
		CommonName:  "CN",
		Fingerprint: []byte("FP"),
	}))
}

func testConsumeString(ptr *string) func(string) error {
	return func(s string) error {
		*ptr = s
		return nil
	}
}

func testConsumeStringWithError() func(string) error {
	return func(s string) error {
		return errors.New("unit-testing")
	}
}

func TestConsumeStringFieldIfDefined_Consume(t *testing.T) {
	d := framework.FieldData{
		Raw: map[string]interface{}{
			"a": "b",
			"i": 45,
		},
		Schema: map[string]*framework.FieldSchema{
			"a": {
				Type: framework.TypeString,
			},
			"c": {
				Type: framework.TypeString,
			},
			"i": {
				Type: framework.TypeInt,
			},
		},
	}

	// The value
	var rv string
	e := consumeStringFieldIfDefined(&d, "a", testConsumeString(&rv))
	assert.Nil(t, e)
	assert.Equal(t, "b", rv)

	e = consumeStringFieldIfDefined(&d, "a", testConsumeStringWithError())
	assert.NotNil(t, e)

	var miss string
	e = consumeStringFieldIfDefined(&d, "c", testConsumeString(&miss))
	assert.Nil(t, e)
	assert.True(t, len(miss) == 0)

	var typeCollision string
	e = consumeStringFieldIfDefined(&d, "i", testConsumeString(&typeCollision))
	// Nothing should happen

	assert.Nil(t, e)
	assert.True(t, len(miss) == 0)
}

func TestBuildQueryString(t *testing.T) {
	d := framework.FieldData{
		Raw: map[string]interface{}{
			"a": "A",
			"b": 45,
			"c": []string{"d", "e", "f"},
		},
		Schema: map[string]*framework.FieldSchema{
			"a": {
				Type: framework.TypeString,
			},
			"b": {
				Type: framework.TypeInt,
			},
			"c": {
				Type: framework.TypeStringSlice,
			},
		},
	}

	rv := buildQueryString(&d, "a", "b", "c")
	assert.Equal(t, "A", rv.Get("a"))
	assert.Equal(t, "45", rv.Get("b"))
	assert.Equal(t, "d,e,f", rv.Get("c"))
}

func TestStringKeyOf(t *testing.T) {
	dat := map[string]interface{}{
		"a": "A",
		"b": 35,
	}

	val, err := stringKeyOf(dat, "a")
	assert.Equal(t, "A", val)
	assert.Nil(t, err)

	val, err = stringKeyOf(dat, "missing")
	assert.Equal(t, "", val)
	assert.Equal(t, "missing key `missing` in input object", err.Error())

	val, err = stringKeyOf(dat, "b")
	assert.Equal(t, "", val)
	assert.Equal(t, "key `b` is not string", err.Error())
}
func TestIntKeyOf(t *testing.T) {
	dat := map[string]interface{}{
		"a":  34,
		"af": float64(34),
		"b":  "string",
	}

	val, err := intKeyOf(dat, "a")
	assert.Equal(t, 34, val)
	assert.Nil(t, err)

	val, err = intKeyOf(dat, "af")
	assert.Equal(t, 34, val)
	assert.Nil(t, err)

	val, err = intKeyOf(dat, "missing")
	assert.Equal(t, -1, val)
	assert.Equal(t, "missing key `missing` in input object", err.Error())

	val, err = intKeyOf(dat, "b")
	assert.Equal(t, -1, val)
	assert.Equal(t, "key `b` is not a recognizable number, but string", err.Error())
}
