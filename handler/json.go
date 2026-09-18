package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
)

const maxJSONBodySize = 1 << 20 // 1 MiB

// decodeJSONBody decodes exactly one JSON object from the request body.
//
// It rejects:
//   - empty request bodies
//   - malformed JSON
//   - unknown JSON fields
//   - multiple JSON values in one request
func decodeJSONBody(
	w http.ResponseWriter,
	r *http.Request,
	dst interface{},
) error {
	if r.Body == nil {
		return errors.New("request body is required")
	}

	decoder := json.NewDecoder(
		http.MaxBytesReader(
			w,
			r.Body,
			maxJSONBodySize,
		),
	)

	// Reject fields that are not part of the request structure.
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(dst); err != nil {
		if errors.Is(err, io.EOF) {
			return errors.New("request body is required")
		}

		var syntaxError *json.SyntaxError
		if errors.As(err, &syntaxError) {
			return errors.New("invalid JSON")
		}

		var typeError *json.UnmarshalTypeError
		if errors.As(err, &typeError) {
			return fmt.Errorf(
				"invalid value for field %s",
				typeError.Field,
			)
		}

		return err
	}

	// Ensure there is no second JSON value after the first one.
	var extra interface{}

	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			return errors.New(
				"request body must contain exactly one JSON object",
			)
		}

		return errors.New("invalid JSON")
	}

	return nil
}
