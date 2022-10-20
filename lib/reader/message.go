/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package reader

import (
	"infini.sh/framework/core/util"
	"time"
)

type Message struct {
	Ts          time.Time   // timestamp the content was read
	Content     []byte      `json:"content"` // actual content read
	Bytes       int         `json:"bytes"`   // total number of bytes read to generate the message
	Fields      util.MapStr // optional fields that can be added by reader
	Meta        util.MapStr // deprecated
	LineNumbers []int       `json:"line_numbers"` // line numbers of current content
	Offset      int64       `json:"offset"`       // content offset in file
}

// IsEmpty returns true in case the message is empty
// A message with only newline character is counted as an empty message
func (m *Message) IsEmpty() bool {
	// If no Bytes were read, event is empty
	// For empty line Bytes is at least 1 because of the newline char
	if m.Bytes == 0 {
		return true
	}

	// Content length can be 0 because of JSON events. Content and Fields must be empty.
	if len(m.Content) == 0 && len(m.Fields) == 0 {
		return true
	}

	return false
}

// AddFields adds fields to the message.
func (m *Message) AddFields(fields util.MapStr) {
	if fields == nil {
		return
	}

	if m.Fields == nil {
		m.Fields = util.MapStr{}
	}
	m.Fields.Update(fields)
}

// AddFlagsWithKey adds flags to the message with an arbitrary key.
// If the field does not exist, it is created.
func (m *Message) AddFlagsWithKey(key string, flags ...string) error {
	if len(flags) == 0 {
		return nil
	}

	if m.Fields == nil {
		m.Fields = util.MapStr{}
	}

	return util.AddTagsWithKey(m.Fields, key, flags)
}
