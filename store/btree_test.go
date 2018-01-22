package store

import (
	"crypto/rand"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestBTreeCacheGetSet does basic sanity checks on our cache
//
// Other tests should handle deletes, setting same value,
// iterating over ranges, and general fuzzing
func TestBTreeCacheGetSet(t *testing.T) {
	// devnull is a black hole... just to keep our types proper
	devnull := BTreeCacheable{EmptyKVStore{}}

	// base is the root of our data, we can layer on top and
	// all queries should work
	base := devnull.CacheWrap()

	// make sure the btree is empty at start but returns results
	// that are writen to it
	k, v := []byte("french"), []byte("fry")
	assert.Nil(t, base.Get(k))
	assert.False(t, base.Has(k))
	base.Set(k, v)
	assert.Equal(t, v, base.Get(k))
	assert.True(t, base.Has(k))

	// now layer another btree on top and make sure that we get
	// base data
	cache := base.CacheWrap()
	assert.Equal(t, v, cache.Get(k))
	assert.True(t, cache.Has(k))

	// writing more data is only visible in the cache
	k2, v2 := []byte("LA"), []byte("Dodgers")
	assert.Nil(t, cache.Get(k2))
	assert.False(t, cache.Has(k2))
	cache.Set(k2, v2)
	assert.Equal(t, v2, cache.Get(k2))
	assert.Nil(t, base.Get(k2))
	assert.True(t, cache.Has(k2))
	assert.False(t, base.Has(k2))

	// we can write the cache to the base layer...
	cache.Write()
	assert.Equal(t, v, base.Get(k))
	assert.Equal(t, v2, base.Get(k2))
	assert.True(t, base.Has(k))
	assert.True(t, base.Has(k2))

	// we can discard one
	k3, v3 := []byte("Bayern"), []byte("Munich")
	c2 := base.CacheWrap()
	assert.Equal(t, v, c2.Get(k))
	assert.Equal(t, v2, c2.Get(k2))
	c2.Set(k3, v3)
	c2.Discard()

	// and commit another
	c3 := base.CacheWrap()
	assert.Equal(t, v, c3.Get(k))
	assert.Equal(t, v2, c3.Get(k2))
	c3.Delete(k)
	c3.Write()

	// make sure it commits proper
	assert.Nil(t, base.Get(k))
	assert.Equal(t, v2, base.Get(k2))
	assert.Nil(t, base.Get(k3))

	// and to test devnull....
	base.Write()
	assert.Nil(t, devnull.Get(k2))
}

// TestBTreeCacheConflicts checks that we can handle
// overwriting values and deleting underlying values
func TestBTreeCacheConflicts(t *testing.T) {
	// devnull is a black hole... just to keep our types proper
	devnull := BTreeCacheable{EmptyKVStore{}}

	// make 10 keys and 20 values....
	ks := randKeys(10, 16)
	vs := randKeys(20, 40)

	cases := [...]struct {
		parentOps     []op
		childOps      []op
		parentQueries []Model // Key is what we query, Value is what we espect
		childQueries  []Model // Key is what we query, Value is what we espect
	}{
		// overwrite one, delete another, add a third
		0: {
			[]op{setOp(ks[1], vs[1]), setOp(ks[2], vs[2])},
			[]op{setOp(ks[1], vs[11]), setOp(ks[3], vs[7]), delOp(ks[2])},
			[]Model{pair(ks[1], vs[1]), pair(ks[2], vs[2]), pair(ks[3], nil)},
			[]Model{pair(ks[1], vs[11]), pair(ks[2], nil), pair(ks[3], vs[7])},
		},
	}

	for i, tc := range cases {
		parent := devnull.CacheWrap()
		for _, op := range tc.parentOps {
			op.apply(parent)
		}

		child := parent.CacheWrap()
		for _, op := range tc.childOps {
			op.apply(child)
		}

		// now check the parent is unaffected
		for j, q := range tc.parentQueries {
			res := parent.Get(q.Key)
			assert.Equal(t, q.Value, res, "%d / %d", i, j)
			has := parent.Has(q.Key)
			assert.Equal(t, q.Value != nil, has, "%d / %d", i, j)
		}

		// the child shows changes
		for j, q := range tc.childQueries {
			res := child.Get(q.Key)
			assert.Equal(t, q.Value, res, "%d / %d", i, j)
			has := child.Has(q.Key)
			assert.Equal(t, q.Value != nil, has, "%d / %d", i, j)
		}

		// write child to parent and make sure it also shows proper data
		child.Write()
		for j, q := range tc.childQueries {
			res := parent.Get(q.Key)
			assert.Equal(t, q.Value, res, "%d / %d", i, j)
			has := parent.Has(q.Key)
			assert.Equal(t, q.Value != nil, has, "%d / %d", i, j)
		}
	}
}

// TestBTreeCacheIterator tests iterating over ranges that
// span both the parent and child caches, combining different
// values, overwrites, and deletes
func TestBTreeCacheIterator(t *testing.T) {
}

// randKeys returns a slice of count keys, all of length
func randKeys(count, length int) [][]byte {
	res := make([][]byte, count)
	for i := 0; i < count; i++ {
		res[i] = randBytes(length)
	}
	return res
}

func randBytes(length int) []byte {
	res := make([]byte, length)
	rand.Read(res)
	return res
}
