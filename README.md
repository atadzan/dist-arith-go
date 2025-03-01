## Распределённый вычислитель арифметических выражений
### Состоит из 2 сервисов:
* Оркестратор - который принимает арифметическое выражение и хранит задачу.
* Worker - который может получить от оркестратора задачу, выполнить его и вернуть серверу результат.

### Чтобы запустить сервисы выполните следующие действии:
* Запуск оркестратора 
``make run-orchestrator``
* Запуск воркера
``make run-worker``
### Оркестратор
#### Добавление вычисления арифметического выражения
  Пример запроса
```
curl --location 'localhost/api/v1/calculate' \
     --header 'Content-Type: application/json' \
    --data '{
            "expression": "5+4"
            }'
```
Пример ответа
```
{
    "id": "4f8b0ed7-eb64-42f2-baee-89bca4ea73d9"
}
```
####  Получение списка выражений
```
curl --request GET \
     --url http://localhost:8080/api/v1/expressions '
```
Пример ответа
```
{
	"expressions": [
		{
			"id": "4f8b0ed7-eb64-42f2-baee-89bca4ea73d9",
			"status": "pending",
			"result": 0
		}
	]
}
```
#### Получение выражения по его идентификатору
Пример запроса
```
curl --location 'localhost/api/v1/expressions/4f8b0ed7-eb64-42f2-baee-89bca4ea73d9'
```
Пример ответа
```
{
	"expression": 
		{
			"id": "4f8b0ed7-eb64-42f2-baee-89bca4ea73d9",
			"status": "pending",
			"result": 0
		}
}
```

#### Получение задачи для выполнения
Пример запроса
```
curl --location 'localhost/internal/task'
```

Пример ответа
```
{
    "task":
        {
            "id": <идентификатор задачи>,
            "operation": <операция>,
            "operationTime": <время выполнения операции>
        }
}
```

#### Прием результата обработки данных.
Пример запроса
```
curl --location 'localhost/internal/task' \
     --header 'Content-Type: application/json' \
     --data '{
       "id": 1,
       "result": 2.5
     }'
```