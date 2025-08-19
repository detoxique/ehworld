package app

import (
	"ehchobyahs/internal/handlers"
	"ehchobyahs/internal/service"
	"log"
	"net/http"
	"os"

	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
)

func Run() {
	// Загрузка из файла .env
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	service.InitDB()
	handlers.UpdateConfig()

	value := os.Getenv("PORT")

	// Настройка маршрутов
	r := mux.NewRouter()

	r.HandleFunc("/", handlers.ServeHomePage)
	r.HandleFunc("/notfound", handlers.ServeNotFoundPage)
	r.HandleFunc("/user/{username}", handlers.ServeUserPageHandler)
	r.HandleFunc("/auth/twitch", handlers.AuthTwitchHandler)
	r.HandleFunc("/auth/callback", handlers.AuthCallbackHandler)

	r.HandleFunc("/main", handlers.AuthMiddleware(handlers.ServeMainPage))
	r.HandleFunc("/feed", handlers.AuthMiddleware(handlers.ServeFeedPage))
	r.HandleFunc("/upload", handlers.AuthMiddleware(handlers.ServeUploadPage))
	r.HandleFunc("/shop", handlers.AuthMiddleware(handlers.ServeShopPage))
	r.HandleFunc("/settings", handlers.AuthMiddleware(handlers.ServeUploadPage))
	r.HandleFunc("/logout", handlers.AuthMiddleware(handlers.LogoutHandler))
	r.HandleFunc("/post/{id}", handlers.ServePostPage)
	r.HandleFunc("/inventory", handlers.AuthMiddleware(handlers.ServeInventoryPage))

	// Модераторские страницы
	r.HandleFunc("/moderator", handlers.ModeratorMiddleware(handlers.ServeModeratorPage))

	// Админские страницы
	r.HandleFunc("/admin", handlers.AdminMiddleware(handlers.ServeAdminPage))
	r.HandleFunc("/admin/twitch", handlers.AdminMiddleware(handlers.AdminAuthHandler))
	r.HandleFunc("/admin/callback", handlers.AdminMiddleware(handlers.AdminCallbackHandler))
	r.HandleFunc("/queue", handlers.AdminMiddleware(handlers.ServeQueuePage))

	// API Gateway
	r.HandleFunc("/api/upload", handlers.AuthMiddleware(handlers.UploadHandler))
	r.HandleFunc("/api/upload/clip", handlers.AuthMiddleware(handlers.UploadClipHandler))
	r.HandleFunc("/api/like/{id}", handlers.AuthMiddleware(handlers.LikeHandler)).Methods("POST", "DELETE")
	r.HandleFunc("/api/fuckyou/{id}", handlers.AuthMiddleware(handlers.FuckYouHandler)).Methods("POST", "DELETE")
	r.HandleFunc("/api/likecomment/{id}", handlers.AuthMiddleware(handlers.LikeCommentHandler)).Methods("POST", "DELETE")
	r.HandleFunc("/api/comment/{id}", handlers.AuthMiddleware(handlers.CommentHandler)).Methods("POST", "DELETE", "PUT")
	r.HandleFunc("/api/feed", handlers.AuthMiddleware(handlers.FeedHandler)).Methods("GET")
	r.HandleFunc("/api/last-files", handlers.AuthMiddleware(handlers.LastFilesHandler)).Methods("GET")
	r.HandleFunc("/api/search", handlers.SearchHandler)
	r.HandleFunc("/api/buy_item/{id}", handlers.AuthMiddleware(handlers.BuyItemHandler))
	r.HandleFunc("/api/notifications", handlers.AuthMiddleware(handlers.GetNotificationsHandler)).Methods("GET")
	r.HandleFunc("/api/posts/{id}", handlers.GetUserPostsHandler).Methods("GET")
	r.HandleFunc("/api/follow/{id}", handlers.AuthMiddleware(handlers.SubscribeHandler)).Methods("POST", "DELETE")
	r.HandleFunc("/api/case-rewards/{id}", handlers.AuthMiddleware(handlers.GetCaseRewardsHandler)).Methods("GET")
	r.HandleFunc("/api/case-open/{id}", handlers.AuthMiddleware(handlers.OpenCaseHandler)).Methods("POST")
	r.HandleFunc("/api/apply-badge/{id}", handlers.AuthMiddleware(handlers.ApplyBadgeHandler)).Methods("POST")
	r.HandleFunc("/api/apply-item/{id}", handlers.AuthMiddleware(handlers.ApplyItem)).Methods("POST")

	// Модераторские API
	r.HandleFunc("/api/moderation/posts/{page}", handlers.ModeratorMiddleware(handlers.GetModerationPostsHandler)).Methods("GET")
	r.HandleFunc("/api/moderation/approve/{id}", handlers.ModeratorMiddleware(handlers.ApprovePostHandler)).Methods("POST")
	r.HandleFunc("/api/moderation/reject/{id}", handlers.ModeratorMiddleware(handlers.RejectPostHandler)).Methods("POST")
	r.HandleFunc("/api/moderation/delete/{id}", handlers.ModeratorMiddleware(handlers.DeletePostHandler)).Methods("POST")
	r.HandleFunc("/api/moderation/ban/{id}", handlers.ModeratorMiddleware(handlers.BanUserHandler)).Methods("POST")
	r.HandleFunc("/api/moderation/banusername/{username}", handlers.ModeratorMiddleware(handlers.BanUsernameHandler)).Methods("POST", "DELETE")

	// Админские API
	r.HandleFunc("/api/admin/moderators", handlers.AdminMiddleware(handlers.GetModeratorsListHandler)).Methods("GET")
	r.HandleFunc("/api/admin/moderatorrole/{username}", handlers.AdminMiddleware(handlers.ModeratorRoleHandler)).Methods("POST", "DELETE")
	r.HandleFunc("/api/admin/users", handlers.AdminMiddleware(handlers.UsersSearchHandler))
	r.HandleFunc("/api/admin/banned", handlers.AdminMiddleware(handlers.GetBannedUsersListHandler)).Methods("GET")
	r.HandleFunc("/api/admin/uploadbadge", handlers.AdminMiddleware(handlers.UploadBadgeHandler)).Methods("POST")
	r.HandleFunc("/api/admin/add-case", handlers.AdminMiddleware(handlers.AddCaseHandler)).Methods("POST")
	r.HandleFunc("/api/admin/add-rewards", handlers.AdminMiddleware(handlers.AddRewardsHandler)).Methods("POST")
	r.HandleFunc("/api/admin/badges", handlers.AdminMiddleware(handlers.GetBadgesHandler))

	r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir("./static"))))

	// Запуск сервера
	log.Println("Запуск сервера. Порт :" + value)
	go log.Fatal(http.ListenAndServe(":"+value, r))
}
