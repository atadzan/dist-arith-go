package delivery

import "time"

type inputExpression struct {
	Expression string `json:"expression"`
}

type Task struct {
	Id         string  `json:"id"`
	Status     string  `json:"status"`
	Result     float64 `json:"result"`
	Expression string  `json:"expression"`
}

type ListExpression struct {
	Id     string  `json:"id"`
	Status string  `json:"status"`
	Result float64 `json:"result"`
}

type ListExpressions struct {
	Expressions []ListExpression `json:"expressions"`
}

type ListExpressionById struct {
	Expression ListExpression `json:"expression"`
}

type ReceiveExpression struct {
	Task struct {
		Id            string        `json:"id"`
		Expression    string        `json:"task"`
		OperationTime time.Duration `json:"operationTime"`
	} `json:"task"`
}

type SendResult struct {
	Id     string  `json:"id"`
	Result float64 `json:"result"`
}
