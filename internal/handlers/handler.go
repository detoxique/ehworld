package handlers

import (
	"ehchobyahs/internal/models"
	"ehchobyahs/internal/service"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"log"
	"math"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/twitch"
)

var (
	sessionName   = "ehcho-session"
	sessionSecret = os.Getenv("SESSION_SECRET")
)

var (
	oauthConfig = &oauth2.Config{
		ClientID:     os.Getenv("TWITCH_CLIENT_ID"),
		ClientSecret: os.Getenv("TWITCH_CLIENT_SECRET"),
		RedirectURL:  "https://ehworld.ru/auth/callback",
		Scopes:       []string{"user:read:email"},
		Endpoint:     twitch.Endpoint,
	}
	store = sessions.NewCookieStore([]byte(sessionSecret))
)

var adminOAuthConfig = &oauth2.Config{
	ClientID:     os.Getenv("TWITCH_CLIENT_ID_BOT"),
	ClientSecret: os.Getenv("TWITCH_CLIENT_SECRET_BOT"),
	RedirectURL:  "https://ehworld.ru/admin/callback",
	Scopes:       []string{"chat:edit", "user:write:chat", "moderator:manage:banned_users", "channel:manage:vips"},
	Endpoint:     twitch.Endpoint,
}

var (
	channelsToCheck    = []string{} // Список каналов для проверки
	cacheMutex         = &sync.RWMutex{}
	cachedLiveChannels []models.TwitchStream
	lastUpdateTime     time.Time
	cacheDuration      = time.Minute
)

var (
	leaderboardMutex       = &sync.RWMutex{}
	cachedTopAuthors       []models.User
	leaderboardLastUpdated time.Time
	updateInterval         = time.Minute
)

func UpdateConfig() {
	sessionSecret = os.Getenv("SESSION_SECRET")
	oauthConfig.ClientID = os.Getenv("TWITCH_CLIENT_ID")
	oauthConfig.ClientSecret = os.Getenv("TWITCH_CLIENT_SECRET")
	store = sessions.NewCookieStore([]byte(sessionSecret))

	adminOAuthConfig.ClientID = os.Getenv("TWITCH_CLIENT_ID_BOT")
	adminOAuthConfig.ClientSecret = os.Getenv("TWITCH_CLIENT_SECRET_BOT")

	channelsToCheck = service.LoadChannelsToCheck()
}

func ServeHomePage(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, sessionName)
	if userID, ok := session.Values["user_id"]; ok {
		_, err := service.GetUserByID(userID.(int))
		if err == nil {
			http.Redirect(w, r, "/main", http.StatusFound)
			return
		}
	}

	tmpl, err := template.ParseFiles("templates/home.html")
	if err != nil {
		log.Println("Не удалось получить шаблон страницы профиля" + err.Error())
		http.Error(w, "Template error", http.StatusInternalServerError)
		return
	}

	err = tmpl.Execute(w, nil)
	if err != nil {
		log.Println(err.Error())
	}
}

