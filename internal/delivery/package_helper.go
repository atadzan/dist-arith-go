package delivery

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"unicode"
)

var (
	ErrExpressionIsNotValid = fmt.Errorf("Expression is not valid")
	ErrDivisionByZero       = fmt.Sprintf("division by zero")
)

const (
	errInternalServerMsg  = "что-то пошло не так"
	errInvalidInputParams = "невалидные данные"
	errExpressionNotFound = "нет такого выражения"
)

type errorStruct struct {
	ErrorMsg string `json:"error"`
}

func newErrorResp(w http.ResponseWriter, httpStatus int, respMsg string) {
	rawBody, err := json.Marshal(errorStruct{ErrorMsg: respMsg})
	if err != nil {
		log.Println(err)
	}
	w.WriteHeader(httpStatus)
	_, err = w.Write(rawBody)
	if err != nil {
		log.Printf("can't write response body. Err: %v. HttpStatus: %d. Response msg: %s", err, httpStatus, respMsg)
	}
}

func sendPOSTRequest(address string, body []byte) (response []byte, err error) {
	resp, err := http.DefaultClient.Post(address, "application/json", bytes.NewReader(body))
	if err != nil {
		return
	}
	return io.ReadAll(resp.Body)
}

func sendGETRequest(address string) (response []byte, err error) {
	resp, err := http.DefaultClient.Get(address)
	if err != nil {
		return
	}
	return io.ReadAll(resp.Body)
}

func applyOperation(op byte, a, b float64) (float64, error) {
	switch op {
	case '+':
		return a + b, nil
	case '-':
		return a - b, nil
	case '*':
		return a * b, nil
	case '/':
		if b == 0 {
			return 0, errors.New(ErrDivisionByZero)
		}
		return a / b, nil
	default:
		return 0, ErrExpressionIsNotValid
	}
}

func precedence(op byte) int {
	switch op {
	case '+', '-':
		return 1
	case '*', '/':
		return 2
	default:
		return 0
	}
}

func calculate(expression string) (float64, error) {
	var values []float64
	var ops []byte

	applyTopOperation := func() error {
		if len(ops) == 0 || len(values) < 2 {
			return ErrExpressionIsNotValid
		}

		b := values[len(values)-1]
		a := values[len(values)-2]
		values = values[:len(values)-2]

		op := ops[len(ops)-1]
		ops = ops[:len(ops)-1]

		result, err := applyOperation(op, a, b)
		if err != nil {
			return err
		}
		values = append(values, result)
		return nil
	}

	for i := 0; i < len(expression); i++ {
		ch := expression[i]

		if ch == ' ' {
			continue
		}
		if unicode.IsDigit(rune(ch)) {
			start := i
			for i < len(expression) && (unicode.IsDigit(rune(expression[i])) || expression[i] == '.') {
				i++
			}
			value, err := strconv.ParseFloat(expression[start:i], 64)
			if err != nil {
				return 0, ErrExpressionIsNotValid
			}
			values = append(values, value)
			i--
		} else if ch == '(' {
			ops = append(ops, ch)
		} else if ch == ')' {
			for len(ops) > 0 && ops[len(ops)-1] != '(' {
				if err := applyTopOperation(); err != nil {
					return 0, err
				}
			}
			if len(ops) == 0 {
				return 0, errors.New("mismatched parentheses")
			}
			ops = ops[:len(ops)-1]
		} else if ch == '+' || ch == '-' || ch == '*' || ch == '/' {
			for len(ops) > 0 && precedence(ops[len(ops)-1]) >= precedence(ch) {
				if err := applyTopOperation(); err != nil {
					return 0, err
				}
			}
			ops = append(ops, ch)
		} else {
			return 0, ErrExpressionIsNotValid
		}
	}
	for len(ops) > 0 {
		if err := applyTopOperation(); err != nil {
			return 0, err
		}
	}
	if len(values) != 1 {
		return 0, ErrExpressionIsNotValid
	}

	return values[0], nil
}
