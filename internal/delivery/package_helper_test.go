package delivery

import (
	"errors"
	"testing"
)

func TestApplyOperation(t *testing.T) {
	tests := []struct {
		name    string
		op      byte
		a, b    float64
		want    float64
		wantErr error
	}{
		{"Addition", '+', 5, 3, 8, nil},
		{"Subtraction", '-', 5, 3, 2, nil},
		{"Multiplication", '*', 5, 3, 15, nil},
		{"Division", '/', 6, 3, 2, nil},
		{"Invalid operation", '%', 6, 3, 0, ErrExpressionIsNotValid},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := applyOperation(tt.op, tt.a, tt.b)
			if got != tt.want || !errors.Is(err, tt.wantErr) {
				t.Errorf("applyOperation(%q, %v, %v) = (%v, %v), want (%v, %v)",
					tt.op, tt.a, tt.b, got, err, tt.want, tt.wantErr)
			}
		})
	}
}

func TestPrecedence(t *testing.T) {
	tests := []struct {
		name string
		op   byte
		want int
	}{
		{"Addition", '+', 1},
		{"Subtraction", '-', 1},
		{"Multiplication", '*', 2},
		{"Division", '/', 2},
		{"Invalid operator", '%', 0},
		{"Empty character", 0, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := precedence(tt.op)
			if got != tt.want {
				t.Errorf("precedence(%q) = %v, want %v", tt.op, got, tt.want)
			}
		})
	}
}

func TestCalculate(t *testing.T) {
	tests := []struct {
		expression string
		want       float64
		wantErr    error
	}{
		{"2+3", 5, nil},
		{"10-4", 6, nil},
		{"3*4", 12, nil},
		{"8/2", 4, nil},
		{"(2+3)*4", 20, nil},
		{"3.5+1.2", 4.7, nil},
		{"10/(2+3)", 2, nil},
		{"4/0", 0, errors.New("division by zero")}, // Division by zero
		{"2++2", 0, ErrExpressionIsNotValid},       // Invalid expression
		{"abc", 0, ErrExpressionIsNotValid},        // Invalid characters
	}

	for _, tt := range tests {
		t.Run(tt.expression, func(t *testing.T) {
			got, err := calculate(tt.expression)
			if (err != nil) != (tt.wantErr != nil) || (err != nil && err.Error() != tt.wantErr.Error()) {
				t.Errorf("calculate(%q) error = %v, wantErr %v", tt.expression, err, tt.wantErr)
			}
			if got != tt.want {
				t.Errorf("calculate(%q) = %v, want %v", tt.expression, got, tt.want)
			}
		})
	}
}
