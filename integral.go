package primegraphcore

import (
	"bytes"
	"regexp"
	"strconv"
	"strings"
)

// OpenAPI defines `integer` by value: a document may write the whole number 100
// as `100`, `100.0` or `1e2`, while `1.5` names no integer. encoding/json
// refuses the last two for an integer field, so the nullable integer wrappers
// respell such a value as its plain digits before decoding it.

// jsonNumberPattern is a number token exactly as JSON spells one.
var jsonNumberPattern = regexp.MustCompile(`^-?(?:0|[1-9][0-9]*)(?:\.[0-9]+)?(?:[eE][+-]?[0-9]+)?$`)

// integralExponentBound keeps the digit arithmetic below in range; an exponent
// past it names a value no int64 holds, or one that is not whole.
const integralExponentBound = 1 << 30

// integralNumber is the plain digits of a JSON number that carries a fraction
// or an exponent and names a whole int64. Anything else — an ordinary integer,
// a fraction, a value past int64, null, a string, a malformed token — is
// returned unchanged, so the decode reports it exactly as before.
func integralNumber(src []byte) []byte {
	token := string(bytes.TrimSpace(src))
	if !strings.ContainsAny(token, ".eE") || !jsonNumberPattern.MatchString(token) {
		return src
	}
	digits, ok := wholeInt64Digits(token)
	if !ok {
		return src
	}
	return []byte(digits)
}

// wholeInt64Digits works the value out from the text itself — sign, digits,
// fraction, exponent — so no value passes through a float64 and an int64 keeps
// every digit.
func wholeInt64Digits(token string) (string, bool) {
	sign := ""
	if strings.HasPrefix(token, "-") {
		sign = "-"
		token = token[1:]
	}
	mantissa, exponent := token, 0
	if at := strings.IndexAny(token, "eE"); at >= 0 {
		parsed, err := strconv.Atoi(token[at+1:])
		if err != nil {
			return "", false
		}
		mantissa, exponent = token[:at], parsed
	}
	whole, fraction, _ := strings.Cut(mantissa, ".")
	digits := strings.TrimLeft(whole+fraction, "0")
	significant := strings.TrimRight(digits, "0")
	if significant == "" {
		return sign + "0", true
	}
	if exponent > integralExponentBound || exponent < -integralExponentBound {
		return "", false
	}
	scale := exponent - len(fraction) + len(digits) - len(significant)
	if scale < 0 || len(significant)+scale > 19 {
		return "", false
	}
	text := sign + significant + strings.Repeat("0", scale)
	if _, err := strconv.ParseInt(text, 10, 64); err != nil {
		return "", false
	}
	return text, true
}
