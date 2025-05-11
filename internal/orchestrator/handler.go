package orchestrator

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/atadzan/dist-arith-go/internal/constants"
	"github.com/atadzan/dist-arith-go/internal/models"
	"github.com/atadzan/dist-arith-go/internal/repository"
)

type HTTPHandlers struct {
	auth      *AuthService
	repo      repository.Repository
	scheduler *Scheduler
}

func NewHTTPHandlers(auth *AuthService, repo repository.Repository, scheduler *Scheduler) *HTTPHandlers {
	return &HTTPHandlers{
		auth:      auth,
		repo:      repo,
		scheduler: scheduler,
	}
}

type AuthRequest struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

type LoginResponse struct {
	Token string `json:"token"`
}

func (h *HTTPHandlers) RegisterHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Метод не разрешен", http.StatusMethodNotAllowed)
		return
	}

	var req AuthRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Ошибка декодирования запроса: "+err.Error(), http.StatusBadRequest)
		return
	}

	login := strings.TrimSpace(req.Login)
	password := strings.TrimSpace(req.Password)

	if login == "" || password == "" {
		http.Error(w, "Логин и пароль не могут быть пустыми", http.StatusBadRequest)
		return
	}

	if len(password) < 6 {
		http.Error(w, "Пароль должен быть не менее 6 символов", http.StatusBadRequest)
		return
	}

	hashedPassword, err := HashPassword(password)
	if err != nil {
		log.Printf("Ошибка хэширования пароля для пользователя %s: %v", login, err)
		http.Error(w, "Внутренняя ошибка сервера", http.StatusInternalServerError)
		return
	}

	_, err = h.repo.CreateUser(login, hashedPassword)
	if err != nil {
		if strings.Contains(err.Error(), "уже существует") {
			http.Error(w, err.Error(), http.StatusConflict)
		} else {
			log.Printf("Ошибка создания пользователя %s в БД: %v", login, err)
			http.Error(w, "Внутренняя ошибка сервера", http.StatusInternalServerError)
		}
		return
	}

	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "Пользователь '%s' успешно зарегистрирован", login)
}

func (h *HTTPHandlers) LoginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Метод не разрешен", http.StatusMethodNotAllowed)
		return
	}

	var req AuthRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Ошибка декодирования запроса: "+err.Error(), http.StatusBadRequest)
		return
	}

	login := strings.TrimSpace(req.Login)
	password := strings.TrimSpace(req.Password)

	if login == "" || password == "" {
		http.Error(w, "Логин и пароль не могут быть пустыми", http.StatusBadRequest)
		return
	}

	user, err := h.repo.GetUserByLogin(login)
	if err != nil {
		log.Printf("Ошибка получения пользователя %s из БД: %v", login, err)
		http.Error(w, "Внутренняя ошибка сервера", http.StatusInternalServerError)
		return
	}

	if user == nil || !CheckPasswordHash(password, user.PasswordHash) {
		http.Error(w, "Неверный логин или пароль", http.StatusUnauthorized)
		return
	}

	tokenString, err := h.auth.GenerateJWT(user.ID)
	if err != nil {
		log.Printf("Ошибка генерации JWT для пользователя %s (ID: %d): %v", login, user.ID, err)
		http.Error(w, "Внутренняя ошибка сервера", http.StatusInternalServerError)
		return
	}

	resp := LoginResponse{Token: tokenString}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}

type CalculateRequest struct {
	Expression string `json:"expression"`
}

func (h *HTTPHandlers) CalculateHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Метод не разрешен", http.StatusMethodNotAllowed)
		return
	}

	userID, ok := GetUserIDFromContext(r.Context())
	if !ok {
		log.Println("Ошибка: не удалось получить userID из контекста в CalculateHandler")
		http.Error(w, "Внутренняя ошибка сервера (контекст пользователя)", http.StatusInternalServerError)
		return
	}

	var req CalculateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Ошибка декодирования JSON: "+err.Error(), http.StatusBadRequest)
		return
	}
	exprStr := strings.TrimSpace(req.Expression) // Восстановлено определение exprStr

	if exprStr == "" {
		http.Error(w, "Пустое выражение недопустимо", http.StatusBadRequest)
		return
	}

	exprID, err := h.repo.CreateExpression(userID, exprStr) // userID и exprStr теперь определены
	if err != nil {
		log.Printf("Ошибка создания выражения в БД для пользователя %d: %v", userID, err)
		http.Error(w, "Внутренняя ошибка сервера при сохранении выражения", http.StatusInternalServerError)
		return
	}

	log.Printf("Создано выражение ID %d для пользователя %d: %s", exprID, userID, exprStr)

	go func(id int64, expression string) {
		err := h.scheduler.ScheduleTasks(id, expression)
		if err != nil {
			log.Printf("Асинхронная ошибка планирования задач для выражения ID %d: %v", id, err)
		}
	}(exprID, exprStr)

	respData := map[string]interface{}{
		"id":         exprID,
		"expression": exprStr,
		"status":     constants.StatusPending,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated) // 201 Created
	if err := json.NewEncoder(w).Encode(respData); err != nil {
		log.Printf("Ошибка записи JSON ответа для CalculateHandler (exprID: %d): %v", exprID, err)
	}
}

func EnableCORS(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")                                                                                   // Разрешаем все источники (для разработки)
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")                                                    // Разрешенные методы
		w.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization") // Разрешенные заголовки

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		handler.ServeHTTP(w, r)
	})
}

func (h *HTTPHandlers) ExpressionsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Метод не разрешен", http.StatusMethodNotAllowed)
		return
	}

	userID, ok := GetUserIDFromContext(r.Context())
	if !ok {
		log.Println("Ошибка: не удалось получить userID из контекста в ExpressionsHandler")
		http.Error(w, "Внутренняя ошибка сервера (контекст пользователя)", http.StatusInternalServerError)
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/api/v1/expressions")
	idStr := strings.Trim(path, "/")

	w.Header().Set("Content-Type", "application/json")

	if idStr == "" {
		expressions, err := h.repo.GetExpressionsByUserID(userID)
		if err != nil {
			log.Printf("Ошибка получения списка выражений для пользователя %d: %v", userID, err)
			http.Error(w, "Внутренняя ошибка сервера при получении выражений", http.StatusInternalServerError)
			return
		}
		if expressions == nil {
			expressions = make([]models.Expression, 0)
		}
		if err := json.NewEncoder(w).Encode(expressions); err != nil {
			log.Printf("Ошибка записи JSON ответа для списка выражений (userID: %d): %v", userID, err)
		}
		return
	}

	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Неверный ID выражения: "+idStr, http.StatusBadRequest)
		return
	}

	expression, err := h.repo.GetExpressionByID(id, userID)
	if err != nil {
		log.Printf("Ошибка получения выражения ID %d для пользователя %d: %v", id, userID, err)
		http.Error(w, "Внутренняя ошибка сервера при получении выражения", http.StatusInternalServerError)
		return
	}

	if expression == nil {
		http.Error(w, fmt.Sprintf("Выражение с ID %d не найдено или доступ запрещен", id), http.StatusNotFound)
		return
	}

	if err := json.NewEncoder(w).Encode(expression); err != nil {
		log.Printf("Ошибка записи JSON ответа для выражения ID %d (userID: %d): %v", id, userID, err)
	}
}
