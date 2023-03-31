package manifest

import (
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/buger/jsonparser"
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

func parseManifestArray(r io.Reader) (manifest, error) {
	m := manifest{}
	data, err := io.ReadAll(r)
	if err != nil {
		return m, errors.Wrap(err, "reading manifest reader")
	}

	jsonparser.ArrayEach(data, func(value []byte, dataType jsonparser.ValueType, offset int, err error) {

		e, errInner := getEntry(value)
		if errInner != nil {
			fmt.Printf("Error decoding input2: %v\n", errInner)
			return
		}

		m.Entries = append(m.Entries, e)

	}, "entries")

	return m, nil
}

func getEntry(data []byte) (*manifestEntry, error) {
	e := &manifestEntry{}

	paths := [][]string{
		[]string{"id"},
		[]string{"labels"},
		[]string{"modified"},
		[]string{"deleted"},
		[]string{"data"},
	}

	failed := false

	jsonparser.EachKey(data, func(idx int, value []byte, vt jsonparser.ValueType, err error) {
		switch idx {
		case 0:
			e.ID = ID(value)
		case 1:
			err := json.Unmarshal(value, &e.Labels)
			if err != nil {
				failed = true
				// return fmt.Errorf("unmarshalling labels: %w", err)
			}
		case 2:
			e.ModTime, err = time.Parse(time.RFC3339, string(value))
			if err != nil {
				failed = true
				// return fmt.Errorf("unmarshalling modtime: %w", err)
			}
		case 3:
			err := json.Unmarshal(value, &e.Deleted)
			if err != nil {
				failed = true
				// return fmt.Errorf("unmarshalling deleted: %w", err)
			}
		case 4:
			e.Content = make([]byte, len(value))
			n := copy(e.Content, value)
			if n != len(value) {
				failed = true
				fmt.Printf("Failed to copy content %d\n", n)
			}
		default:
			fmt.Printf("Unexpected Input: %v\n", idx)
			failed = true
			// return errors.New("Unexpected Input: " + string(key))
		}
		return
	}, paths...)

	if failed {
		return nil, errors.New("Failed to unmarshal entry")
	}
	return e, nil
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
