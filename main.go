package main

import (
	"html/template"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// Функция для добавления двух чисел (используется в шаблонах)
func add(a, b int) int {
	return a + b
}

// Определение структуры пользователя
type User struct {
	Username string
	Email    string
	Password string
}

// Глобальная переменная для хранения зарегистрированных пользователей и текущего пользователя
var users = make(map[string]User)

// Определение структуры вопроса
type Question struct {
	Text    string
	Answers []string
}

// Определение структуры опроса
type Poll struct {
	Theme     string
	Questions []Question
}

// Определение структуры ответа
type Answer struct {
	QuestionIndex int
	ResponseIndex int
}

// Глобальные переменные для хранения всех опросов и всех ответов
var allPolls []Poll
var allAnswers [][]Answer

func main() {
	// Определение маршрутов
	http.HandleFunc("/", homeHandler)
	http.HandleFunc("/register", registerHandler)
	http.HandleFunc("/login", loginHandler)
	http.HandleFunc("/logout", logoutHandler)
	http.HandleFunc("/create", createPollHandler)
	http.HandleFunc("/poll", takePollHandler)
	http.HandleFunc("/submit", submitPollHandler)
	http.HandleFunc("/results", showResultsHandler)
	http.Handle("/styles.css", http.FileServer(http.Dir(".")))

	// Запуск сервера на порту 8080
	http.ListenAndServe(":8080", nil)
}

// Проверка, авторизован ли пользователь
func isAuthenticated(r *http.Request) bool {
	cookie, err := r.Cookie("session")
	if err != nil {
		return false
	}
	_, exists := users[cookie.Value]
	return exists
}

// Обработчик домашней страницы
func homeHandler(w http.ResponseWriter, r *http.Request) {
	if !isAuthenticated(r) {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	username := getUserFromSession(r)
	tmpl, err := template.ParseFiles("home.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	tmpl.Execute(w, struct{ Username string }{Username: username})
}

// Получение имени пользователя из сессии
func getUserFromSession(r *http.Request) string {
	cookie, _ := r.Cookie("session")
	user, exists := users[cookie.Value]
	if exists {
		return user.Username
	}
	return ""
}

// Обработчик регистрации
func registerHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		username := r.FormValue("username")
		email := r.FormValue("email")
		password := r.FormValue("password")

		if len(password) < 8 {
			renderTemplate(w, "register.html", "Пароль должен содержать не менее 8 символов")
			return
		}

		if _, exists := users[email]; exists {
			renderTemplate(w, "register.html", "Пользователь с таким email уже существует")
			return
		}

		users[email] = User{
			Username: username,
			Email:    email,
			Password: password,
		}
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	renderTemplate(w, "register.html", nil)
}

// Обработчик входа
func loginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		email := r.FormValue("email")
		password := r.FormValue("password")

		user, exists := users[email]
		if !exists || user.Password != password {
			renderTemplate(w, "login.html", "Неверный email или пароль")
			return
		}

		cookie := http.Cookie{
			Name:    "session",
			Value:   email,
			Expires: time.Now().Add(24 * time.Hour),
		}
		http.SetCookie(w, &cookie)

		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	renderTemplate(w, "login.html", nil)
}

// Обработчик выхода
func logoutHandler(w http.ResponseWriter, r *http.Request) {
	cookie := http.Cookie{
		Name:    "session",
		Value:   "",
		Expires: time.Now().Add(-1 * time.Hour),
	}
	http.SetCookie(w, &cookie)
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

// Обработчик создания опроса
func createPollHandler(w http.ResponseWriter, r *http.Request) {
	if !isAuthenticated(r) {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	if r.Method == http.MethodPost {
		theme := r.FormValue("theme")
		var questions []Question
		for i := 1; ; i++ {
			question := r.FormValue("question" + strconv.Itoa(i))
			if question == "" {
				break
			}
			answers := strings.Split(r.FormValue("answers"+strconv.Itoa(i)), ",")
			questions = append(questions, Question{
				Text:    question,
				Answers: answers,
			})
		}
		allPolls = append(allPolls, Poll{
			Theme:     theme,
			Questions: questions,
		})
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	http.ServeFile(w, r, "create.html")
}

// Обработчик прохождения опроса
func takePollHandler(w http.ResponseWriter, r *http.Request) {
	if len(allPolls) == 0 {
		http.Error(w, "No available polls", http.StatusInternalServerError)
		return
	}

	tmpl, err := template.New("poll.html").Funcs(template.FuncMap{"add": add}).ParseFiles("poll.html")
	if err != nil {
		http.Error(w, "Error parsing template: "+err.Error(), http.StatusInternalServerError)
		return
	}

	data := struct {
		Theme     string
		Questions []Question
	}{
		Theme:     allPolls[len(allPolls)-1].Theme,
		Questions: allPolls[len(allPolls)-1].Questions,
	}

	if err := tmpl.Execute(w, data); err != nil {
		http.Error(w, "Error executing template: "+err.Error(), http.StatusInternalServerError)
	}
}

// Обработчик отправки ответов
func submitPollHandler(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Failed to parse form: "+err.Error(), http.StatusInternalServerError)
		return
	}

	var userAnswers []Answer
	for i := range allPolls[len(allPolls)-1].Questions {
		responseIndex, _ := strconv.Atoi(r.FormValue("answers" + strconv.Itoa(i+1)))
		userAnswers = append(userAnswers, Answer{
			QuestionIndex: i,
			ResponseIndex: responseIndex,
		})
	}

	allAnswers = append(allAnswers, userAnswers)
	http.Redirect(w, r, "/results", http.StatusSeeOther)
}

// Обработчик отображения результатов
func showResultsHandler(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.New("results.html").Funcs(template.FuncMap{"add": add}).ParseFiles("results.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Получаем последние ответы
	lastAnswers := allAnswers[len(allAnswers)-1]

	data := struct {
		Theme     string
		Questions []Question
		Answers   []Answer
	}{
		Theme:     allPolls[len(allPolls)-1].Theme,
		Questions: allPolls[len(allPolls)-1].Questions,
		Answers:   lastAnswers,
	}

	if err := tmpl.Execute(w, data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// Вспомогательная функция для рендеринга шаблонов с ошибками
func renderTemplate(w http.ResponseWriter, tmpl string, data interface{}) {
	t, err := template.ParseFiles(tmpl)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	t.Execute(w, data)
}