func AuthTwitchHandler(w http.ResponseWriter, r *http.Request) {
	url := oauthConfig.AuthCodeURL("state", oauth2.AccessTypeOnline)
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

func AdminAuthHandler(w http.ResponseWriter, r *http.Request) {
	url := adminOAuthConfig.AuthCodeURL("state", oauth2.AccessTypeOffline)
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

func AdminCallbackHandler(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	token, err := adminOAuthConfig.Exchange(r.Context(), code)

	log.Println("Access token " + token.AccessToken + "\nRefresh Token " + token.RefreshToken)

	if err != nil {
		log.Printf("Admin token exchange error: %v", err)
		http.Error(w, "Failed to exchange token", http.StatusInternalServerError)
		return
	}

	os.Setenv("TWITCH_BOT_TOKEN", token.AccessToken)
	err = os.Setenv("TWITCH_BOT_REFRESH_TOKEN", token.RefreshToken)
	if err != nil {
		log.Printf("Failed to save: %v", err)
	}

	service.SaveTokens(token.AccessToken, token.RefreshToken)

	// Получаем ID бота
	req, _ := http.NewRequest("GET", "https://api.twitch.tv/helix/users", nil)
	req.Header.Set("Client-ID", os.Getenv("TWITCH_CLIENT_ID_BOT"))
	req.Header.Set("Authorization", "Bearer "+token.AccessToken)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Failed to get bot info: %v", err)
		http.Error(w, "Failed to get bot ID", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	body, _ := ioutil.ReadAll(resp.Body)
	var userData struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	json.Unmarshal(body, &userData)

	log.Println(userData.Data)

	if len(userData.Data) == 0 {
		http.Error(w, "Failed to get bot ID", http.StatusInternalServerError)
		return
	}

	// Перезапускаем приложение
	http.Redirect(w, r, "/admin", http.StatusFound)
}

func AuthCallbackHandler(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	token, err := oauthConfig.Exchange(r.Context(), code)
	if err != nil {
		log.Printf("Token exchange error: %v", err)
		http.Error(w, "Failed to exchange token: "+err.Error(), http.StatusInternalServerError)
		return
	}

	//log.Println("Token: " + token.AccessToken)

	// Получаем информацию о пользователе с правильными заголовками
	req, err := http.NewRequest("GET", "https://api.twitch.tv/helix/users", nil)
	if err != nil {
		log.Printf("Request creation error: %v", err)
		http.Error(w, "Failed to create request: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Устанавливаем обязательные заголовки
	req.Header.Set("Client-ID", oauthConfig.ClientID)
	req.Header.Set("Authorization", "Bearer "+token.AccessToken)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("API request error: %v", err)
		http.Error(w, "Failed to get user info: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	//log.Println(string(body))

	var userData struct {
		Data []models.User `json:"data"`
	}
	if err := json.Unmarshal(body, &userData); err != nil || len(userData.Data) == 0 {
		http.Error(w, "Failed to parse user info", http.StatusInternalServerError)
		return
	}

	twitchUser := userData.Data[0]
	user, err := service.CreateOrUpdateUser(twitchUser)
	if err != nil {
		log.Println(err.Error())
		http.Error(w, "Failed to get user info: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// if err != nil {
	// 	log.Println("Failed to save user " + err.Error())
	// 	http.Error(w, "Error: "+err.Error(), http.StatusInternalServerError)
	// 	return
	// }

	session, _ := store.Get(r, sessionName)
	session.Values["user_id"] = user.ID
	err = session.Save(r, w)
	if err != nil {
		log.Println("Failed to save session " + err.Error())
		http.Error(w, "Failed to save session", http.StatusBadRequest)
		return
	}

	http.Redirect(w, r, "/main", http.StatusFound)
}

func AuthMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		session, _ := store.Get(r, sessionName)
		_, ok := session.Values["user_id"]
		if !ok {
			http.Redirect(w, r, "/", http.StatusFound)
			return
		}
		next.ServeHTTP(w, r)
	}
}

// Проверка модератора
func ModeratorMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		session, _ := store.Get(r, sessionName)
		user_id, ok := session.Values["user_id"]
		if !ok {
			http.Redirect(w, r, "/notfound", http.StatusFound)
			return
		}
		if service.CheckModeratorOrAdminRole(user_id.(int)) {
			next.ServeHTTP(w, r)
		} else {
			http.Redirect(w, r, "/notfound", http.StatusFound)
			return
		}
	}
}

// Проверка админа
func AdminMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		session, _ := store.Get(r, sessionName)
		user_id, ok := session.Values["user_id"]
		if !ok {
			http.Redirect(w, r, "/notfound", http.StatusFound)
			return
		}
		if service.CheckAdminRole(user_id.(int)) {
			next.ServeHTTP(w, r)
		} else {
			http.Redirect(w, r, "/notfound", http.StatusFound)
			return
		}
	}
}

func ServeModeratorPage(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, sessionName)
	userID, ok := session.Values["user_id"]
	if !ok {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	user, err := service.GetUserByID(userID.(int))
	if err != nil {
		log.Println("Не удалось получить пользователя из БД" + err.Error())
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	tmpl, err := template.New("moderator.html").Funcs(template.FuncMap{
		"isVideo": func(filename string) bool {
			ext := strings.ToLower(filepath.Ext(filename))
			return ext == ".mp4" || ext == ".mov" || ext == ".avi" || ext == ".webm"
		},
		"formatViews":    service.FormatViews,
		"checkModRole":   service.CheckModeratorOrAdminRole,
		"checkAdminRole": service.CheckAdminRole,
		"safeHTML": func(s string) template.HTML {
			return template.HTML(s)
		},
		"hasNotifications": service.HasNotifications,
	}).ParseFiles("templates/moderator.html")
	if err != nil {
		log.Println(err.Error())
	}

	err = tmpl.Execute(w, user)
	if err != nil {
		log.Println(err.Error())
	}
}

func ServeAdminPage(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, sessionName)
	userID, ok := session.Values["user_id"]
	if !ok {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	user, err := service.GetUserByID(userID.(int))
	if err != nil {
		log.Println("Не удалось получить пользователя из БД" + err.Error())
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	tmpl, err := template.New("admin.html").Funcs(template.FuncMap{
		"isVideo": func(filename string) bool {
			ext := strings.ToLower(filepath.Ext(filename))
			return ext == ".mp4" || ext == ".mov" || ext == ".avi"
		},
		"formatViews":      service.FormatViews,
		"checkModRole":     service.CheckModeratorOrAdminRole,
		"checkAdminRole":   service.CheckAdminRole,
		"hasNotifications": service.HasNotifications,
	}).ParseFiles("templates/admin.html")
	if err != nil {
		log.Println(err.Error())
	}

	stats, err := service.GetStats()
	if err != nil {
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	data := struct {
		User  *models.User
		Stats models.Stats
	}{
		User:  user,
		Stats: stats,
	}

	err = tmpl.Execute(w, data)
	if err != nil {
		log.Println(err.Error())
	}
}

func GetModerationPostsHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	page, err := strconv.Atoi(vars["page"])
	if err != nil {
		http.Error(w, "Wrong request", http.StatusInternalServerError)
		return
	}

	if page < 1 {
		page = 1
	}

	limit := 10
	offset := (page - 1) * limit

	posts, err := service.GetModerationPosts(limit, offset)
	if err != nil {
		http.Error(w, "Wrong request", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(posts)
}

func ApprovePostHandler(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, sessionName)
	userID, ok := session.Values["user_id"]
	if !ok {
		http.Error(w, "Wrong request", http.StatusInternalServerError)
		return
	}
	vars := mux.Vars(r)
	post_id, err := strconv.Atoi(vars["id"])
	if err != nil {
		http.Error(w, "Wrong request", http.StatusInternalServerError)
		return
	}

	err = service.ApprovePost(userID.(int), post_id)
	if err != nil {
		http.Error(w, "Wrong request", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func RejectPostHandler(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, sessionName)
	userID, ok := session.Values["user_id"]
	if !ok {
		http.Error(w, "Wrong request", http.StatusInternalServerError)
		return
	}
	vars := mux.Vars(r)
	post_id, err := strconv.Atoi(vars["id"])
	if err != nil {
		http.Error(w, "Wrong request: "+err.Error(), http.StatusInternalServerError)
		return
	}

	err = service.RejectPost(userID.(int), post_id)
	if err != nil {
		http.Error(w, "Wrong request: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func DeletePostHandler(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, sessionName)
	userID, ok := session.Values["user_id"]
	if !ok {
		http.Error(w, "Wrong request", http.StatusInternalServerError)
		return
	}
	vars := mux.Vars(r)
	post_id, err := strconv.Atoi(vars["id"])
	if err != nil {
		http.Error(w, "Wrong request: "+err.Error(), http.StatusInternalServerError)
		return
	}

	err = service.DeletePost(userID.(int), post_id)
	if err != nil {
		http.Error(w, "Wrong request: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func BanUserHandler(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, sessionName)
	userID, ok := session.Values["user_id"]
	if !ok {
		http.Error(w, "Wrong request", http.StatusInternalServerError)
		return
	}
	vars := mux.Vars(r)
	user_id, err := strconv.Atoi(vars["id"])
	if err != nil {
		http.Error(w, "Wrong request", http.StatusInternalServerError)
		log.Println(err.Error())
		return
	}

	err = service.BanUser(userID.(int), user_id)
	if err != nil {
		http.Error(w, "Wrong request", http.StatusInternalServerError)
		log.Println(err.Error())
		return
	}

	w.WriteHeader(http.StatusOK)
}

func BanUsernameHandler(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, sessionName)
	userID, ok := session.Values["user_id"]
	if !ok {
		http.Error(w, "Wrong request", http.StatusInternalServerError)
		return
	}
	vars := mux.Vars(r)
	username := vars["username"]

	target, err := service.GetUserByUsername(username)
	if err != nil {
		if !ok {
			http.Error(w, "Wrong request", http.StatusInternalServerError)
			return
		}
	}

	switch r.Method {
	case "POST":
		err = service.BanUser(userID.(int), target.ID)
		if err != nil {
			http.Error(w, "Wrong request", http.StatusInternalServerError)
			log.Println(err.Error())
			return
		}
	case "DELETE":
		err = service.UnbanUser(userID.(int), target.ID)
		if err != nil {
			http.Error(w, "Wrong request", http.StatusInternalServerError)
			log.Println(err.Error())
			return
		}
	}

	w.WriteHeader(http.StatusOK)
}

func ServeMainPage(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, sessionName)
	userID, ok := session.Values["user_id"]
	if !ok {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	user, err := service.GetUserByID(userID.(int))
	if err != nil {
		log.Println("Не удалось получить пользователя из БД c ID " + strconv.Itoa(userID.(int)) + " " + err.Error())
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	last_files, err := service.GetLastFilesWithAuthors()
	if err != nil {
		log.Println("Ошибка получения файлов:", err)
		last_files = []models.FileWithAuthor{} // Пустой список при ошибке
	}

	popular_files, err := service.GetMostPopularWithAuthors()
	if err != nil {
		log.Println("Ошибка получения файлов:", err)
		popular_files = []models.FileWithAuthor{} // Пустой список при ошибке
	}

	last_seen_files, err := service.GetLastSeenFilesWithAuthors(user.ID)
	if err != nil {
		log.Println("Ошибка получения файлов:", err)
		last_seen_files = []models.FileWithAuthor{} // Пустой список при ошибке
	}

	new_followings, err := service.GetNewFollowings(user.ID)
	if err != nil {
		log.Println("Ошибка получения файлов:", err)
		new_followings = []models.FileWithAuthor{} // Пустой список при ошибке
	}

	data := struct {
		User             *models.User            `json:"user"`
		MostPopularFiles []models.FileWithAuthor `json:"popular_files"`
		LastFiles        []models.FileWithAuthor `json:"files"`
		LastSeen         []models.FileWithAuthor `json:"last_seen"`
		Followings       []models.FileWithAuthor `json:"followings"`
	}{
		User:             user,
		MostPopularFiles: popular_files,
		LastFiles:        last_files,
		LastSeen:         last_seen_files,
		Followings:       new_followings,
	}

	tmpl, err := template.New("main.html").Funcs(template.FuncMap{
		"isVideo": func(filename string) bool {
			ext := strings.ToLower(filepath.Ext(filename))
			return ext == ".mp4" || ext == ".mov" || ext == ".avi"
		},
		"formatViews":      service.FormatViews,
		"checkModRole":     service.CheckModeratorOrAdminRole,
		"checkAdminRole":   service.CheckAdminRole,
		"hasNotifications": service.HasNotifications,
		"hasFollowings":    service.HasFollowings,
		"formatLikes":      service.FormatLikes,
		"formatValue":      service.FormatValue,
	}).ParseFiles("templates/main.html")
	if err != nil {
		log.Println(err.Error())
	}
	err = tmpl.Execute(w, data)
	if err != nil {
		log.Println(err.Error())
	}
}

func ServeFeedPage(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, sessionName)
	userID, ok := session.Values["user_id"]
	if !ok {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	user, err := service.GetUserByID(userID.(int))
	if err != nil {
		log.Println("Не удалось получить пользователя из БД" + err.Error())
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	tmpl, err := template.New("feed.html").Funcs(template.FuncMap{
		"isVideo": func(filename string) bool {
			ext := strings.ToLower(filepath.Ext(filename))
			return ext == ".mp4" || ext == ".mov" || ext == ".avi"
		},
		"formatViews":    service.FormatViews,
		"checkModRole":   service.CheckModeratorOrAdminRole,
		"checkAdminRole": service.CheckAdminRole,
		"safeHTML": func(s string) template.HTML {
			return template.HTML(s)
		},
		"hasNotifications": service.HasNotifications,
	}).ParseFiles("templates/feed.html")
	if err != nil {
		log.Println(err.Error())
	}

	err = tmpl.Execute(w, user)
	if err != nil {
		log.Println(err.Error())
	}
}

func ServeUploadPage(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, sessionName)
	userID, ok := session.Values["user_id"]
	if !ok {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	user, err := service.GetUserByID(userID.(int))
	if err != nil {
		log.Println("Не удалось получить пользователя из БД" + err.Error())
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	tmpl, err := template.New("upload.html").Funcs(template.FuncMap{
		"isVideo": func(filename string) bool {
			ext := strings.ToLower(filepath.Ext(filename))
			return ext == ".mp4" || ext == ".mov" || ext == ".avi"
		},
		"formatViews":      service.FormatViews,
		"checkModRole":     service.CheckModeratorOrAdminRole,
		"checkAdminRole":   service.CheckAdminRole,
		"hasNotifications": service.HasNotifications,
	}).ParseFiles("templates/upload.html")
	if err != nil {
		log.Println(err.Error())
	}

	err = tmpl.Execute(w, user)
	if err != nil {
		log.Println(err.Error())
	}
}

func LogoutHandler(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, sessionName)
	session.Options.MaxAge = -1
	session.Save(r, w)
	http.Redirect(w, r, "/", http.StatusFound)
}

func GenerateTwitchAccessToken(clientID, clientSecret string) (string, error) {
	// URL для запроса токена
	authURL := "https://id.twitch.tv/oauth2/token"

	// Формируем данные для POST-запроса
	data := url.Values{}
	data.Set("client_id", clientID)
	data.Set("client_secret", clientSecret)
	data.Set("grant_type", "client_credentials")

	// Создаем HTTP-запрос
	req, err := http.NewRequest("POST", authURL, strings.NewReader(data.Encode()))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	// Устанавливаем заголовки
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	// Отправляем запрос
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("authentication failed: %w", err)
	}
	defer resp.Body.Close()

	// Проверяем статус ответа
	if resp.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(resp.Body)
		return "", fmt.Errorf("twitch auth error (%d): %s", resp.StatusCode, string(body))
	}

	// Парсим ответ
	var tokenData models.TwitchTokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenData); err != nil {
		return "", fmt.Errorf("failed to parse token response: %w", err)
	}

	return tokenData.AccessToken, nil
}

func UploadClipHandler(w http.ResponseWriter, r *http.Request) {
	title := r.FormValue("title")
	if title == "" {
		http.Error(w, "Название обязательно", http.StatusBadRequest)
		return
	}

	clipURL := r.FormValue("clipUrl")
	if clipURL == "" {
		http.Error(w, "Ссылка обязательна", http.StatusBadRequest)
		return
	}

	description := r.FormValue("description")

	parsedURL, err := url.Parse(clipURL)
	if err != nil {
		http.Error(w, "Internal error", http.StatusBadRequest)
		return
	}

	// Разбиваем путь на сегменты
	segments := strings.Split(strings.Trim(parsedURL.Path, "/"), "/")
	if len(segments) == 0 {
		http.Error(w, "Internal error", http.StatusBadRequest)
		return
	}

	// Извлекаем идентификатор клипа (последний сегмент пути)
	clipID := segments[len(segments)-1]

	// Валидация идентификатора
	validID := regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)
	if !validID.MatchString(clipID) {
		http.Error(w, "Internal error", http.StatusBadRequest)
		return
	}

	// Формируем HTML для встраивания
	embedHTML := fmt.Sprintf(
		`<iframe src="https://clips.twitch.tv/embed?clip=%s&parent=ehworld.ru" `+
			`width="640" height="360" frameborder="0" `+
			`allowfullscreen="true" scrolling="no"></iframe>`,
		clipID)

	thumbnail, err := GetClipThumbnail(clipID)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	session, _ := store.Get(r, sessionName)
	userID, ok := session.Values["user_id"]
	if !ok {
		http.Error(w, "Неавторизованный доступ", http.StatusUnauthorized)
		return
	}

	// Сохраняем информацию в БД
	fileInfo := models.File{
		UserID:      userID.(int),
		Title:       title,
		FileName:    embedHTML,
		Thumbnail:   thumbnail,
		IsPublic:    false,
		Description: description,
	}

	id, err := service.SaveClip(&fileInfo)
	if err != nil {
		http.Error(w, "Internal error", http.StatusBadRequest)
		return
	}

	data := struct {
		Id int `json:"id"`
	}{
		Id: id,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

func GetClipThumbnail(clipID string) (string, error) {
	// Создаем HTTP-запрос
	req, err := http.NewRequest("GET", "https://api.twitch.tv/helix/clips", nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	accessToken, err := GenerateTwitchAccessToken(oauthConfig.ClientID, oauthConfig.ClientSecret)
	if err != nil {
		return "", nil
	}

	// Добавляем параметры и заголовки
	q := req.URL.Query()
	q.Add("id", clipID)
	req.URL.RawQuery = q.Encode()
	req.Header.Set("Client-Id", oauthConfig.ClientID)
	req.Header.Set("Authorization", "Bearer "+accessToken)

	// Выполняем запрос
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("API request failed: %w", err)
	}
	defer resp.Body.Close()

	// Проверяем статус ответа
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("twitch API error: %s", resp.Status)
	}

	// Читаем и парсим ответ
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	var clipData models.ClipResponse
	if err := json.Unmarshal(body, &clipData); err != nil {
		return "", fmt.Errorf("failed to parse JSON: %w", err)
	}

	// Проверяем наличие данных
	if len(clipData.Data) == 0 {
		return "", fmt.Errorf("clip data not found")
	}

	return clipData.Data[0].ThumbnailURL, nil
}

func UploadHandler(w http.ResponseWriter, r *http.Request) {
	// Ограничиваем размер файла (100 MB)
	r.ParseMultipartForm(100 << 20)

	file, handler, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "Ошибка при получении файла", http.StatusBadRequest)
		return
	}
	defer file.Close()

	title := r.FormValue("title")
	if title == "" {
		http.Error(w, "Название обязательно", http.StatusBadRequest)
		return
	}

	description := r.FormValue("description")

	session, _ := store.Get(r, sessionName)
	userID, ok := session.Values["user_id"]
	if !ok {
		http.Error(w, "Неавторизованный доступ", http.StatusUnauthorized)
		return
	}

	// Создаем директорию для загрузок
	os.MkdirAll("./static/uploads", os.ModePerm)

	// Генерируем уникальное имя файла
	newFileName := service.GenerateUniqueFileName(handler.Filename)
	filePath := "./static/uploads/" + newFileName

	// Сохраняем файл
	dst, err := os.Create(filePath)
	if err != nil {
		http.Error(w, "Ошибка сохранения файла", http.StatusInternalServerError)
		return
	}
	defer dst.Close()

	if _, err := io.Copy(dst, file); err != nil {
		http.Error(w, "Ошибка копирования файла", http.StatusInternalServerError)
		return
	}

	// Сохраняем информацию в БД
	fileInfo := models.File{
		UserID:      userID.(int),
		Title:       title,
		FileName:    newFileName,
		FileSize:    handler.Size,
		IsPublic:    false,
		Description: description,
	}

	if service.IsVideoFile(newFileName) {
		thumbFileName := "thumb_" + strings.ReplaceAll(
			strings.TrimSuffix(newFileName, filepath.Ext(newFileName)),
			" ", "_") + ".jpg"
		thumbPath := "./static/uploads/" + thumbFileName

		if err := service.GenerateThumbnail(filePath, thumbPath); err == nil {
			fileInfo.Thumbnail = thumbFileName
		} else {
			log.Printf("Ошибка генерации превью: %v", err)
		}
	}
	id, err := service.SaveFile(&fileInfo)
	if err != nil {
		go os.Remove(filePath)
		http.Error(w, "Ошибка сохранения информации", http.StatusInternalServerError)
		return
	}

	data := struct {
		Id int `json:"id"`
	}{
		Id: id,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

func ServePostPage(w http.ResponseWriter, r *http.Request) {
	var authorised bool
	vars := mux.Vars(r)
	fileID, err := strconv.Atoi(vars["id"])
	if err != nil {
		http.Redirect(w, r, "/notfound", http.StatusFound)
		return
	}
	session, _ := store.Get(r, sessionName)
	userID, ok := session.Values["user_id"].(int)
	var user *models.User
	if !ok {
		authorised = false
	} else {
		authorised = true
		user, _ = service.GetUserByID(userID)
	}

	// Увеличить просмотры
	service.IncrementViews(fileID)

	// Получить данные
	var file *models.FileWithAuthor
	var comments []models.CommentWithAuthor
	var hasLiked, hasFucked bool
	if authorised {
		file, err = service.GetFileByIDAuthorised(fileID, userID)
		if err != nil {
			log.Println(err.Error())
			http.Redirect(w, r, "/notfound", http.StatusNotFound)
			return
		}
		comments, err = service.GetComments(user.ID, fileID)
		if err != nil {
			comments = []models.CommentWithAuthor{}
		}
		hasLiked, _ = service.HasLiked(userID, fileID)
		hasFucked, _ = service.HasFuckYou(userID, fileID)
	} else {
		file, err = service.GetFileByID(fileID)
		if err != nil {
			http.Redirect(w, r, "/notfound", http.StatusNotFound)
			return
		}
		comments, err = service.GetCommentsUnauthorised(fileID)
		if err != nil {
			comments = []models.CommentWithAuthor{}
		}
	}

	data := struct {
		User       *models.User
		File       *models.FileWithAuthor
		Comments   []models.CommentWithAuthor
		HasLiked   bool
		HasFuckYou bool
	}{
		User:       user,
		File:       file,
		Comments:   comments,
		HasLiked:   hasLiked,
		HasFuckYou: hasFucked,
	}

	if ok {
		tmpl, err := template.New("post.html").Funcs(template.FuncMap{
			"isVideo": func(filename string) bool {
				ext := strings.ToLower(filepath.Ext(filename))
				return ext == ".mp4" || ext == ".mov" || ext == ".avi" || ext == ".webm"
			},
			"formatTimeAgo":  service.FormatTimeAgo,
			"formatViews":    service.FormatViews,
			"checkModRole":   service.CheckModeratorOrAdminRole,
			"checkAdminRole": service.CheckAdminRole,
			"safeHTML": func(s string) template.HTML {
				return template.HTML(s)
			},
			"hasDescription":   service.HasDescription,
			"hasNotifications": service.HasNotifications,
			"hasBadge":         service.HasBadge,
		}).ParseFiles("templates/post.html")
		if err != nil {
			log.Println(err.Error())
		}
		err = tmpl.Execute(w, data)
		if err != nil {
			log.Println(err.Error())
		}
	} else {
		tmpl, err := template.New("postunauthorised.html").Funcs(template.FuncMap{
			"isVideo": func(filename string) bool {
				ext := strings.ToLower(filepath.Ext(filename))
				return ext == ".mp4" || ext == ".mov" || ext == ".avi" || ext == ".webm"
			},
			"formatTimeAgo":  service.FormatTimeAgo,
			"formatViews":    service.FormatViews,
			"checkModRole":   service.CheckModeratorOrAdminRole,
			"checkAdminRole": service.CheckAdminRole,
			"safeHTML": func(s string) template.HTML {
				return template.HTML(s)
			},
			"hasDescription":   service.HasDescription,
			"hasNotifications": service.HasNotifications,
			"hasBadge":         service.HasBadge,
		}).ParseFiles("templates/postunauthorised.html")
		if err != nil {
			log.Println(err.Error())
		}
		err = tmpl.Execute(w, data)
		if err != nil {
			log.Println(err.Error())
		}
	}

}

func LikeHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	fileID, _ := strconv.Atoi(vars["id"])

	session, _ := store.Get(r, sessionName)
	userID, _ := session.Values["user_id"].(int)

	if r.Method == "POST" {
		err := service.LikeFile(userID, fileID)
		if err != nil {
			log.Println(err.Error())
			http.Error(w, "Internal error", http.StatusInternalServerError)
			return
		}
	} else {
		err := service.UnlikeFile(userID, fileID)
		if err != nil {
			log.Println(err.Error())
			http.Error(w, "Internal error", http.StatusInternalServerError)
			return
		}
	}

	w.WriteHeader(http.StatusOK)
}

func FuckYouHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	fileID, _ := strconv.Atoi(vars["id"])

	session, _ := store.Get(r, sessionName)
	userID, _ := session.Values["user_id"].(int)

	if r.Method == "POST" {
		err := service.FuckYouFile(userID, fileID)
		if err != nil {
			http.Error(w, "Internal error", http.StatusInternalServerError)
			log.Println(err.Error())
			return
		}
	} else {
		err := service.UnFuckYouFile(userID, fileID)
		if err != nil {
			http.Error(w, "Internal error", http.StatusInternalServerError)
			log.Println(err.Error())
			return
		}
	}

	w.WriteHeader(http.StatusOK)
}

func LikeCommentHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	fileID, _ := strconv.Atoi(vars["id"])

	session, _ := store.Get(r, sessionName)
	userID, _ := session.Values["user_id"].(int)

	if r.Method == "POST" {
		err := service.LikeComment(userID, fileID)
		if err != nil {
			log.Println("Не удалось поставить лайк комментарию: " + err.Error())
			http.Error(w, "Internal error", http.StatusInternalServerError)
			return
		}
	} else {
		err := service.UnlikeComment(userID, fileID)
		if err != nil {
			log.Println("Не удалось убрать лайк с комментария: " + err.Error())
			http.Error(w, "Internal error", http.StatusInternalServerError)
			return
		}
	}

	w.WriteHeader(http.StatusOK)
}

func CommentHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	fileID, _ := strconv.Atoi(vars["id"])

	session, _ := store.Get(r, sessionName)
	userID, _ := session.Values["user_id"].(int)

	text := r.FormValue("text")
	parentID, _ := strconv.Atoi(r.FormValue("parent_id"))

	comment := models.Comment{
		UserID:   userID,
		FileID:   fileID,
		Text:     text,
		ParentID: parentID,
	}

	switch r.Method {
	case "POST":
		id, _ := service.AddComment(&comment)
		user, _ := service.GetUserByID(userID)
		comment.AuthorName = user.DisplayName
		comment.ID = id
		com_wth_author := models.CommentWithAuthor{
			Comment:               comment,
			AuthorProfileImageURL: user.ProfileImageURL,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(com_wth_author)

	case "DELETE":
		// Проверка прав
		comment.ID = fileID
		comment.UserID = userID
		if !service.IsCommentOwner(&comment) {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}
		err := service.DeleteComment(&comment)
		if err != nil {
			http.Error(w, err.Error(), http.StatusForbidden)
			return
		}

	case "PUT":
		if !service.IsCommentOwner(&comment) {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}

		err := service.UpdateComment(&comment)
		if err != nil {
			http.Error(w, err.Error(), http.StatusForbidden)
			return
		}
		json.NewEncoder(w).Encode(comment)
	}

	// Возвращаем новый комментарий
	// w.Header().Set("Content-Type", "application/json")
	// json.NewEncoder(w).Encode(comment)
}

func ServeNotFoundPage(w http.ResponseWriter, r *http.Request) {
	authorized := true
	session, _ := store.Get(r, sessionName)
	userID, ok := session.Values["user_id"]
	if !ok {
		authorized = false
	}

	if authorized {
		user, err := service.GetUserByID(userID.(int))
		if err != nil {
			log.Println("Не удалось получить пользователя из БД" + err.Error())
			http.Redirect(w, r, "/notfound", http.StatusFound)
			return
		}

		data := struct {
			User *models.User `json:"user"`
		}{
			User: user,
		}

		tmpl, err := template.New("notfoundauthorised.html").Funcs(template.FuncMap{
			"checkModRole":     service.CheckModeratorOrAdminRole,
			"checkAdminRole":   service.CheckAdminRole,
			"hasNotifications": service.HasNotifications,
		}).ParseFiles("templates/notfoundauthorised.html")
		if err != nil {
			log.Println(err.Error())
		}
		err = tmpl.Execute(w, data)
		if err != nil {
			log.Println(err.Error())
		}
	} else {
		tmpl, err := template.New("notfoundunauthorised.html").Funcs(template.FuncMap{
			"isVideo": func(filename string) bool {
				ext := strings.ToLower(filepath.Ext(filename))
				return ext == ".mp4" || ext == ".mov" || ext == ".avi"
			},
			"formatTimeAgo": service.FormatTimeAgo,
			"formatViews":   service.FormatViews,
		}).ParseFiles("templates/notfoundunauthorised.html")
		if err != nil {
			log.Println(err.Error())
		}
		err = tmpl.Execute(w, nil)
		if err != nil {
			log.Println(err.Error())
		}
	}
}

func ServeUserPageHandler(w http.ResponseWriter, r *http.Request) {
	authorized := true
	session, _ := store.Get(r, sessionName)
	userID, ok := session.Values["user_id"]
	if !ok {
		authorized = false
	}

	vars := mux.Vars(r)
	profileUsername := vars["username"]

	if authorized {
		user, err := service.GetUserByID(userID.(int))
		if err != nil {
			log.Println("Не удалось получить пользователя из БД" + err.Error())
			http.Redirect(w, r, "/notfound", http.StatusFound)
			return
		}

		profileUser, err := service.GetUserByUsername(profileUsername)
		if err != nil {
			log.Println("Не удалось получить пользователя из БД" + err.Error())
			http.Redirect(w, r, "/notfound", http.StatusFound)
			return
		}

		stats := struct {
			Likes int
			Fucks int
			Posts int
		}{
			Likes: service.GetTotalLikes(profileUser.ID),
			Fucks: service.GetTotalFucks(profileUser.ID),
			Posts: service.GetTotalPosts(profileUser.ID),
		}

		badges, err := service.GetUserBadges(profileUser.ID)
		if err != nil {
			log.Println("Не удалось получить значки" + err.Error())
			http.Redirect(w, r, "/notfound", http.StatusFound)
			return
		}

		isFollowing := false
		if user.ID != profileUser.ID {
			isFollowing = service.CheckFollow(user.ID, profileUser.ID)
		}

		data := struct {
			User        *models.User
			ProfileUser *models.User
			Stats       interface{}
			Badges      []models.Badge
			IsFollowing bool
		}{
			User:        user,
			ProfileUser: profileUser,
			Stats:       stats,
			Badges:      badges,
			IsFollowing: isFollowing,
		}

		tmpl, err := template.New("user.html").Funcs(template.FuncMap{
			"isVideo": func(filename string) bool {
				ext := strings.ToLower(filepath.Ext(filename))
				return ext == ".mp4" || ext == ".mov" || ext == ".avi"
			},
			"formatTimeAgo":    service.FormatTimeAgo,
			"formatViews":      service.FormatViews,
			"checkModRole":     service.CheckModeratorOrAdminRole,
			"checkAdminRole":   service.CheckAdminRole,
			"hasNotifications": service.HasNotifications,
			"formatFollowers":  service.FormatFollowers,
			"hasBadge":         service.HasBadge,
		}).ParseFiles("templates/user.html")
		if err != nil {
			log.Println(err.Error())
		}
		err = tmpl.Execute(w, data)
		if err != nil {
			log.Println(err.Error())
		}
	} else {
		profileUser, err := service.GetUserByUsername(profileUsername)
		if err != nil {
			log.Println("Не удалось получить пользователя из БД" + err.Error())
			http.Redirect(w, r, "/notfound", http.StatusFound)
			return
		}

		stats := struct {
			Likes int
			Fucks int
			Posts int
		}{
			Likes: service.GetTotalLikes(profileUser.ID),
			Fucks: service.GetTotalFucks(profileUser.ID),
			Posts: service.GetTotalPosts(profileUser.ID),
		}

		badges, err := service.GetUserBadges(profileUser.ID)
		if err != nil {
			log.Println("Не удалось получить значки" + err.Error())
			http.Redirect(w, r, "/notfound", http.StatusFound)
			return
		}

		data := struct {
			ProfileUser *models.User
			Stats       interface{}
			Badges      []models.Badge
		}{
			ProfileUser: profileUser,
			Stats:       stats,
			Badges:      badges,
		}

		tmpl, err := template.New("userunauthorised.html").Funcs(template.FuncMap{
			"isVideo": func(filename string) bool {
				ext := strings.ToLower(filepath.Ext(filename))
				return ext == ".mp4" || ext == ".mov" || ext == ".avi"
			},
			"formatTimeAgo":   service.FormatTimeAgo,
			"formatViews":     service.FormatViews,
			"formatFollowers": service.FormatFollowers,
			"hasBadge":        service.HasBadge,
		}).ParseFiles("templates/userunauthorised.html")
		if err != nil {
			log.Println(err.Error())
		}
		err = tmpl.Execute(w, data)
		if err != nil {
			log.Println(err.Error())
		}
	}
}

func ServeInventoryPage(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, sessionName)
	userID, ok := session.Values["user_id"]
	if !ok {
		http.Redirect(w, r, "/", http.StatusFound)
	}

	user, err := service.GetUserByID(userID.(int))
	if err != nil {
		log.Println("Не удалось получить пользователя из БД" + err.Error())
		http.Redirect(w, r, "/notfound", http.StatusFound)
		return
	}

	badges, err := service.GetUserBadges(user.ID)
	if err != nil {
		http.Redirect(w, r, "/notfound", http.StatusFound)
		return
	}

	items, err := service.GetUserInventory(user.ID)
	if err != nil {
		http.Redirect(w, r, "/notfound", http.StatusFound)
		return
	}

	data := struct {
		User   *models.User
		Badges []models.Badge
		Items  []models.CaseReward
	}{
		User:   user,
		Badges: badges,
		Items:  items,
	}

	tmpl, err := template.New("inventory.html").Funcs(template.FuncMap{
		"isVideo": func(filename string) bool {
			ext := strings.ToLower(filepath.Ext(filename))
			return ext == ".mp4" || ext == ".mov" || ext == ".avi"
		},
		"formatTimeAgo":    service.FormatTimeAgo,
		"formatViews":      service.FormatViews,
		"checkModRole":     service.CheckModeratorOrAdminRole,
		"checkAdminRole":   service.CheckAdminRole,
		"hasNotifications": service.HasNotifications,
		"formatFollowers":  service.FormatFollowers,
		"hasBadge":         service.HasBadge,
	}).ParseFiles("templates/inventory.html")
	if err != nil {
		log.Println(err.Error())
	}
	err = tmpl.Execute(w, data)
	if err != nil {
		log.Println(err.Error())
	}
}

func GetUserPostsHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userID, _ := strconv.Atoi(vars["id"])

	query := r.URL.Query()
	page, _ := strconv.Atoi(query.Get("page"))
	limit, _ := strconv.Atoi(query.Get("limit"))
	contentType := query.Get("type")
	sort := query.Get("sort")
	search := query.Get("search")

	// Параметры по умолчанию
	if page < 1 {
		page = 1
	}
	if limit == 0 {
		limit = 12
	}

	// Получение пользователя
	user, err := service.GetUserByID(userID)
	if err != nil {
		log.Println(err.Error())
		http.Error(w, "Internal error getting user", http.StatusInternalServerError)
		return
	}

	// Получение постов
	posts, _, err := service.GetUserPosts(user.ID, page, limit, contentType, sort, search)
	if err != nil {
		log.Println(err.Error())
		http.Error(w, "Internal error getting posts", http.StatusInternalServerError)
		return
	}
	total := service.GetTotalPosts(user.ID)

	response := struct {
		Posts      []models.File
		Total      int
		Page       int
		TotalPages int
	}{
		Posts:      posts,
		Total:      total,
		Page:       page,
		TotalPages: int(math.Ceil(float64(total) / float64(limit))),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func SubscribeHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	targetID, _ := strconv.Atoi(vars["id"])

	session, _ := store.Get(r, sessionName)
	userID, ok := session.Values["user_id"].(int)
	if !ok {
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	switch r.Method {
	case "POST":
		err := service.Subscribe(userID, targetID)
		if err != nil {
			log.Println(err.Error())
			http.Error(w, "Internal error", http.StatusInternalServerError)
			return
		}
	case "DELETE":
		err := service.Unsubscribe(userID, targetID)
		if err != nil {
			log.Println(err.Error())
			http.Error(w, "Internal error", http.StatusInternalServerError)
			return
		}
	}

	w.WriteHeader(http.StatusOK)
}

func ServeShopPage(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, sessionName)
	userID, ok := session.Values["user_id"]
	if !ok {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	user, err := service.GetUserByID(userID.(int))
	if err != nil {
		log.Println("Не удалось получить пользователя из БД" + err.Error())
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	items := service.GetShopItems(user.ID)

	cases := service.GetCases(user.ID)

	tmpl, err := template.New("shop.html").Funcs(template.FuncMap{
		"isVideo": func(filename string) bool {
			ext := strings.ToLower(filepath.Ext(filename))
			return ext == ".mp4" || ext == ".mov" || ext == ".avi"
		},
		"formatViews":    service.FormatViews,
		"checkModRole":   service.CheckModeratorOrAdminRole,
		"checkAdminRole": service.CheckAdminRole,
		"safeHTML": func(s string) template.HTML {
			return template.HTML(s)
		},
		"hasNotifications": service.HasNotifications,
		"isVIP":            service.IsVIP,
		"hasBadge":         service.HasBadge,
	}).ParseFiles("templates/shop.html")
	if err != nil {
		log.Println(err.Error())
	}

	data := struct {
		User  *models.User
		Items []models.ShopItem
		Cases []models.Case
	}{
		User:  user,
		Items: items,
		Cases: cases,
	}

	err = tmpl.Execute(w, data)
	if err != nil {
		log.Println(err.Error())
	}
}

func BuyItemHandler(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, sessionName)
	userID, ok := session.Values["user_id"]
	if !ok {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	vars := mux.Vars(r)
	itemID, err := strconv.Atoi(vars["id"])
	if err != nil {
		http.Error(w, "Wrong request "+err.Error(), http.StatusUnauthorized)
		log.Println("Wrong request (itemID) " + err.Error())
		return
	}

	err = service.BuyItem(userID.(int), itemID)
	if err != nil {
		http.Error(w, "Wrong request "+err.Error(), http.StatusUnauthorized)
		log.Println("Wrong request (buying item) " + err.Error())
		return
	}

	w.WriteHeader(http.StatusOK)
}

func FeedHandler(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, sessionName)
	userID, ok := session.Values["user_id"].(int)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit == 0 {
		limit = 10 // default value
	}

	posts, err := service.GetFeedPosts(userID, offset, limit)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(posts)
}

func LastFilesHandler(w http.ResponseWriter, r *http.Request) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 8
	}

	offset := (page - 1) * limit

	posts, err := service.GetLastPosts(limit, offset)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		log.Println(err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(posts)
}

func SearchHandler(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	if len(query) < 2 || len(query) > 100 { // Невалидный запрос
		json.NewEncoder(w).Encode([]models.SearchResult{})
		return
	}

	searchTerm := "%" + strings.ToLower(query) + "%"
	results := service.Search(searchTerm)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
}

func UsersSearchHandler(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	if len(query) < 2 || len(query) > 100 { // Невалидный запрос
		json.NewEncoder(w).Encode([]models.UserSearchResult{})
		return
	}

	searchTerm := "%" + strings.ToLower(query) + "%"
	results := service.SearchUsers(searchTerm)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
}

func ModeratorRoleHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	username := vars["username"]

	switch r.Method {
	case "POST":
		err := service.AddModerator(username)
		if err != nil {
			http.Error(w, "Wrong request: "+err.Error(), http.StatusInternalServerError)
			return
		}
	case "DELETE":
		err := service.DeleteModerator(username)
		if err != nil {
			http.Error(w, "Wrong request: "+err.Error(), http.StatusInternalServerError)
			return
		}
	}

	w.WriteHeader(http.StatusOK)
}

func GetModeratorsListHandler(w http.ResponseWriter, r *http.Request) {
	list := service.GetModeratorsList()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(list)
}

func GetNotificationsHandler(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, sessionName)
	userID, ok := session.Values["user_id"].(int)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	notifications, err := service.GetNotifications(userID)
	if err != nil {
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(notifications)
}

func GetBannedUsersListHandler(w http.ResponseWriter, r *http.Request) {
	users, err := service.GetBannedUsersList()
	if err != nil {
		http.Error(w, "Internal error", http.StatusInternalServerError)
		log.Println(err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(users)
}

func UploadBadgeHandler(w http.ResponseWriter, r *http.Request) {
	r.ParseMultipartForm(100 << 20)

	file, handler, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "Ошибка при получении файла", http.StatusBadRequest)
		return
	}
	defer file.Close()

	title := r.FormValue("title")
	if title == "" {
		http.Error(w, "Название обязательно", http.StatusBadRequest)
		return
	}

	cost := r.FormValue("cost")
	if title == "" {
		http.Error(w, "Цена обязательна", http.StatusBadRequest)
		return
	}

	session, _ := store.Get(r, sessionName)
	_, ok := session.Values["user_id"]
	if !ok {
		http.Error(w, "Неавторизованный доступ", http.StatusUnauthorized)
		return
	}

	// Создаем директорию для загрузок
	os.MkdirAll("./static/uploads/badges", os.ModePerm)

	// Генерируем уникальное имя файла
	newFileName := service.GenerateUniqueFileName(handler.Filename)
	filePath := "./static/uploads/badges/" + newFileName

	// Сохраняем файл
	dst, err := os.Create(filePath)
	if err != nil {
		log.Println("Failed to create file: " + err.Error())
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}
	defer dst.Close()

	if _, err := io.Copy(dst, file); err != nil {
		log.Println("Failed to copy file: " + err.Error())
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	costValue, err := strconv.Atoi(cost)
	if err != nil {
		http.Error(w, "Стоимость должна быть числом", http.StatusInternalServerError)
		return
	}

	err = service.SaveBadge("."+filePath, title, costValue)
	if err != nil {
		go os.Remove(filePath)
		log.Println("Failed to save file to database: " + err.Error())
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func AddCaseHandler(w http.ResponseWriter, r *http.Request) {
	r.ParseMultipartForm(100 << 20)

	file, handler, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "Ошибка при получении файла", http.StatusBadRequest)
		return
	}
	defer file.Close()

	title := r.FormValue("title")
	if title == "" {
		http.Error(w, "Название обязательно", http.StatusBadRequest)
		return
	}

	description := r.FormValue("description")

	price := r.FormValue("price")
	if title == "" {
		http.Error(w, "Цена обязательна", http.StatusBadRequest)
		return
	}

	// Создаем директорию для загрузок
	os.MkdirAll("./static/uploads/cases", os.ModePerm)

	// Генерируем уникальное имя файла
	newFileName := service.GenerateUniqueFileName(handler.Filename)
	filePath := "./static/uploads/cases/" + newFileName

	// Сохраняем файл
	dst, err := os.Create(filePath)
	if err != nil {
		log.Println("Failed to create file: " + err.Error())
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}
	defer dst.Close()

	if _, err := io.Copy(dst, file); err != nil {
		log.Println("Failed to copy file: " + err.Error())
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	priceValue, err := strconv.Atoi(price)
	if err != nil {
		http.Error(w, "Стоимость должна быть числом", http.StatusInternalServerError)
		return
	}

	id, err := service.SaveCase("."+filePath, title, description, priceValue)
	if err != nil {
		go os.Remove(filePath)
		log.Println("Failed to save file to database: " + err.Error())
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	data := struct {
		Id int `json:"id"`
	}{
		Id: id,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

func GetBadgesHandler(w http.ResponseWriter, r *http.Request) {
	badges := service.GetBadges()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(badges)
}

func AddRewardsHandler(w http.ResponseWriter, r *http.Request) {
	rewardType := r.FormValue("type")
	if rewardType == "" {
		http.Error(w, "Тип обязателен", http.StatusBadRequest)
		return
	}

	caseID, err := strconv.Atoi(r.FormValue("case_id"))
	if err != nil {
		http.Error(w, "ID кейса обязателен", http.StatusBadRequest)
		return
	}

	probability := r.FormValue("probability")
	var probability_value float64
	if probability == "" {
		http.Error(w, "Шанс выпадения обязателен", http.StatusBadRequest)
		return
	} else {
		probability_value, err = strconv.ParseFloat(probability, 64)
		if err != nil {
			http.Error(w, "Internal error", http.StatusBadRequest)
			return
		}
	}

	badge := r.FormValue("badge_id")
	var badge_id, auk_value int
	if badge != "" {
		badge_id, err = strconv.Atoi(badge)
		if err != nil {
			http.Error(w, "Internal error", http.StatusBadRequest)
			log.Println("Error converting badge_id " + err.Error())
			return
		}
	}
	auk := r.FormValue("auk_value")
	if auk != "" {
		auk_value, err = strconv.Atoi(auk)
		if err != nil {
			http.Error(w, "Internal error", http.StatusBadRequest)
			log.Println("Error converting auk_value " + err.Error())
			return
		}
	}

	reward := models.CaseReward{
		Type:        rewardType,
		CaseID:      caseID,
		BadgeID:     badge_id,
		AukValue:    auk_value,
		Probability: probability_value,
	}

	err = service.AddReward(reward)
	if err != nil {
		http.Error(w, "Internal error", http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func GetCaseRewardsHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	caseID, _ := strconv.Atoi(vars["id"])

	rewards, err := service.GetCaseRewards(caseID)
	if err != nil {
		http.Error(w, "Internal error", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(rewards)
}

func OpenCaseHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	caseID, _ := strconv.Atoi(vars["id"])
	session, _ := store.Get(r, sessionName)
	userID, ok := session.Values["user_id"]
	if !ok {
		http.Error(w, "Unauthorised", http.StatusUnauthorized)
		return
	}

	reward, err := service.OpenCase(caseID, userID.(int))
	if err != nil {
		http.Error(w, "Internal error", http.StatusUnauthorized)
		log.Println("Failed to open case " + err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(reward)
}

func ApplyBadgeHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	badgeID, _ := strconv.Atoi(vars["id"])
	session, _ := store.Get(r, sessionName)
	userID, ok := session.Values["user_id"]
	if !ok {
		http.Error(w, "Unauthorised", http.StatusUnauthorized)
		return
	}

	err := service.ApplyBadge(badgeID, userID.(int))
	if err != nil {
		http.Error(w, "Internal error", http.StatusUnauthorized)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func ApplyItem(w http.ResponseWriter, r *http.Request) {
	r.ParseMultipartForm(100 << 20)
	vars := mux.Vars(r)
	itemID, _ := strconv.Atoi(vars["id"])
	session, _ := store.Get(r, sessionName)
	userID, ok := session.Values["user_id"]
	if !ok {
		http.Error(w, "Unauthorised", http.StatusUnauthorized)
		return
	}

	lot_name := r.FormValue("lot_name")

	err := service.ApplyItem(itemID, userID.(int), lot_name)
	if err != nil {
		http.Error(w, "Internal error", http.StatusUnauthorized)
		log.Println(err.Error())
		return
	}

	w.WriteHeader(http.StatusOK)
}

func ServeQueuePage(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, sessionName)
	userID, ok := session.Values["user_id"]
	if !ok {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	user, err := service.GetUserByID(userID.(int))
	if err != nil {
		log.Println("Не удалось получить пользователя из БД" + err.Error())
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	tmpl, err := template.New("queue.html").Funcs(template.FuncMap{
		"isVideo": func(filename string) bool {
			ext := strings.ToLower(filepath.Ext(filename))
			return ext == ".mp4" || ext == ".mov" || ext == ".avi"
		},
		"formatViews":    service.FormatViews,
		"checkModRole":   service.CheckModeratorOrAdminRole,
		"checkAdminRole": service.CheckAdminRole,
		"safeHTML": func(s string) template.HTML {
			return template.HTML(s)
		},
		"hasNotifications": service.HasNotifications,
		"isVIP":            service.IsVIP,
		"hasBadge":         service.HasBadge,
	}).ParseFiles("templates/queue.html")
	if err != nil {
		log.Println(err.Error())
	}

	data := struct {
		User *models.User
	}{
		User: user,
	}

	err = tmpl.Execute(w, data)
	if err != nil {
		log.Println(err.Error())
	}
}

func GetQueueHandler(w http.ResponseWriter, r *http.Request) {
	queue, err := service.GetQueue()
	if err != nil {
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(queue)
}

func DeleteSubmission(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	subID, _ := strconv.Atoi(vars["id"])

	err := service.DeleteSubmission(subID)
	if err != nil {
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func GetLiveChannelsHandler(w http.ResponseWriter, r *http.Request) {
	// Устанавливаем заголовки CORS
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	// Проверяем, нужно ли обновить кэш
	cacheMutex.RLock()
	needsUpdate := time.Since(lastUpdateTime) > cacheDuration || len(cachedLiveChannels) == 0
	cacheMutex.RUnlock()

	if needsUpdate {
		updateLiveChannelsCache()
	}

	// Получаем кэшированные данные
	cacheMutex.RLock()
	liveChannels := make([]models.TwitchStream, len(cachedLiveChannels))
	copy(liveChannels, cachedLiveChannels)
	cacheMutex.RUnlock()

	// Разделяем каналы на онлайн и оффлайн
	var onlineChannels []models.TwitchStream
	var offlineChannels []models.TwitchStream

	for _, channel := range liveChannels {
		if channel.IsLive {
			onlineChannels = append(onlineChannels, channel)
		} else {
			offlineChannels = append(offlineChannels, channel)
		}
	}

	// Применяем сортировку только к онлайн-каналам
	sortOrder := r.URL.Query().Get("sort")
	if sortOrder == "asc" {
		sort.Slice(onlineChannels, func(i, j int) bool {
			return onlineChannels[i].ViewerCount < onlineChannels[j].ViewerCount
		})
	} else {
		// По умолчанию сортируем по убыванию
		sort.Slice(onlineChannels, func(i, j int) bool {
			return onlineChannels[i].ViewerCount > onlineChannels[j].ViewerCount
		})
	}

	// Объединяем онлайн и оффлайн каналы
	result := append(onlineChannels, offlineChannels...)

	// Возвращаем данные
	json.NewEncoder(w).Encode(result)
}

func updateLiveChannelsCache() {
	var allChannels []models.TwitchStream
	var mutex sync.Mutex

	// Создаем клиент с таймаутом
	client := &http.Client{Timeout: 10 * time.Second}

	token, err := service.RefreshTwitchToken()
	if err != nil {
		return
	}

	// Получаем информацию о текущих стримах
	streamsURL := "https://api.twitch.tv/helix/streams?"
	for i, channel := range channelsToCheck {
		if i > 0 {
			streamsURL += "&"
		}
		streamsURL += "user_login=" + channel
	}

	reqStreams, err := http.NewRequest("GET", streamsURL, nil)
	if err != nil {
		fmt.Printf("Error creating streams request: %v\n", err)
		return
	}
	reqStreams.Header.Set("Client-ID", os.Getenv("TWITCH_CLIENT_ID_BOT"))
	reqStreams.Header.Set("Authorization", "Bearer "+token)

	respStreams, err := client.Do(reqStreams)
	if err != nil {
		fmt.Printf("Error making request to Twitch API (streams): %v\n", err)
		return
	}
	defer respStreams.Body.Close()

	bodyStreams, err := ioutil.ReadAll(respStreams.Body)
	if err != nil {
		fmt.Printf("Error reading streams response: %v\n", err)
		return
	}

	var streamResponse models.TwitchStreamResponse
	err = json.Unmarshal(bodyStreams, &streamResponse)
	if err != nil {
		fmt.Printf("Error parsing streams JSON: %v\n", err)
		return
	}

	// Получаем информацию о пользователях (для всех каналов)
	usersURL := "https://api.twitch.tv/helix/users?"
	for i, channel := range channelsToCheck {
		if i > 0 {
			usersURL += "&"
		}
		usersURL += "login=" + channel
	}

	reqUsers, err := http.NewRequest("GET", usersURL, nil)
	if err != nil {
		fmt.Printf("Error creating users request: %v\n", err)
		return
	}
	reqUsers.Header.Set("Client-ID", os.Getenv("TWITCH_CLIENT_ID_BOT"))
	reqUsers.Header.Set("Authorization", "Bearer "+token)

	respUsers, err := client.Do(reqUsers)
	if err != nil {
		fmt.Printf("Error making request to Twitch API (users): %v\n", err)
		return
	}
	defer respUsers.Body.Close()

	bodyUsers, err := ioutil.ReadAll(respUsers.Body)
	if err != nil {
		fmt.Printf("Error reading users response: %v\n", err)
		return
	}

	var usersResponse models.TwitchUsersResponse
	err = json.Unmarshal(bodyUsers, &usersResponse)
	if err != nil {
		fmt.Printf("Error parsing users JSON: %v\n", err)
		return
	}

	// Создаем мапы для быстрого доступа
	liveStreamsMap := make(map[string]models.TwitchStream)
	for _, stream := range streamResponse.Data {
		liveStreamsMap[stream.UserLogin] = stream
	}

	usersMap := make(map[string]models.TwitchUser)
	for _, user := range usersResponse.Data {
		usersMap[user.Login] = user
	}

	// Формируем список каналов
	for _, channelLogin := range channelsToCheck {
		if stream, exists := liveStreamsMap[channelLogin]; exists {
			// Канал в эфире
			if user, userExists := usersMap[channelLogin]; userExists {
				stream.ProfileImageURL = user.ProfileImageURL
			}
			stream.IsLive = true
			mutex.Lock()
			allChannels = append(allChannels, stream)
			mutex.Unlock()
		} else if user, exists := usersMap[channelLogin]; exists {
			// Канал не в эфире
			offlineChannel := models.TwitchStream{
				UserID:          user.ID,
				UserLogin:       user.Login,
				UserName:        user.DisplayName,
				ProfileImageURL: user.ProfileImageURL,
				IsLive:          false,
				ViewerCount:     0,
			}
			mutex.Lock()
			allChannels = append(allChannels, offlineChannel)
			mutex.Unlock()
		} else {
			fmt.Printf("Channel %s not found on Twitch\n", channelLogin)
		}
	}

	// Обновляем кэш
	cacheMutex.Lock()
	cachedLiveChannels = allChannels
	lastUpdateTime = time.Now()
	cacheMutex.Unlock()
}

func updateTopAuthorsCache() {
	go func() {
		authors, err := service.GetLeaderboard()
		if err != nil {
			log.Printf("Ошибка при обновлении кэша: %v", err)
			return
		}

		leaderboardMutex.Lock()
		defer leaderboardMutex.Unlock()
		cachedTopAuthors = authors
		leaderboardLastUpdated = time.Now()
	}()
}

func StartCacheUpdater() {
	ticker := time.NewTicker(cacheDuration)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			updateLiveChannelsCache()
		}
	}
}

func StartTopUpdater() {
	ticker := time.NewTicker(cacheDuration)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			updateTopAuthorsCache()
		}
	}
}

func AddLiveChannelHandler(w http.ResponseWriter, r *http.Request) {
	// TODO
}

func GetTopAuthorsHandler(w http.ResponseWriter, r *http.Request) {
	leaderboardMutex.RLock()
	defer leaderboardMutex.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(cachedTopAuthors)
}

func LoadMessagesHistoryHandler(w http.ResponseWriter, r *http.Request) {
	limit, err := strconv.Atoi(r.URL.Query().Get("limit"))
	if err != nil {
		http.Error(w, "Wrong request", http.StatusBadRequest)
		return
	}

	messages, err := service.LoadMessagesHistory(limit)
	if err != nil {
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}
	//log.Println("Messages: " + strconv.Itoa(len(messages)))

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(messages)
}

func SendMessageHandler(w http.ResponseWriter, r *http.Request) {
	// Ограничиваем размер запроса (50 MB для сообщений с файлами)
	r.ParseMultipartForm(50 << 20)

	// Получаем текст сообщения
	messageText := r.FormValue("message")

	// Проверяем авторизацию пользователя
	session, err := store.Get(r, sessionName)
	if err != nil {
		http.Error(w, "Session error", http.StatusInternalServerError)
		return
	}

	userID, ok := session.Values["user_id"]
	if !ok {
		http.Error(w, "Unauthorised", http.StatusUnauthorized)
		return
	}

	// Проверка на бан
	if service.IsBanned(userID.(int)) {
		http.Error(w, "Вы забанены и не можете отправлять сообщения в чат", http.StatusForbidden)
		return
	}

	user, err := service.GetUserByID(userID.(int))
	if err != nil {
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	// Обрабатываем прикрепленные файлы
	files := r.MultipartForm.File["files"]

	messageID, err := service.SaveMessage(user.ID, user.CurrentBadgeID, messageText)
	if err != nil {
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	for _, fileHeader := range files {
		file, err := fileHeader.Open()
		if err != nil {
			log.Printf("Ошибка открытия файла: %v", err)
			continue
		}
		defer file.Close()

		// Проверяем тип файла (только изображения и аудио)
		buff := make([]byte, 512)
		_, err = file.Read(buff)
		if err != nil {
			log.Printf("Ошибка чтения файла: %v", err)
			continue
		}

		fileType := http.DetectContentType(buff)
		if !strings.HasPrefix(fileType, "image/") && !strings.HasPrefix(fileType, "audio/") {
			log.Printf("Недопустимый тип файла: %s", fileType)
			continue
		}

		// Возвращаем указатель файла в начало
		file.Seek(0, 0)

		// Создаем директорию для загрузок чата, если её нет
		os.MkdirAll("./static/chat_uploads", os.ModePerm)

		// Генерируем уникальное имя файла
		newFileName := service.GenerateUniqueFileName(fileHeader.Filename)
		filePath := "./static/chat_uploads/" + newFileName

		// Сохраняем файл
		dst, err := os.Create(filePath)
		if err != nil {
			log.Printf("Ошибка создания файла: %v", err)
			continue
		}

		if _, err := io.Copy(dst, file); err != nil {
			log.Printf("Ошибка копирования файла: %v", err)
			dst.Close()
			continue
		}
		dst.Close()

		// Сохраняем информацию в БД
		err = service.ClipFile(messageID, "../static/chat_uploads/"+newFileName)

		if err != nil {
			log.Printf("Ошибка сохранения информации о файле: %v", err)
			continue
		}
	}

	w.WriteHeader(http.StatusOK)
}
