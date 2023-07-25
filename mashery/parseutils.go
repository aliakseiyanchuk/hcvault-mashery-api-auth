package mashery

import (
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/aliakseiyanchuk/mashery-v3-go-client/transport"
	"github.com/hashicorp/vault/sdk/framework"
	"net/url"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"
)

func copyStringFieldIfDefined(d *framework.FieldData, fld string, dest *string) {
	if v, ok := d.GetOk(fld); ok {
		if val, ok := v.(string); ok {
			if len(val) > 0 {
				*dest = val
			}
		}
	}
}

func copyIntFieldIfDefined(d *framework.FieldData, fld string, dest *int) {
	if v, ok := d.GetOk(fld); ok {
		if val, ok := v.(int); ok {
			*dest = val
		}
	}
}

func copyBooleanFieldIfDefined(d *framework.FieldData, fld string, dest *bool) {
	if v, ok := d.GetOk(fld); ok {
		if val, ok := v.(bool); ok {
			*dest = val
		}
	}
}

func formatOptionalSecretValue(str []byte) string {
	if len(str) == 0 {
		return ""
	} else {
		hash := sha256.New()
		hash.Write(str)

		return fmt.Sprintf("sha256:%s", hex.EncodeToString(hash.Sum(nil)))
	}
}

func formatTLSPinningOption(opt int) string {
	switch opt {
	case TLSPinningDefault:
		return tlsPinningDefaultOpt
	case TLSPinningSystem:
		return tlsPinningSystemOpt
	case TLSPinningCustom:
		return tlsPinningCustomOpt
	case TLSPinningInsecure:
		return tlsPinningInsecureOpt
	default:
		return strconv.Itoa(opt)
	}
}

func formatRootCA(cfg *BackendConfiguration) string {
	if len(cfg.TLSCerts) == 0 || cfg.TLSCerts == "-" {
		return "---- use system ----"
	} else {
		pool := x509.NewCertPool()
		if !pool.AppendCertsFromPEM([]byte(cfg.TLSCerts)) {
			return "---- CAN NOT BE PARSED! ----"
		} else {
			return fmt.Sprintf("---- custom root CA certificates ----")
		}
	}
}

func formatCertPin(pin transport.TLSCertChainPin) string {
	if pin.IsEmpty() {
		return ""
	} else {
		rv := strings.Builder{}
		appendNoEmpty(pin.CommonName, "cn", &rv)
		appendNoEmpty(hex.EncodeToString(pin.SerialNumber), "sn", &rv)
		appendNoEmpty(hex.EncodeToString(pin.Fingerprint), "fp", &rv)

		return rv.String()
	}
}

func appendNoEmpty(val string, key string, builder *strings.Builder) {
	if len(val) > 0 {
		if builder.Len() > 0 {
			builder.WriteString(", ")
		}
		builder.WriteString(key)
		builder.WriteString("=")
		builder.WriteString(val)
	}
}

type StringConsumer func(string) error

func consumeStringFieldIfDefined(d *framework.FieldData, fld string, consumer StringConsumer) error {
	if v, ok := d.GetOk(fld); ok {
		if str, ok := v.(string); ok {
			return consumer(str)
		}
	}

	return nil
}

func stringKeyOf(dat map[string]interface{}, key string) (string, error) {
	val := dat[key]
	if val != nil {
		if str, ok := val.(string); ok {
			return str, nil
		} else {
			return "", errors.New(fmt.Sprintf("key `%s` is not string", key))
		}
	} else {
		return "", errors.New(fmt.Sprintf("missing key `%s` in input object", key))
	}
}

func intKeyOf(dat map[string]interface{}, key string) (int, error) {
	val := dat[key]
	if val != nil {
		if iVal, ok := val.(int); ok {
			return iVal, nil
		} else if fVal, ok := val.(float64); ok {
			return int(fVal), nil
		} else {
			return -1, errors.New(fmt.Sprintf("key `%s` is not a recognizable number, but %s", key, reflect.TypeOf(val)))
		}
	} else {
		return -1, errors.New(fmt.Sprintf("missing key `%s` in input object", key))
	}
}

func buildQueryString(d *framework.FieldData, params ...string) url.Values {
	vals := url.Values{}

	for i := range params {
		k := params[i]

		if v, ok := d.GetOk(k); ok {
			if mv, ok := v.(string); ok {
				vals[k] = []string{mv}
			} else if mv, ok := v.(int); ok {
				vals[k] = []string{strconv.Itoa(mv)}
			} else if mv, ok := v.([]string); ok {
				vals[k] = []string{strings.Join(mv, ",")}
			}
		}
	}

	return vals
}

var dayPattern *regexp.Regexp
var weekPattern *regexp.Regexp
var datePattern *regexp.Regexp

func init() {
	dayPattern = regexp.MustCompile("(\\d+)d")
	weekPattern = regexp.MustCompile("(\\d+)w")
	datePattern = regexp.MustCompile("(\\d{4})-(\\d{1,2})-(\\d{1,2})")
}

// ParseUserInputDuration parses multiple formats of the user input duration
// - days: 30d
// - weeks: 6w
// - explicit date in year-month-format, 2023-01-01
// - any valid go-language duration
// The method returns the duration object if any valid input exists.
func ParseUserInputDuration(suppliedInput string) (time.Duration, error) {

	if dayPattern.MatchString(suppliedInput) {
		m := dayPattern.FindStringSubmatch(suppliedInput)
		days, _ := strconv.Atoi(m[1])
		return time.Hour * 24 * time.Duration(days), nil
	} else if weekPattern.MatchString(suppliedInput) {
		m := weekPattern.FindStringSubmatch(suppliedInput)
		weeks, _ := strconv.Atoi(m[1])
		return time.Hour * 24 * 7 * time.Duration(weeks), nil
	} else if datePattern.MatchString(suppliedInput) {
		m := datePattern.FindStringSubmatch(suppliedInput)
		year, _ := strconv.Atoi(m[1])
		month, _ := strconv.Atoi(m[2])
		date, _ := strconv.Atoi(m[3])

		supDate := time.Date(year, time.Month(month), date, 0, 0, 0, 0, time.UTC)

		return supDate.Sub(time.Now()), nil
	}

	// Otherwise, it's a go-lang duration format. I'll let it parse as-is.
	return time.ParseDuration(suppliedInput)
}
