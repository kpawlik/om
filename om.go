package om

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"sync"
)

type OrderedMap struct {
	mutex sync.RWMutex
	Map  map[string]interface{}
	Keys []string
}

// Create a new OrderedMap
func NewOrderedMap() *OrderedMap {
	return &OrderedMap{
		Map:  make(map[string]interface{}),
		Keys: []string{},
		mutex: sync.RWMutex{},
	}
}


func (om *OrderedMap) SetChild (key string, child *OrderedMap){
	om.mutex.Lock()
	defer om.mutex.Unlock()
	om.Map[key] = child
	om.Keys = append(om.Keys, key)
}

func (om *OrderedMap) CreateChild (key string) *OrderedMap{
	om.mutex.Lock()
	defer om.mutex.Unlock()
	child := NewOrderedMap()
	om.Map[key] = child
	om.Keys = append(om.Keys, key)
	return child
}

func (om *OrderedMap) ParseObject(dec *json.Decoder) (err error) {
	var t json.Token
	var value interface{}
	for dec.More() {
		t, err = dec.Token()
		if err != nil {
			return err
		}

		key, ok := t.(string)
		if !ok {
			return fmt.Errorf("expecting JSON key should be always a string: %T: %v", t, t)
		}

		t, err = dec.Token()
		if err == io.EOF {
			break
		} else if err != nil {
			return err
		}

		value, err = HandleDelim(t, dec)
		if err != nil {
			return err
		}
		om.Map[key] = value
		om.Keys = append(om.Keys, key)
	}
	t, err = dec.Token()
	if err != nil {
		return err
	}
	if delim, ok := t.(json.Delim); !ok || delim != '}' {
		return fmt.Errorf("expect JSON object close with '}'")
	}

	return nil
}

// this implements type json.Unmarshaler interface, so can be called in json.Unmarshal(data, om)
func (om *OrderedMap) UnmarshalJSON(data []byte) error {
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber()

	// must open with a delim token '{'
	t, err := dec.Token()
	if err != nil {
		return err
	}
	if delim, ok := t.(json.Delim); !ok || delim != '{' {
		return fmt.Errorf("expect JSON object open with '{'")
	}

	err = om.ParseObject(dec)
	if err != nil {
		return err
	}

	t, err = dec.Token()
	if err != io.EOF {
		return fmt.Errorf("expect end of JSON object but got more token: %T: %v or err: %v", t, t, err)
	}

	return nil
}

// this implements type json.Marshaler interface, so can be called in json.Marshal(om)
func (om *OrderedMap) MarshalJSON() (res []byte, err error) {
	lines := make([]string, len(om.Keys))
	for i, key := range om.Keys {
		var b []byte
		b, err = json.Marshal(om.Map[key])
		if err != nil {
			return
		}
		lines[i] = fmt.Sprintf("\"%s\": %s", key, string(b))
	}
	res = append(res, '{')
	res = append(res, []byte(strings.Join(lines, ","))...)
	res = append(res, '}')
	return
}


// this implements type json.Marshaler interface, so can be called in json.Marshal(om)
func (om *OrderedMap) MarshalIndent(indent string) (res []byte, err error) {
	if res, err = om.MarshalJSON(); err!=nil{
		return
	}
	buff := bytes.NewBuffer([]byte{})
	if err = json.Indent(buff, res, "", indent); err != nil{
		return
	}
	res = buff.Bytes()
	return
}

func ParseArray(dec *json.Decoder) (arr []interface{}, err error) {
	var t json.Token
	arr = make([]interface{}, 0)
	for dec.More() {
		t, err = dec.Token()
		if err != nil {
			return
		}

		var value interface{}
		value, err = HandleDelim(t, dec)
		if err != nil {
			return
		}
		arr = append(arr, value)
	}
	t, err = dec.Token()
	if err != nil {
		return
	}
	if delim, ok := t.(json.Delim); !ok || delim != ']' {
		err = fmt.Errorf("expect JSON array close with ']'")
		return
	}

	return
}

func HandleDelim(t json.Token, dec *json.Decoder) (res interface{}, err error) {
	if delim, ok := t.(json.Delim); ok {
		switch delim {
		case '{':
			om2 := NewOrderedMap()
			err = om2.ParseObject(dec)
			if err != nil {
				return
			}
			return om2, nil
		case '[':
			var value []interface{}
			value, err = ParseArray(dec)
			if err != nil {
				return
			}
			return value, nil
		default:
			return nil, fmt.Errorf("unexpected delimiter: %q", delim)
		}
	}
	return t, nil
}
