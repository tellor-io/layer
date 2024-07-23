package types

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIsQueryId64chars(t *testing.T) {
	// Test cases for valid query IDs
	validQueryIDs := []string{
		"0x1234567890123456789012345678901234567890123456789012345678901234",
		"1234567890123456789012345678901234567890123456789012345678901234",
	}
	for _, queryID := range validQueryIDs {
		if !IsQueryId64chars(queryID) {
			t.Errorf("Expected query ID %s to be valid, but it was invalid", queryID)
		}
	}

	// Test cases for invalid query IDs
	invalidQueryIDs := []string{
		"0x123456789012345678901234567890123456789012345678901234567890123",   // Length less than 64
		"0x12345678901234567890123456789012345678901234567890123456789012345", // Length greater than 64
		"0x", // Empty query ID
		"",   // Empty query ID without 0x prefix
		"abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!@#$%^&*()-=_+[]{}|;:'~,.<>?/", // Length greater than 64
		"0xabcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789999",                         // Length 65
	}
	for _, queryID := range invalidQueryIDs {
		if IsQueryId64chars(queryID) {
			t.Errorf("Expected query ID %s to be invalid, but it was valid", queryID)
		}
	}
}

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
