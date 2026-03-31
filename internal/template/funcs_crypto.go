package template

import (
	"crypto/hmac"
	"crypto/md5"
	"crypto/rand"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"

	"github.com/golang-jwt/jwt/v5"
	"github.com/spf13/cast"
)

func (e *Engine) registerCryptoFuncs() {
	e.funcs["md5"] = func(s interface{}) string {
		h := md5.New()
		h.Write([]byte(cast.ToString(s)))
		return hex.EncodeToString(h.Sum(nil))
	}

	e.funcs["sha256"] = func(s interface{}) string {
		h := sha256.New()
		h.Write([]byte(cast.ToString(s)))
		return hex.EncodeToString(h.Sum(nil))
	}

	e.funcs["sha512"] = func(s interface{}) string {
		h := sha512.New()
		h.Write([]byte(cast.ToString(s)))
		return hex.EncodeToString(h.Sum(nil))
	}

	e.funcs["hmacSHA256"] = func(key, s interface{}) string {
		h := hmac.New(sha256.New, []byte(cast.ToString(key)))
		h.Write([]byte(cast.ToString(s)))
		return hex.EncodeToString(h.Sum(nil))
	}

	e.funcs["jwt"] = func(tokenString interface{}) (map[string]interface{}, error) {
		token, _, err := new(jwt.Parser).ParseUnverified(cast.ToString(tokenString), jwt.MapClaims{})
		if err != nil {
			return nil, err
		}
		if claims, ok := token.Claims.(jwt.MapClaims); ok {
			return claims, nil
		}
		return nil, nil
	}

	e.funcs["randomBytes"] = func(n interface{}) ([]byte, error) {
		b := make([]byte, cast.ToInt(n))
		if _, err := rand.Read(b); err != nil {
			return nil, err
		}
		return b, nil
	}
}
