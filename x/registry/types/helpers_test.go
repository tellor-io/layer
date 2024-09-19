package types

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestHas0xPrefix(t *testing.T) {
	// Test cases for strings with 0x prefix
	stringsWith0xPrefix := []struct {
		str      string
		expected bool
	}{
		{"0x1234567890", true},
		{"0xabcdef", true},
		{"0x", true},
		{"0X1234567890", true},
		{"0Xabcdef", true},
		{"0X", true},
	}
	for _, tc := range stringsWith0xPrefix {
		result := Has0xPrefix(tc.str)
		if result != tc.expected {
			t.Errorf("Expected Has0xPrefix(%s) to be %v, but got %v", tc.str, tc.expected, result)
		}
	}

	// Test cases for strings without 0x prefix
	stringsWithout0xPrefix := []struct {
		str      string
		expected bool
	}{
		{"1234567890", false},
		{"abcdef", false},
		{"", false},
		{"X1234567890", false},
		{"Xabcdef", false},
		{"X", false},
	}
	for _, tc := range stringsWithout0xPrefix {
		result := Has0xPrefix(tc.str)
		if result != tc.expected {
			t.Errorf("Expected Has0xPrefix(%s) to be %v, but got %v", tc.str, tc.expected, result)
		}
	}
}

func TestRemove0xPrefix(t *testing.T) {
	stringsWith0xPrefix := []struct {
		str      string
		expected bool
	}{
		{"0x1234567890", true},
		{"0xabcdef", true},
		{"0x", true},
		{"0X1234567890", true},
		{"0Xabcdef", true},
		{"0X", true},
	}
	for _, tc := range stringsWith0xPrefix {
		result := Remove0xPrefix(tc.str)
		if strings.Contains(result, "0x") {
			t.Errorf("Expected Remove0xPrefix(%s) to be %v, but got %v", tc.str, tc.expected, result)
		}
	}

	// Test cases for strings without 0x prefix
	stringsWithout0xPrefix := []struct {
		str      string
		expected bool
	}{
		{"1234567890", false},
		{"abcdef", false},
		{"", false},
		{"X1234567890", false},
		{"Xabcdef", false},
		{"X", false},
	}
	for _, tc := range stringsWithout0xPrefix {
		result := Remove0xPrefix(tc.str)
		if strings.Contains(result, "0x") {
			t.Errorf("Expected Remove0xPrefix(%s) to be %v, but got %v", tc.str, tc.expected, result)
		}
	}
}

func TestConvertToJSON(t *testing.T) {
	require := require.New(t)

	slice := []interface{}{
		"test",
		1,
		true,
	}
	json, err := ConvertToJSON(slice)
	require.NoError(err)
	require.Equal("[\"test\",1,true]", json)

	slice = []interface{}{}
	json, err = ConvertToJSON(slice)
	require.NoError(err)
	require.Equal("[]", json)
}
