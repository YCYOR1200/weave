package coins

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type cmp int

const (
	neg  cmp = -1
	zero     = 0
	pos      = 1
)

func TestCompareCoin(t *testing.T) {

	cases := []struct {
		a      Coin
		b      Coin
		expect int
		aState cmp
		bState cmp
	}{
		{
			NewCoin(20, 1234, "ABC"),
			NewCoin(19, 999999999, "ABC"),
			1,
			pos,
			pos,
		},
		{
			NewCoin(0, -2, "FOO"),
			NewCoin(0, 1, "FOO"),
			-1,
			neg,
			pos,
		},
		{
			NewCoin(-4, -2456, "BAR"),
			NewCoin(-4, -4567, "BAR"),
			1,
			neg,
			neg,
		},
		{
			Coin{},
			Coin{},
			0,
			zero,
			zero,
		},
	}

	for idx, tc := range cases {
		i := strconv.Itoa(idx)

		// make sure both show proper results
		assert.Equal(t, tc.a.IsZero(), tc.aState == zero, i)
		assert.Equal(t, tc.a.IsPositive(), tc.aState == pos, i)
		assert.Equal(t, !tc.a.IsNonNegative(), tc.aState == neg, i)

		assert.Equal(t, tc.b.IsZero(), tc.bState == zero, i)
		assert.Equal(t, tc.b.IsPositive(), tc.bState == pos, i)
		assert.Equal(t, !tc.b.IsNonNegative(), tc.bState == neg, i)

		// make sure compare is correct
		assert.Equal(t, tc.a.Compare(tc.b), tc.expect, i)

		assert.True(t, tc.a.SameType(tc.b), i)
	}
}

func TestValidCoin(t *testing.T) {
	cases := []struct {
		coin            Coin
		valid           bool
		normalized      Coin
		normalizedValid bool
	}{
		{
			NewCoin(4, -123456789, "FOO"),
			false,
			NewCoin(3, 876543211, "FOO"),
			true,
		},
		{
			NewCoin(1, 0, "eth2"),
			false,
			NewCoin(1, 0, "eth2"),
			false,
		},
		{
			NewCoin(2, -1500500500, "ABC"),
			false,
			NewCoin(0, 499499500, "ABC"),
			true,
		},
		{
			NewCoin(-1, 1777888111, "ABC"),
			false,
			NewCoin(0, 777888111, "ABC"),
			true,
		},
		{
			NewCoin(0, -100, "DIN"),
			true,
			NewCoin(0, -100, "DIN"),
			true,
		},
	}

	for idx, tc := range cases {
		i := strconv.Itoa(idx)

		// Validate this one
		err := tc.coin.Validate()
		// normalize and check if there are still errors
		nrm, nerr := tc.coin.normalize()
		if nerr == nil {
			nerr = nrm.Validate()
		}

		if tc.valid {
			assert.NoError(t, err, i)
		} else {
			assert.Error(t, err, i)
		}

		assert.Equal(t, tc.normalized, nrm, i)
		assert.True(t, tc.normalized.Equals(nrm), i)

		if tc.normalizedValid {
			assert.NoError(t, nerr, i)
		} else {
			assert.Error(t, nerr, i)
		}
	}
}

func TestAddCoin(t *testing.T) {
	base := NewCoin(17, 2345566, "DEF")
	cases := []struct {
		a, b Coin
		res  Coin
		bad  bool
	}{
		// plus and minus equals 0
		{base, base.Negative(), NewCoin(0, 0, "DEF"), false},
		// wrong types
		{
			NewCoin(1, 2, "FOO"),
			NewCoin(2, 3, "BAR"),
			Coin{},
			true,
		},
		// normal math
		{
			NewCoin(7, 5000, "ABC"),
			NewCoin(-4, -12000, "ABC"),
			NewCoin(2, 999993000, "ABC"),
			false,
		},
		// overflow
		{
			NewCoin(500500500, 0, "SEE"),
			NewCoin(500500500, 0, "SEE"),
			Coin{},
			true,
		},
	}

	for idx, tc := range cases {
		i := strconv.Itoa(idx)

		c, err := tc.a.Add(tc.b)
		if tc.bad {
			assert.Error(t, err, i)
		} else {
			assert.NoError(t, err, i)
			assert.Equal(t, tc.res, c, i)
		}
	}
}

func TestMakeSet(t *testing.T) {
	cases := []struct {
		inputs   []Coin
		isEmpty  bool
		isNonNeg bool
		has      []Coin // <= the wallet
		dontHave []Coin // > or outside the wallet
	}{
		// empty
		{
			nil,
			true,
			true,
			nil,
			[]Coin{NewCoin(0, 0, "")},
		},
		// ignore 0
		{
			[]Coin{NewCoin(0, 0, "FOO")},
			true,
			true,
			nil,
			[]Coin{NewCoin(0, 0, "FOO")},
		},
		// simple
		{
			[]Coin{NewCoin(40, 0, "FUD")},
			false,
			true,
			[]Coin{NewCoin(10, 0, "FUD"), NewCoin(40, 0, "FUD")},
			[]Coin{NewCoin(40, 1, "FUD"), NewCoin(40, 0, "FUN")},
		},
		// out of order, with negative
		{
			[]Coin{NewCoin(-20, -3, "FIN"), NewCoin(40, 5, "BON")},
			false,
			false,
			[]Coin{NewCoin(40, 4, "BON"), NewCoin(-30, 0, "FIN")},
			[]Coin{NewCoin(40, 6, "BON"), NewCoin(-20, 0, "FIN")},
		},
		// combine and remove
		{
			[]Coin{NewCoin(-123, -456, "BOO"), NewCoin(123, 456, "BOO")},
			true,
			true,
			nil,
			[]Coin{NewCoin(0, 0, "BOO")},
		},
		// safely combine
		{
			[]Coin{NewCoin(12, 0, "ADA"), NewCoin(-123, -456, "BOO"), NewCoin(124, 756, "BOO")},
			false,
			true,
			[]Coin{NewCoin(12, 0, "ADA"), NewCoin(1, 300, "BOO")},
			[]Coin{NewCoin(13, 0, "ADA"), NewCoin(1, 400, "BOO")},
		},
	}

	for idx, tc := range cases {
		i := strconv.Itoa(idx)

		s, err := NewSet(tc.inputs...)
		require.NoError(t, err, i)
		assert.NoError(t, s.Validate(), i)
		assert.Equal(t, tc.isEmpty, s.IsEmpty(), i)
		assert.Equal(t, tc.isNonNeg, s.IsNonNegative(), i)

		for _, h := range tc.has {
			assert.True(t, s.Contains(h), i)
		}
		for _, d := range tc.dontHave {
			assert.False(t, s.Contains(d), i)
		}
	}
}

// TODO: Test Combine, Equals

// Test Add/Remove
func TestAddSet(t *testing.T) {

}
