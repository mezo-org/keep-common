package main

// Tests for the copied Solidity-to-Go type-mapping helpers.

import (
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

func TestBindBasicType(t *testing.T) {
	mustType := func(solidity string) abi.Type {
		t.Helper()
		typ, err := abi.NewType(solidity, "", nil)
		if err != nil {
			t.Fatalf("abi.NewType(%q) failed: %v", solidity, err)
		}
		return typ
	}

	tests := []struct {
		solidity string
		expected string
	}{
		{"address", "common.Address"},
		{"bool", "bool"},
		{"string", "string"},
		{"bytes", "[]byte"},
		{"bytes32", "[32]byte"},
		{"bytes20", "[20]byte"},
		{"uint8", "uint8"},
		{"uint16", "uint16"},
		{"uint32", "uint32"},
		{"uint64", "uint64"},
		{"uint128", "*big.Int"},
		{"uint256", "*big.Int"},
		{"int8", "int8"},
		{"int16", "int16"},
		{"int32", "int32"},
		{"int64", "int64"},
		{"int128", "*big.Int"},
		{"int256", "*big.Int"},
	}

	for _, tc := range tests {
		t.Run(tc.solidity, func(t *testing.T) {
			got := bindBasicType(mustType(tc.solidity))
			if got != tc.expected {
				t.Errorf("bindBasicType(%q) = %q, want %q", tc.solidity, got, tc.expected)
			}
		})
	}
}

func TestBindStructType_Primitives(t *testing.T) {
	typ, err := abi.NewType("uint256", "", nil)
	if err != nil {
		t.Fatalf("abi.NewType failed: %v", err)
	}
	structs := map[string]*tmplStruct{}
	if got := bindStructType(typ, structs); got != "*big.Int" {
		t.Errorf("bindStructType(uint256) = %q, want %q", got, "*big.Int")
	}
	if len(structs) != 0 {
		t.Errorf("primitives should not populate the struct tracker, got %d entries", len(structs))
	}
}

func TestBindStructType_Arrays(t *testing.T) {
	typ, err := abi.NewType("address[3]", "", nil)
	if err != nil {
		t.Fatalf("abi.NewType failed: %v", err)
	}
	structs := map[string]*tmplStruct{}
	if got := bindStructType(typ, structs); got != "[3]common.Address" {
		t.Errorf("bindStructType(address[3]) = %q, want %q", got, "[3]common.Address")
	}
}

func TestBindStructType_Slices(t *testing.T) {
	typ, err := abi.NewType("uint256[]", "", nil)
	if err != nil {
		t.Fatalf("abi.NewType failed: %v", err)
	}
	structs := map[string]*tmplStruct{}
	if got := bindStructType(typ, structs); got != "[]*big.Int" {
		t.Errorf("bindStructType(uint256[]) = %q, want %q", got, "[]*big.Int")
	}
}

func TestBindStructType_NamedTuple(t *testing.T) {
	components := []abi.ArgumentMarshaling{
		{Name: "amount", Type: "uint256"},
		{Name: "recipient", Type: "address"},
	}
	typ, err := abi.NewType("tuple", "struct Transfer", components)
	if err != nil {
		t.Fatalf("abi.NewType failed: %v", err)
	}
	structs := map[string]*tmplStruct{}
	if got := bindStructType(typ, structs); got != "Transfer" {
		t.Errorf("bindStructType(named tuple) = %q, want %q", got, "Transfer")
	}
	if len(structs) != 1 {
		t.Errorf("expected 1 tuple registered, got %d", len(structs))
	}

	if got := bindStructType(typ, structs); got != "Transfer" {
		t.Errorf("re-entry bindStructType(named tuple) = %q, want %q", got, "Transfer")
	}
	if len(structs) != 1 {
		t.Errorf("re-entry should not add another entry; got %d", len(structs))
	}
}

func TestBindStructType_AnonymousTuple(t *testing.T) {
	components := []abi.ArgumentMarshaling{
		{Name: "a", Type: "uint256"},
		{Name: "b", Type: "bool"},
	}
	typ, err := abi.NewType("tuple", "", components)
	if err != nil {
		t.Fatalf("abi.NewType failed: %v", err)
	}
	structs := map[string]*tmplStruct{}
	got := bindStructType(typ, structs)
	if got != "Struct0" {
		t.Errorf("bindStructType(anon tuple) = %q, want %q", got, "Struct0")
	}
}

func TestBindStructType_AnonymousTupleReentry(t *testing.T) {
	first, err := abi.NewType("tuple", "", []abi.ArgumentMarshaling{
		{Name: "a", Type: "uint256"},
		{Name: "b", Type: "bool"},
	})
	if err != nil {
		t.Fatalf("abi.NewType first tuple failed: %v", err)
	}
	second, err := abi.NewType("tuple", "", []abi.ArgumentMarshaling{
		{Name: "recipient", Type: "address"},
	})
	if err != nil {
		t.Fatalf("abi.NewType second tuple failed: %v", err)
	}

	structs := map[string]*tmplStruct{}
	if got := bindStructType(first, structs); got != "Struct0" {
		t.Errorf("bindStructType(first) = %q, want %q", got, "Struct0")
	}
	if got := bindStructType(second, structs); got != "Struct1" {
		t.Errorf("bindStructType(second) = %q, want %q", got, "Struct1")
	}
	if got := bindStructType(first, structs); got != "Struct0" {
		t.Errorf("bindStructType(first re-entry) = %q, want %q", got, "Struct0")
	}
}

func TestBindStructType_NestedTupleFields(t *testing.T) {
	typ, err := abi.NewType("tuple", "struct Outer", []abi.ArgumentMarshaling{
		{Name: "inner", Type: "tuple", InternalType: "struct Inner", Components: []abi.ArgumentMarshaling{
			{Name: "amount", Type: "uint256"},
		}},
		{Name: "recipient", Type: "address"},
	})
	if err != nil {
		t.Fatalf("abi.NewType failed: %v", err)
	}

	structs := map[string]*tmplStruct{}
	if got := bindStructType(typ, structs); got != "Outer" {
		t.Errorf("bindStructType(nested tuple) = %q, want %q", got, "Outer")
	}

	outer := structs[typ.TupleRawName+typ.String()]
	if outer.Name != "Outer" {
		t.Errorf("outer.Name = %q, want %q", outer.Name, "Outer")
	}
	if len(outer.Fields) != 2 {
		t.Fatalf("outer field count = %d, want %d", len(outer.Fields), 2)
	}
	if outer.Fields[0].Name != "Inner" {
		t.Errorf("outer.Fields[0].Name = %q, want %q", outer.Fields[0].Name, "Inner")
	}
	if outer.Fields[0].Type != "Inner" {
		t.Errorf("outer.Fields[0].Type = %q, want %q", outer.Fields[0].Type, "Inner")
	}
}

func TestBindTopicType_DynamicCollapsesToHash(t *testing.T) {
	tests := []struct {
		solidity string
		expected string
	}{
		{"string", "common.Hash"},
		{"bytes", "common.Hash"},
		{"address", "common.Address"},
		{"bytes32", "[32]byte"},
		{"uint256", "*big.Int"},
	}
	for _, tc := range tests {
		t.Run(tc.solidity, func(t *testing.T) {
			typ, err := abi.NewType(tc.solidity, "", nil)
			if err != nil {
				t.Fatalf("abi.NewType failed: %v", err)
			}
			structs := map[string]*tmplStruct{}
			if got := bindTopicType(typ, structs); got != tc.expected {
				t.Errorf("bindTopicType(%q) = %q, want %q", tc.solidity, got, tc.expected)
			}
		})
	}
}

func TestStructured(t *testing.T) {
	mk := func(pairs ...[2]string) abi.Arguments {
		t.Helper()
		out := make(abi.Arguments, 0, len(pairs))
		for _, p := range pairs {
			typ, err := abi.NewType(p[1], "", nil)
			if err != nil {
				t.Fatalf("abi.NewType(%q) failed: %v", p[1], err)
			}
			out = append(out, abi.Argument{Name: p[0], Type: typ})
		}
		return out
	}

	tests := []struct {
		name     string
		args     abi.Arguments
		expected bool
	}{
		{
			name:     "zero args",
			args:     mk(),
			expected: false,
		},
		{
			name:     "one arg is not enough",
			args:     mk([2]string{"amount", "uint256"}),
			expected: false,
		},
		{
			name:     "two named distinct args",
			args:     mk([2]string{"amount", "uint256"}, [2]string{"recipient", "address"}),
			expected: true,
		},
		{
			name:     "anonymous arg disqualifies",
			args:     mk([2]string{"amount", "uint256"}, [2]string{"", "address"}),
			expected: false,
		},
		{
			name:     "colliding names after normalisation disqualifies",
			args:     mk([2]string{"my_var", "uint256"}, [2]string{"myVar", "address"}),
			expected: false,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := structured(tc.args); got != tc.expected {
				t.Errorf("structured(%s) = %v, want %v", tc.name, got, tc.expected)
			}
		})
	}
}
