package manifest

import (
	"encoding/json"
	"io"
	"time"

	"github.com/pkg/errors"
)

type manifest struct {
	Entries []*manifestEntry `json:"entries"`
}

type manifestEntry struct {
	ID      ID                `json:"id"`
	Labels  map[string]string `json:"labels"`
	ModTime time.Time         `json:"modified"`
	Deleted bool              `json:"deleted,omitempty"`
	Content json.RawMessage   `json:"data"`
}

const (
	objectOpen  = "{"
	objectClose = "}"
	arrayOpen   = "["
	arrayClose  = "]"
)

var errEOF = errors.New("unexpected end of input")

func expectDelimToken(dec *json.Decoder, expectedToken string) error {
	t, err := dec.Token()
	if err == io.EOF {
		return errors.WithStack(errEOF)
	} else if err != nil {
		return errors.Wrap(err, "reading JSON token")
	}

	d, ok := t.(json.Delim)
	if !ok {
		return errors.Errorf("unexpected token: (%T) %v", t, t)
	} else if d.String() != expectedToken {
		return errors.Errorf("unexpected token; wanted %s, got %s", expectedToken, d)
	}

	return nil
}

func decodeManifestArray(r io.Reader) (manifest, error) {
	var (
		dec = json.NewDecoder(r)
		res = manifest{}
	)

	if err := expectDelimToken(dec, objectOpen); err != nil {
		return res, err
	}

	// Need to manually decode fields here since we can't reuse the stdlib
	// decoder due to memory issues.
	if err := parseFields(dec, &res); err != nil {
		return res, err
	}

	// Consumes closing object curly brace after we're done. Don't need to check
	// for EOF because json.Decode only guarantees decoding the next JSON item in
	// the stream so this follows that.
	return res, expectDelimToken(dec, objectClose)
}

func parseFields(dec *json.Decoder, res *manifest) error {
	for dec.More() {
		t, err := dec.Token()
		if err == io.EOF {
			return errors.WithStack(errEOF)
		} else if err != nil {
			return errors.Wrap(err, "reading JSON token")
		}

		l, ok := t.(string)
		if !ok {
			return errors.Errorf("unexpected token (%T) %v; wanted field name", t, t)
		}

		// Only have `entries` field right now.
		if l != "entries" {
			return errors.Errorf("unexpected field name %s", l)
		}

		if err = decodeArray(dec, &res.Entries); err != nil {
			return err
		}
	}

	return nil
}

func decodeArray[T any](dec *json.Decoder, output *[]T) error {
	// Consume starting bracket.
	if err := expectDelimToken(dec, arrayOpen); err != nil {
		return err
	}

	// Read elements.
	for dec.More() {
		tmp := *new(T)
		if err := dec.Decode(&tmp); err != nil {
			return errors.Wrap(err, "decoding array element")
		}

		*output = append(*output, tmp)
	}

	// Consume ending bracket.
	return expectDelimToken(dec, arrayClose)
}
