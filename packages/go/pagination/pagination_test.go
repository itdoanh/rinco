package pagination

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEncodeDecodeCursor(t *testing.T) {
	type payload struct {
		ID        string `json:"id"`
		CreatedAt int64  `json:"t"`
	}
	in := payload{ID: "abc", CreatedAt: 1700000000}
	cur, err := EncodeCursor(in)
	require.NoError(t, err)
	assert.False(t, cur.Empty())

	var out payload
	require.NoError(t, DecodeCursor(cur, &out))
	assert.Equal(t, in, out)

	// Tampered cursor (flip first char to detect tampering)
	bad := Cursor("Z" + string(cur)[1:])
	var out2 payload
	assert.Error(t, DecodeCursor(bad, &out2))

	// Empty
	assert.Error(t, DecodeCursor("", &out2))

	// Encode error (unsupported type)
	_, err = EncodeCursor(make(chan int))
	assert.Error(t, err)
}

func TestPageDefaultsAndClamp(t *testing.T) {
	p := NewPage(-5, 0)
	assert.Equal(t, 0, p.Offset)
	assert.Equal(t, DefaultLimit, p.Limit)

	p = NewPage(10, 99999)
	assert.Equal(t, 10, p.Offset)
	assert.Equal(t, MaxLimit, p.Limit)

	p = NewPage(20, 50)
	assert.Equal(t, 20, p.Offset)
	assert.Equal(t, 50, p.Limit)
	assert.False(t, p.HasMore())

	p.Total = 100
	p.Limit = 25
	assert.True(t, p.HasMore())
	assert.Equal(t, 45, p.NextOffset())
}

func TestParsePageFromValues(t *testing.T) {
	v := url.Values{}
	v.Set("offset", "10")
	v.Set("limit", "5")
	p := ParsePage(v)
	assert.Equal(t, 10, p.Offset)
	assert.Equal(t, 5, p.Limit)

	v = url.Values{}
	v.Set("limit", "0") // default
	p = ParsePage(v)
	assert.Equal(t, DefaultLimit, p.Limit)
}

func TestPageOffsetClause(t *testing.T) {
	p := NewPage(20, 10)
	clause := p.OffsetClause(1)
	assert.Equal(t, "OFFSET $1 LIMIT $2", clause)
	args := p.Args()
	assert.Equal(t, []any{20, 10}, args)
}

func TestCursorPage(t *testing.T) {
	p := NewCursorPage("abc", 0, false)
	assert.Equal(t, DefaultLimit, p.Limit)
	assert.False(t, p.Desc)

	p = NewCursorPage("abc", 99999, true)
	assert.Equal(t, MaxLimit, p.Limit)
	assert.True(t, p.Desc)

	v := url.Values{}
	v.Set("cursor", "xyz")
	v.Set("limit", "15")
	v.Set("order", "asc")
	cp := ParseCursorPage(v)
	assert.Equal(t, Cursor("xyz"), cp.Cursor)
	assert.Equal(t, 15, cp.Limit)
	assert.False(t, cp.Desc)

	v.Set("order", "DESC")
	cp = ParseCursorPage(v)
	assert.True(t, cp.Desc)
}

func TestResponse(t *testing.T) {
	items := []string{"a", "b", "c"}
	resp := NewResponse(items, 100, true)
	assert.Equal(t, items, resp.Items)
	assert.True(t, resp.HasMore)
	assert.Equal(t, int64(100), resp.Total)

	resp2 := resp.WithNextCursor("cur")
	assert.Equal(t, Cursor("cur"), resp2.NextCursor)
	// original unchanged
	assert.True(t, resp.NextCursor.Empty())
}

func TestKeysetArgs(t *testing.T) {
	k := Keyset{ID: "id-1", CreatedAt: 1700000000}
	assert.Equal(t, "(created_at, id) < ($1, $2)", k.ClauseDesc())
	assert.Equal(t, []any{int64(1700000000), "id-1"}, k.Args())
}