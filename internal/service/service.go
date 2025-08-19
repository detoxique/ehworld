package service

import (
	"bytes"
	"database/sql"
	"ehchobyahs/internal/models"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"math"
	"math/rand/v2"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	_ "github.com/lib/pq"
)

var (
	db *sql.DB
)

func InitDB() {
	var err error
	db, err = sql.Open("postgres", os.Getenv("DB_URL"))
	if err != nil {
		log.Println(err.Error())
	}
}

func GetUserByID(id int) (*models.User, error) {
	var user models.User
	err := db.QueryRow(`
		SELECT id, twitch_id, login, display_name, profile_image_url, email, created_at, role, rating, is_banned, followers
		FROM users 
		WHERE id = $1
	`, id).Scan(
		&user.ID, &user.TwitchID, &user.Login, &user.DisplayName,
		&user.ProfileImageURL, &user.Email, &user.CreatedAt, &user.Role, &user.Rating, &user.IsBanned, &user.Followers,
	)

	if HasBadge(id) {
		err = db.QueryRow(`
			SELECT badges.image, badges.id
			FROM users, badges
			WHERE users.id = $1 AND users.badge_id = badges.id
		`, id).Scan(&user.Badge, &user.CurrentBadgeID)
		if err != nil {
			return nil, err
		}

	}

	return &user, err
}

func GetUserByUsername(username string) (*models.User, error) {
	var user models.User
	err := db.QueryRow(`
		SELECT id, twitch_id, login, display_name, profile_image_url, email, created_at, role, rating, is_banned, followers
		FROM users 
		WHERE login = $1
	`, strings.ToLower(username)).Scan(
		&user.ID, &user.TwitchID, &user.Login, &user.DisplayName,
		&user.ProfileImageURL, &user.Email, &user.CreatedAt, &user.Role, &user.Rating, &user.IsBanned, &user.Followers,
	)

	if HasBadge(user.ID) {
		err = db.QueryRow(`
			SELECT badges.image
			FROM users, badges
			WHERE users.id = $1 AND users.badge_id = badges.id
		`, user.ID).Scan(&user.Badge)
		if err != nil {
			return nil, err
		}
	}

	return &user, err
}

func CreateOrUpdateUser(twitchUser models.User) (*models.User, error) {
	var user models.User
	err := db.QueryRow(`
		SELECT id, twitch_id, login, display_name, profile_image_url, email, rating, is_banned 
		FROM users 
		WHERE twitch_id = $1
	`, twitchUser.TwitchID).Scan(
		&user.ID, &user.TwitchID, &user.Login, &user.DisplayName,
		&user.ProfileImageURL, &user.Email, &user.Rating, &user.IsBanned,
	)

	if err == sql.ErrNoRows {
		var id int
		err := db.QueryRow(`
			INSERT INTO users (
				twitch_id, login, display_name, profile_image_url, email
			) VALUES ($1, $2, $3, $4, $5) RETURNING id
		`, twitchUser.TwitchID, twitchUser.Login, twitchUser.DisplayName,
			twitchUser.ProfileImageURL, twitchUser.Email).Scan(&id)

		if err != nil {
			return nil, errors.New("failed to insert " + err.Error())
		}

		return GetUserByID(int(id))
	} else if err != nil {
		return nil, errors.New("failed to select")
	}

	// Обновляем данные пользователя при каждом входе
	_, err = db.Exec(`
		UPDATE users SET 
			login = $1,
			display_name = $2,
			profile_image_url = $3,
			email = $4
		WHERE id = $5
	`, twitchUser.Login, twitchUser.DisplayName,
		twitchUser.ProfileImageURL, twitchUser.Email, user.ID)

	if err != nil {
		return nil, errors.New("failed to update")
	}

	if user.IsBanned {
		return nil, errors.New("user is banned")
	}

	return GetUserByID(user.ID)
}

func CheckModeratorOrAdminRole(userID int) bool {
	user, err := GetUserByID(userID)
	if err != nil {
		return false
	}

	if user.Role == "admin" || user.Role == "moderator" {
		return true
	}
	return false
}

func CheckAdminRole(userID int) bool {
	user, err := GetUserByID(userID)
	if err != nil {
		return false
	}

	if user.Role == "admin" {
		return true
	}
	return false
}

func SaveFile(file *models.File) (int, error) {
	// var less_than_limit bool
	// err := db.QueryRow(`
	// 	SELECT SUM(file_size) + $1 < $2 FROM files WHERE user_id = $3
	// `, file.FileSize, 209_715_200, file.UserID).Scan(&less_than_limit)
	// if err != nil {
	// 	return -1, err
	// }

	// if !less_than_limit {
	// 	return -1, errors.New("storage size limit reached")
	// }

	var id int
	err := db.QueryRow(`
		INSERT INTO files (user_id, title, file_name, thumbnail, file_size, uploaded_at, is_public, description)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8) RETURNING id
	`, file.UserID, file.Title, file.FileName, file.Thumbnail, file.FileSize, time.Now(), false, file.Description).Scan(&id)
	return id, err
}

func SaveClip(file *models.File) (int, error) {
	var id int
	err := db.QueryRow(`
		INSERT INTO files (user_id, title, file_name, thumbnail, file_size, uploaded_at, is_public, type, description)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9) RETURNING id
	`, file.UserID, file.Title, file.FileName, file.Thumbnail, 0, time.Now(), false, "clip", file.Description).Scan(&id)
	return id, err
}

func GetFiles() ([]models.File, error) {
	rows, err := db.Query(`
        SELECT id, user_id, title, file_name, file_size, uploaded_at 
        FROM files 
        WHERE is_public = true
        ORDER BY uploaded_at DESC
    `)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var files []models.File
	for rows.Next() {
		var file models.File
		err := rows.Scan(
			&file.ID, &file.UserID, &file.Title,
			&file.FileName, &file.FileSize, &file.UploadedAt,
		)
		if err != nil {
			return nil, err
		}
		files = append(files, file)
	}
	return files, nil
}

func GetMostPopularWithAuthors() ([]models.FileWithAuthor, error) {
	rows, err := db.Query(`
		SELECT 
    f.id, 
    f.user_id, 
    f.title, 
    f.file_name, 
    f.thumbnail, 
    f.file_size, 
    f.uploaded_at, 
    f.views, 
    f.likes, 
    f.type, 
    u.display_name
FROM files f
JOIN users u ON u.id = f.user_id
WHERE 
    f.is_public = true
    AND f.uploaded_at >= NOW() - INTERVAL '7 days'
ORDER BY f.views DESC
LIMIT 10;
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var files []models.FileWithAuthor
	for rows.Next() {
		var file models.FileWithAuthor
		err := rows.Scan(
			&file.ID, &file.UserID, &file.Title,
			&file.FileName, &file.Thumbnail, &file.FileSize, &file.UploadedAt, &file.Views, &file.Likes, &file.Type,
			&file.AuthorName,
		)
		if err != nil {
			return nil, err
		}
		files = append(files, file)
	}
	return files, nil
}

func GetLastFilesWithAuthors() ([]models.FileWithAuthor, error) {
	rows, err := db.Query(`
        SELECT f.id, f.user_id, f.title, f.file_name, f.thumbnail, f.file_size, 
               f.uploaded_at, f.views, f.likes, f.type, u.display_name
        FROM files f
        JOIN users u ON u.id = f.user_id
        WHERE f.is_public = true
        ORDER BY f.uploaded_at DESC
		LIMIT 8
    `)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var files []models.FileWithAuthor
	for rows.Next() {
		var file models.FileWithAuthor
		err := rows.Scan(
			&file.ID, &file.UserID, &file.Title,
			&file.FileName, &file.Thumbnail, &file.FileSize, &file.UploadedAt, &file.Views, &file.Likes, &file.Type,
			&file.AuthorName,
		)
		if err != nil {
			return nil, err
		}
		files = append(files, file)
	}
	return files, nil
}

func GetLastSeenFilesWithAuthors(userID int) ([]models.FileWithAuthor, error) {
	rows, err := db.Query(`
		SELECT f.id, f.user_id, f.title, f.file_name, f.thumbnail, f.file_size,
			f.uploaded_at, f.views, f.likes, f.type, u.display_name
		FROM (
			SELECT DISTINCT ON (ls.post_id) 
				ls.post_id, 
				ls.id AS last_seen_id
			FROM last_seen ls
			WHERE ls.user_id = $1
			ORDER BY ls.post_id, ls.id DESC
		) AS unique_ls
		JOIN files f ON unique_ls.post_id = f.id
		JOIN users u ON u.id = f.user_id
		WHERE f.is_public = true
		ORDER BY unique_ls.last_seen_id DESC
		LIMIT 10;
    `, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var files []models.FileWithAuthor
	for rows.Next() {
		var file models.FileWithAuthor
		err := rows.Scan(
			&file.ID, &file.UserID, &file.Title,
			&file.FileName, &file.Thumbnail, &file.FileSize, &file.UploadedAt, &file.Views, &file.Likes, &file.Type,
			&file.AuthorName,
		)
		if err != nil {
			return nil, err
		}
		files = append(files, file)
	}
	return files, nil
}

func GetNewFollowings(userID int) ([]models.FileWithAuthor, error) {
	rows, err := db.Query(`
		SELECT 
			f.id,
			f.user_id,
			f.title,
			f.file_name,
			f.thumbnail,
			f.file_size,
			f.uploaded_at,
			f.views,
			f.likes,
			f.type,
			u.display_name
		FROM files f
		JOIN follows fl ON f.user_id = fl.target_id
		JOIN users u ON f.user_id = u.id
		WHERE fl.user_id = $1
		AND f.is_public = true
		AND f.id NOT IN (
			SELECT post_id
			FROM last_seen
			WHERE user_id = 123
		)
		ORDER BY f.uploaded_at DESC
		LIMIT 10;
    `, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var files []models.FileWithAuthor
	for rows.Next() {
		var file models.FileWithAuthor
		err := rows.Scan(
			&file.ID, &file.UserID, &file.Title,
			&file.FileName, &file.Thumbnail, &file.FileSize, &file.UploadedAt, &file.Views, &file.Likes, &file.Type,
			&file.AuthorName,
		)
		if err != nil {
			return nil, err
		}
		files = append(files, file)
	}
	return files, nil
}

func GenerateThumbnail(videoPath, thumbnailPath string) error {
	cmd := exec.Command(
		"ffmpeg",
		"-i", videoPath,
		"-ss", "00:00:01", // Время для снимка (1 секунда)
		"-vframes", "1", // Количество кадров
		"-vf", "scale=320:-1", // Масштабирование
		thumbnailPath,
	)
	return cmd.Run()
}

func IsVideoFile(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	return ext == ".mp4" || ext == ".mov" || ext == ".avi" || ext == ".mkv" || ext == ".webm"
}

func IsImageFile(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	return ext == ".png" || ext == ".gif" || ext == ".jpg" || ext == ".jpeg" || ext == ".bmp"
}

func GenerateUniqueFileName(original string) string {
	ext := filepath.Ext(original)
	base := strings.TrimSuffix(original, ext)
	// Заменяем пробелы и спецсимволы
	base = strings.ReplaceAll(base, " ", "_")
	base = strings.Map(func(r rune) rune {
		if r < 32 || r > 126 || r == ' ' || strings.ContainsRune("!*'();:@&=+$,/?%#[]", r) {
			return '_'
		}
		return r
	}, base)
	return fmt.Sprintf("%s_%d%s", base, time.Now().UnixNano(), ext)
}

// Факи
func FuckYouFile(userID, fileID int) error {
	if !IsBanned(userID) {
		_, err := db.Exec(`
		INSERT INTO fucks (user_id, file_id)
		VALUES ($1, $2)
		ON CONFLICT (user_id, file_id) DO NOTHING
	`, userID, fileID)
		if err != nil {
			return err
		}

		var author_id int
		err = db.QueryRow(`
		SELECT user_id FROM files
		WHERE id = $1
		`, fileID).Scan(&author_id)
		if err != nil {
			return err
		}

		// Отправить уведомление
		var count int
		err = db.QueryRow(`
		SELECT COUNT(*) FROM notifications
		WHERE author_id = $1 AND file_id = $2 AND user_id = $3 AND type = 'fuck'
		`, userID, fileID, author_id).Scan(&count)
		if err != nil {
			return err
		}

		// Уведомлений нет
		if count == 0 {
			var user_display_name, user_profile_image string
			err = db.QueryRow(`
		SELECT display_name, profile_image_url FROM users
		WHERE id = $1
		`, userID).Scan(&user_display_name, &user_profile_image)
			if err != nil {
				return err
			}

			_, err = db.Exec(`
		INSERT INTO notifications (user_id, author_id, notification, image, link, file_id, type)
		VALUES ($1, $2, $3, $4, $5, $6, 'fuck')
	`, author_id, userID, user_display_name+" послал вас нах под вашей публикацией!", user_profile_image, "https://ehworld.ru/post/"+strconv.Itoa(fileID), fileID)
			if err != nil {
				return err
			}
		}

		_, err = db.Exec(`
		UPDATE files 
		SET fucks = (SELECT COUNT(*) FROM fucks WHERE file_id = $1)
		WHERE id = $2
	`, fileID, fileID)
		return err
	} else {
		return errors.New("user is banned")
	}
}

func UnFuckYouFile(userID, fileID int) error {
	_, err := db.Exec(`
		DELETE FROM fucks 
		WHERE user_id = $1 AND file_id = $2
	`, userID, fileID)
	if err != nil {
		return err
	}

	_, err = db.Exec(`
		UPDATE files 
		SET fucks = (SELECT COUNT(*) FROM fucks WHERE file_id = $1)
		WHERE id = $2
	`, fileID, fileID)
	return err
}

// Лайки
func LikeFile(userID, fileID int) error {
	if !IsBanned(userID) {
		_, err := db.Exec(`
		INSERT INTO likes (user_id, file_id)
		VALUES ($1, $2)
		ON CONFLICT (user_id, file_id) DO NOTHING
	`, userID, fileID)
		if err != nil {
			return err
		}

		var file_author_id int
		err = db.QueryRow(`
		SELECT user_id FROM files
		WHERE id = $1
		`, fileID).Scan(&file_author_id)
		if err != nil {
			return err
		}

		// Не засчитывать собственные лайки
		if userID != file_author_id {
			err = IncrementRating(file_author_id, 1)
			if err != nil {
				return err
			}
		}

		var author_id int
		err = db.QueryRow(`
		SELECT user_id FROM files
		WHERE id = $1
		`, fileID).Scan(&author_id)
		if err != nil {
			return err
		}

		// Отправить уведомление
		var count int
		err = db.QueryRow(`
		SELECT COUNT(*) FROM notifications
		WHERE author_id = $1 AND file_id = $2 AND user_id = $3 AND type = 'like'
		`, userID, fileID, author_id).Scan(&count)
		if err != nil {
			return err
		}

		// Уведомлений нет
		if count == 0 {
			var user_display_name, user_profile_image string
			err = db.QueryRow(`
		SELECT display_name, profile_image_url FROM users
		WHERE id = $1
		`, userID).Scan(&user_display_name, &user_profile_image)
			if err != nil {
				return err
			}

			_, err = db.Exec(`
		INSERT INTO notifications (user_id, author_id, notification, image, link, file_id, type)
		VALUES ($1, $2, $3, $4, $5, $6, 'like')
	`, author_id, userID, user_display_name+" послал поставил лайк вашей публикации!", user_profile_image, "https://ehworld.ru/post/"+strconv.Itoa(fileID), fileID)
			if err != nil {
				return err
			}
		}

		_, err = db.Exec(`
		UPDATE files 
		SET likes = (SELECT COUNT(*) FROM likes WHERE file_id = $1)
		WHERE id = $2
	`, fileID, fileID)
		return err
	} else {
		return errors.New("user is banned")
	}
}

func LikeComment(userID, commentID int) error {
	if !IsBanned(userID) {
		_, err := db.Exec(`
		INSERT INTO comments_likes (user_id, comment_id)
		VALUES ($1, $2)
		ON CONFLICT (user_id, comment_id) DO NOTHING
	`, userID, commentID)
		if err != nil {
			return err
		}

		var author_id int
		err = db.QueryRow(`
		SELECT user_id FROM comments
		WHERE id = $1
		`, commentID).Scan(&author_id)
		if err != nil {
			return err
		}

		// Не засчитывать собственные лайки
		if userID != author_id {
			err = IncrementRating(author_id, 1)
			if err != nil {
				return err
			}
		}

		_, err = db.Exec(`
		UPDATE comments 
		SET likes = (SELECT COUNT(*) FROM comments_likes WHERE comment_id = $1)
		WHERE id = $2
	`, commentID, commentID)
		return err
	} else {
		return errors.New("user is banned")
	}
}

func UnlikeFile(userID, fileID int) error {
	_, err := db.Exec(`
		DELETE FROM likes 
		WHERE user_id = $1 AND file_id = $2
	`, userID, fileID)
	if err != nil {
		return err
	}

	IncrementRating(userID, -1)

	_, err = db.Exec(`
		UPDATE files 
		SET likes = (SELECT COUNT(*) FROM likes WHERE file_id = $1)
		WHERE id = $2
	`, fileID, fileID)
	return err
}

func UnlikeComment(userID, commentID int) error {
	_, err := db.Exec(`
		DELETE FROM comments_likes 
		WHERE user_id = $1 AND comment_id = $2
	`, userID, commentID)
	if err != nil {
		return err
	}

	IncrementRating(userID, -1)

	_, err = db.Exec(`
		UPDATE comments 
		SET likes = (SELECT COUNT(*) FROM comments_likes WHERE comment_id = $1)
		WHERE id = $2
	`, commentID, commentID)
	return err
}

func HasLiked(userID, fileID int) (bool, error) {
	var exists bool
	err := db.QueryRow(`
		SELECT EXISTS(
			SELECT 1 FROM likes 
			WHERE user_id = $1 AND file_id = $2
		)
	`, userID, fileID).Scan(&exists)
	return exists, err
}

func HasLikedComment(userID, commentID int) (bool, error) {
	var exists bool
	err := db.QueryRow(`
		SELECT EXISTS(
			SELECT 1 FROM comments_likes 
			WHERE user_id = $1 AND comment_id = $2
		)
	`, userID, commentID).Scan(&exists)
	return exists, err
}

func HasFuckYou(userID, fileID int) (bool, error) {
	var exists bool
	err := db.QueryRow(`
		SELECT EXISTS(
			SELECT 1 FROM fucks 
			WHERE user_id = $1 AND file_id = $2
		)
	`, userID, fileID).Scan(&exists)
	return exists, err
}

// Комментарии
func AddComment(comment *models.Comment) (int, error) {
	filteredText, err := FilterBadWords(comment.Text)
	if err != nil {
		return -1, err
	}

	var existingComm int
	err = db.QueryRow(`SELECT id FROM comments WHERE user_id = $1 AND text = $2 AND file_id = $3`, comment.UserID, comment.Text, comment.FileID).Scan(&existingComm)
	if err == sql.ErrNoRows {
		var id int
		if comment.ParentID == 0 {
			err = db.QueryRow(`
		INSERT INTO comments (user_id, file_id, text)
		VALUES ($1, $2, $3) RETURNING id
	`, comment.UserID, comment.FileID, filteredText).Scan(&id)
			return id, err
		} else {
			err = db.QueryRow(`
		INSERT INTO comments (user_id, file_id, text, parent_id)
		VALUES ($1, $2, $3, $4) RETURNING id
	`, comment.UserID, comment.FileID, filteredText, comment.ParentID).Scan(&id)
			return id, err
		}

	} else {
		return -1, errors.New("comment already exists")
	}
}

func UpdateComment(comment *models.Comment) error {
	filteredText, err := FilterBadWords(comment.Text)
	if err != nil {
		return err
	}

	var commentID int
	err = db.QueryRow(`SELECT id FROM comments WHERE user_id = $1 AND file_id = $2`, comment.UserID, comment.FileID).Scan(&commentID)
	if err != nil {
		return err
	}

	_, err = db.Exec(`
		UPDATE comments 
		SET text = $1
		WHERE id = $2
	`, filteredText, commentID)
	if err != nil {
		return err
	}

	return nil
}

func DeleteComment(comment *models.Comment) error {
	_, err := db.Exec(`
		DELETE FROM comments_likes  
		WHERE comment_id = $1
	`, comment.ID)
	if err != nil {
		return err
	}
	_, err = db.Exec(`
		DELETE FROM comments
		WHERE id = $1
	`, comment.ID)
	if err != nil {
		return err
	}
	return nil
}

func GetComments(userID, fileID int) ([]models.CommentWithAuthor, error) {
	// Основные комментарии (без родителя)
	rows, err := db.Query(`
        SELECT c.id, c.user_id, c.text, c.created_at, c.likes, 
               u.display_name, u.profile_image_url, u.id
        FROM comments c
        JOIN users u ON u.id = c.user_id
        WHERE c.file_id = $1 AND c.parent_id IS NULL
        ORDER BY c.created_at DESC
    `, fileID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var comments []models.CommentWithAuthor
	// Обработка и добавление ответов
	for rows.Next() {
		var c models.CommentWithAuthor
		// Сканирование данных
		err := rows.Scan(&c.ID, &c.UserID, &c.Text, &c.CreatedAt, &c.Likes, &c.AuthorName, &c.AuthorProfileImageURL, &c.AuthorID)
		if err != nil {
			return nil, err
		}

		// Проверяем на лайк
		c.HasLiked, _ = HasLikedComment(userID, c.ID)

		// Загружаем значок если есть
		if HasBadge(c.AuthorID) {
			err = db.QueryRow(`
			SELECT badges.image
			FROM users, badges
			WHERE users.id = $1 AND users.badge_id = badges.id
		`, c.AuthorID).Scan(&c.Badge)
			if err != nil {
				return nil, err
			}
		}

		// Получение ответов
		replies, _ := getReplies(userID, c.Comment.ID)
		c.Replies = replies
		comments = append(comments, c)
	}
	return comments, nil
}

func GetCommentsUnauthorised(fileID int) ([]models.CommentWithAuthor, error) {
	// Основные комментарии (без родителя)
	rows, err := db.Query(`
        SELECT c.id, c.user_id, c.text, c.created_at, c.likes, 
               u.display_name, u.profile_image_url, u.id
        FROM comments c
        JOIN users u ON u.id = c.user_id
        WHERE c.file_id = $1 AND c.parent_id IS NULL
        ORDER BY c.created_at DESC
    `, fileID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var comments []models.CommentWithAuthor
	// Обработка и добавление ответов
	for rows.Next() {
		var c models.CommentWithAuthor
		// Сканирование данных
		err := rows.Scan(&c.ID, &c.UserID, &c.Text, &c.CreatedAt, &c.Likes, &c.AuthorName, &c.AuthorProfileImageURL, &c.AuthorID)
		if err != nil {
			return nil, err
		}

		// Загружаем значок если есть
		if HasBadge(c.AuthorID) {
			err = db.QueryRow(`
			SELECT badges.image
			FROM users, badges
			WHERE users.id = $1 AND users.badge_id = badges.id
		`, c.AuthorID).Scan(&c.Badge)
			if err != nil {
				return nil, err
			}
		}

		// Получение ответов
		replies, _ := getRepliesUnauthorised(c.Comment.ID)
		c.Replies = replies
		comments = append(comments, c)
	}
	return comments, nil
}

func getReplies(userID, parentID int) ([]models.CommentWithAuthor, error) {
	rows, err := db.Query(`
		SELECT c.id, c.user_id, c.file_id, c.text, c.created_at, c.likes, u.display_name, u.profile_image_url, u.id 
		FROM comments c
		JOIN users u ON u.id = c.user_id
		WHERE c.file_id = $1 AND c.is_deleted = FALSE
		ORDER BY c.created_at DESC
	`, parentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var comments []models.CommentWithAuthor
	for rows.Next() {
		var c models.CommentWithAuthor
		err := rows.Scan(&c.ID, &c.UserID, &c.FileID, &c.Text, &c.CreatedAt, &c.Likes, &c.AuthorName, &c.AuthorProfileImageURL, &c.AuthorID)
		if err != nil {
			return nil, err
		}

		// Проверяем на лайк
		c.HasLiked, _ = HasLikedComment(userID, c.ID)

		// Загружаем значок если есть
		if HasBadge(c.AuthorID) {
			err = db.QueryRow(`
			SELECT badges.image
			FROM users, badges
			WHERE users.id = $1 AND users.badge_id = badges.id
		`, c.AuthorID).Scan(&c.Badge)
			if err != nil {
				return nil, err
			}
		}

		comments = append(comments, c)
	}
	return comments, nil
}

func getRepliesUnauthorised(parentID int) ([]models.CommentWithAuthor, error) {
	rows, err := db.Query(`
		SELECT c.id, c.user_id, c.file_id, c.text, c.created_at, c.likes, u.display_name, u.profile_image_url, u.id 
		FROM comments c
		JOIN users u ON u.id = c.user_id
		WHERE c.file_id = $1 AND c.is_deleted = FALSE
		ORDER BY c.created_at DESC
	`, parentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var comments []models.CommentWithAuthor
	for rows.Next() {
		var c models.CommentWithAuthor
		err := rows.Scan(&c.ID, &c.UserID, &c.FileID, &c.Text, &c.CreatedAt, &c.Likes, &c.AuthorName, &c.AuthorProfileImageURL, &c.AuthorID)
		if err != nil {
			return nil, err
		}

		// Загружаем значок если есть
		if HasBadge(c.AuthorID) {
			err = db.QueryRow(`
			SELECT badges.image
			FROM users, badges
			WHERE users.id = $1 AND users.badge_id = badges.id
		`, c.AuthorID).Scan(&c.Badge)
			if err != nil {
				return nil, err
			}
		}

		comments = append(comments, c)
	}
	return comments, nil
}

func IsCommentOwner(comment *models.Comment) bool {
	if CheckModeratorOrAdminRole(comment.UserID) {
		return true
	}
	var ownerID int
	err := db.QueryRow("SELECT user_id FROM comments WHERE id = $1", comment.ID).Scan(&ownerID)
	if err == sql.ErrNoRows {
		log.Println("no rows")
		return false
	}
	return err == nil && ownerID == comment.UserID
}

// Фильтрация
func FilterBadWords(text string) (string, error) {
	content, err := os.ReadFile("badwords.txt")
	if err != nil {
		return text, err
	}

	badWords := strings.Split(string(content), "\n")
	for _, word := range badWords {
		if word == "" {
			continue
		}
		re := regexp.MustCompile(`(?i)\b` + regexp.QuoteMeta(word) + `\b`)
		text = re.ReplaceAllString(text, "***")
	}
	return text, nil
}

// Получение файла
func GetFileByID(id int) (*models.FileWithAuthor, error) {
	var file models.FileWithAuthor
	err := db.QueryRow(`
		SELECT f.id, f.user_id, f.title, f.file_name, f.thumbnail, 
			f.views, f.likes, u.display_name, u.profile_image_url, f.uploaded_at, f.is_moderated, f.type 
		FROM files f
		JOIN users u ON u.id = f.user_id
		WHERE f.id = $1
	`, id).Scan(
		&file.ID, &file.UserID, &file.Title, &file.FileName,
		&file.Thumbnail, &file.Views, &file.Likes, &file.AuthorName, &file.AuthorProfileImageURL, &file.UploadedAt, &file.IsModerated, &file.Type,
	)
	return &file, err
}

func GetFileByIDAuthorised(id, userID int) (*models.FileWithAuthor, error) {
	var file models.FileWithAuthor
	err := db.QueryRow(`
		SELECT f.id, f.user_id, f.title, f.file_name, f.thumbnail, 
			f.views, f.likes, u.display_name, u.profile_image_url, f.uploaded_at, f.is_moderated, f.type, f.description, f.fucks, u.id 
		FROM files f
		JOIN users u ON u.id = f.user_id
		WHERE f.id = $1
	`, id).Scan(
		&file.ID, &file.UserID, &file.Title, &file.FileName,
		&file.Thumbnail, &file.Views, &file.Likes, &file.AuthorName, &file.AuthorProfileImageURL, &file.UploadedAt, &file.IsModerated, &file.Type, &file.Description, &file.Fucks, &file.AuthorID,
	)
	if err != nil {
		return nil, errors.New("post doesn't exist")
	}

	if HasBadge(file.AuthorID) {
		err = db.QueryRow(`
			SELECT badges.image
			FROM users, badges
			WHERE users.id = $1 AND users.badge_id = badges.id
		`, file.AuthorID).Scan(&file.Badge)
		if err != nil {
			return nil, err
		}
	}

	_, err = db.Exec(`
		INSERT INTO last_seen (user_id, post_id)
		VALUES ($1, $2)
	`, userID, id)
	if err != nil {
		return nil, err
	}

	return &file, err
}

func IncrementViews(fileID int) error {
	_, err := db.Exec(`
		UPDATE files SET views = views + 1 WHERE id = $1
	`, fileID)
	return err
}

// Форматирование времени
func FormatTimeAgo(t time.Time) string {
	now := time.Now()
	diff := now.Sub(t)

	switch {
	case diff < time.Minute:
		return "только что"
	case diff < time.Hour:
		return fmt.Sprintf("%d мин назад", int(diff.Minutes()))
	case diff < 24*time.Hour:
		return fmt.Sprintf("%d ч назад", int(diff.Hours()))
	case diff < 168*time.Hour:
		return fmt.Sprintf("%d д назад", int(diff.Hours()/24))
	default:
		return t.Format("02.01.2006")
	}
}

func FormatViews(num int64) string {
	lastTwo := num % 100
	lastOne := num % 10

	if lastTwo >= 11 && lastTwo <= 19 {
		return FormatValue(num) + " просомтров"
	}

	switch lastOne {
	case 1:
		return FormatValue(num) + " просмотр"
	case 2, 3, 4:
		return FormatValue(num) + " просмотра"
	default:
		return FormatValue(num) + " просмотров"
	}
}

func FormatValue(count int64) string {
	switch {
	case count < 1000:
		return strconv.FormatInt(count, 10)
	case count < 1_000_000:
		value := float64(count) / 1000.0
		rounded := math.Round(value*10) / 10
		if rounded >= 1000 {
			valueM := float64(count) / 1_000_000.0
			return formatFloat(valueM) + "M"
		}
		return formatFloat(value) + "K"
	}
	value := float64(count) / 1_000_000.0
	return formatFloat(value) + "M"
}

func formatFloat(value float64) string {
	s := fmt.Sprintf("%.1f", value)
	parts := strings.Split(s, ".")
	if len(parts) == 2 {
		if parts[1] == "0" {
			return parts[0]
		}
		return parts[0] + "," + parts[1]
	}
	return s
}

func FormatFollowers(num int) string {
	lastTwo := num % 100
	lastOne := num % 10

	if lastTwo >= 11 && lastTwo <= 19 {
		return strconv.Itoa(num) + " подписчиков"
	}

	switch lastOne {
	case 1:
		return strconv.Itoa(num) + " подписчик"
	case 2, 3, 4:
		return strconv.Itoa(num) + " подписчика"
	default:
		return strconv.Itoa(num) + " подписчиков"
	}
}

func IncrementRating(userID int, value int) error {
	_, err := db.Exec("UPDATE users SET rating = rating + $1 WHERE id = $2", value, userID)
	if err != nil {
		return err
	}

	return nil
}

// Лента
func GetFeedPosts(userID, offset, limit int) ([]models.FeedFile, error) {
	query := `
        SELECT 
			f.id, f.user_id, f.title, f.file_name, f.thumbnail, f.file_size, 
			f.uploaded_at, f.views, f.likes, f.fucks, f.type, f.description, u.display_name, u.profile_image_url, u.id
		FROM files f
		JOIN users u ON u.id = f.user_id
		WHERE f.is_public = true
		ORDER BY 
			CASE 
				WHEN EXISTS (
					SELECT 1 
					FROM last_seen 
					WHERE user_id = $1 AND post_id = f.id
				) THEN 1 
				ELSE 0 
			END, -- 0 = непросмотренные, 1 = просмотренные
			(f.views * 0.7 + f.likes * 0.3) DESC
		LIMIT $2 OFFSET $3;
    `
	rows, err := db.Query(query, userID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var posts []models.FeedFile
	for rows.Next() {
		var post models.FeedFile
		err := rows.Scan(
			&post.ID, &post.UserID, &post.Title, &post.FileName,
			&post.Thumbnail, &post.FileSize, &post.UploadedAt,
			&post.Views, &post.Likes, &post.Fucks, &post.Type, &post.Description, &post.AuthorName, &post.AuthorProfileImageURL, &post.AuthorID,
		)
		if err != nil {
			return nil, err
		}

		// Загружаем значок если есть
		if HasBadge(post.AuthorID) {
			err = db.QueryRow(`
			SELECT badges.image
			FROM users, badges
			WHERE users.id = $1 AND users.badge_id = badges.id
		`, post.AuthorID).Scan(&post.Badge)
			if err != nil {
				return nil, err
			}
		}

		// Считаем комментарии
		var commentsCount int
		err = db.QueryRow(`
			SELECT COUNT(*)
			FROM comments
			WHERE file_id = $1
		`, post.ID).Scan(&commentsCount)
		if err != nil {
			return nil, err
		}

		MarkPostSeen(userID, post.ID)
		IncrementViews(post.ID)

		post.IsLiked, _ = HasLiked(userID, post.ID)
		post.IsFucked, _ = HasFuckYou(userID, post.ID)
		post.FormatTime = FormatTimeAgo(post.UploadedAt)
		post.FormatViews = FormatViews(post.Views)
		post.Comments = commentsCount
		posts = append(posts, post)
	}
	return posts, nil
}

func MarkPostSeen(userID, postID int) error {
	_, err := db.Exec(`
        INSERT INTO last_seen (user_id, post_id) 
        VALUES ($1, $2) 
        ON CONFLICT (user_id, post_id) DO NOTHING
    `, userID, postID)
	return err
}

func Search(searchTerm string) []models.SearchResult {
	results := []models.SearchResult{}

	// Поиск постов
	posts, err := db.Query(`
			SELECT f.id, f.title, u.display_name, u.login
			FROM files f
			JOIN users u ON f.user_id = u.id
			WHERE f.is_public = true 
			AND (LOWER(f.title) LIKE $1 OR LOWER(u.display_name) LIKE $1)
			LIMIT 5
		`, searchTerm)
	if err == nil {
		defer posts.Close()
		for posts.Next() {
			var p models.SearchResult
			var username string
			if err := posts.Scan(&p.ID, &p.Title, &p.DisplayName, &username); err == nil {
				p.Type = "post"
				p.Username = username
				results = append(results, p)
			}
		}
	}

	// Поиск пользователей
	users, err := db.Query(`
			SELECT id, display_name, login
			FROM users
			WHERE LOWER(display_name) LIKE $1 OR LOWER(login) LIKE $1
			LIMIT 3
		`, searchTerm)
	if err == nil {
		defer users.Close()
		for users.Next() {
			var u models.SearchResult
			if err := users.Scan(&u.ID, &u.DisplayName, &u.Username); err == nil {
				u.Type = "user"
				results = append(results, u)
			}
		}
	}

	return results
}

func SearchUsers(searchTerm string) []models.UserSearchResult {
	results := []models.UserSearchResult{}

	// Поиск пользователей
	users, err := db.Query(`
			SELECT id, display_name, profile_image_url 
			FROM users
			WHERE LOWER(display_name) LIKE $1 OR LOWER(login) LIKE $1
			LIMIT 3
		`, searchTerm)
	if err == nil {
		defer users.Close()
		for users.Next() {
			var u models.UserSearchResult
			if err := users.Scan(&u.ID, &u.DisplayName, &u.ProfileImageURL); err == nil {
				results = append(results, u)
			}
		}
	}

	return results
}

func GetModeratorsList() []models.UserSearchResult {
	results := []models.UserSearchResult{}

	// Поиск пользователей
	users, err := db.Query(`
			SELECT id, display_name, profile_image_url 
			FROM users
			WHERE role = 'moderator'
			LIMIT 10
		`)
	if err == nil {
		defer users.Close()
		for users.Next() {
			var u models.UserSearchResult
			if err := users.Scan(&u.ID, &u.DisplayName, &u.ProfileImageURL); err == nil {
				results = append(results, u)
			}
		}
	}

	return results
}

func GetModerationPosts(limit, offset int) (models.ModerationResponse, error) {
	rows, err := db.Query(`
        SELECT f.id, f.user_id, u.display_name, u.profile_image_url, f.uploaded_at, f.file_name, f.title, f.type, f.description
        FROM files f
        JOIN users u ON f.user_id = u.id
        WHERE f.is_moderated = false
        ORDER BY f.uploaded_at ASC
        LIMIT $1 OFFSET $2
    `, 10, 0)
	if err != nil {
		var resp models.ModerationResponse
		return resp, err
	}
	defer rows.Close()

	var posts []models.ModerationPost
	count := 0

	for rows.Next() {
		var post models.ModerationPost
		err := rows.Scan(
			&post.ID,
			&post.AuthorID,
			&post.AuthorName,
			&post.AuthorAvatar,
			&post.UploadedAt,
			&post.FileName,
			&post.Title,
			&post.Type,
			&post.Description,
		)

		if err != nil {
			continue
		}

		// Ограничиваем количество результатов limit+1 для определения has_more
		if count < limit {
			posts = append(posts, post)
		}
		count++
	}

	hasMore := count > limit

	var resp models.ModerationResponse
	resp.Posts = posts
	resp.HasMore = hasMore

	return resp, nil
}

func GetLastPosts(limit, offset int) ([]models.MainFile, error) {
	rows, err := db.Query(`
        SELECT f.id, f.title, f.thumbnail, 
               f.views, f.type, f.file_name, u.display_name
        FROM files f
        JOIN users u ON u.id = f.user_id
        WHERE f.is_public = true
        ORDER BY f.uploaded_at DESC
		LIMIT $1 OFFSET $2
    `, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var files []models.MainFile
	for rows.Next() {
		var file models.MainFile
		err := rows.Scan(
			&file.ID, &file.Title, &file.Thumbnail, &file.Views, &file.Type, &file.FileName,
			&file.AuthorName,
		)
		if err != nil {
			return nil, err
		}
		views_s, _ := strconv.Atoi(file.Views)
		file.Views = FormatViews(int64(views_s))
		file.IsVideo = IsVideoFile(file.FileName)
		files = append(files, file)
	}
	return files, nil
}

func GetNotifications(userID int) ([]models.Notification, error) {
	rows, err := db.Query(`
        SELECT id, notification, image, link, created_at
		FROM notifications
		WHERE mark_seen = false AND user_id = $1
		ORDER BY created_at DESC
    `, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var notifications []models.Notification
	for rows.Next() {
		var n models.Notification
		err := rows.Scan(
			&n.ID, &n.Notification, &n.Image, &n.Link, &n.CreatedAt,
		)
		if err != nil {
			return nil, err
		}

		_, err = db.Exec("UPDATE notifications SET mark_seen = true WHERE id = $1", n.ID)
		if err != nil {
			return nil, err
		}

		n.FormattedTime = FormatTimeAgo(n.CreatedAt)
		notifications = append(notifications, n)
	}
	return notifications, nil
}

func LogModAction(userID int, action string) error {
	_, err := db.Exec("INSERT INTO mod_logs (user_id, action) VALUES ($1, $2)", userID, action)
	if err != nil {
		return err
	}

	return nil
}

func ApprovePost(userID, postID int) error {
	_, err := db.Exec("UPDATE files SET is_public = true WHERE id = $1", postID)
	if err != nil {
		return err
	}

	_, err = db.Exec("UPDATE files SET is_moderated = true WHERE id = $1", postID)
	if err != nil {
		return err
	}

	var user_id int
	err = db.QueryRow("SELECT user_id FROM files WHERE id = $1", postID).Scan(&user_id)
	if err != nil {
		return err
	}

	_, err = db.Exec("INSERT INTO notifications (user_id, notification, link, file_id, type, image) VALUES ($1, $2, $3, $4, $5, $6)", user_id, "Модераторы одобрили ваш пост!", "https://ehworld.ru/post/"+strconv.Itoa(postID), postID, "approved", "https://ehworld.ru/static/img/approved.svg")
	if err != nil {
		return err
	}

	LogModAction(userID, "Approved post "+strconv.Itoa(postID))

	return nil
}

func RejectPost(userID, postID int) error {
	_, err := db.Exec("UPDATE files SET is_moderated = true WHERE id = $1", postID)
	if err != nil {
		return err
	}

	var user_id int
	err = db.QueryRow("SELECT user_id FROM files WHERE id = $1", postID).Scan(&user_id)
	if err != nil {
		return err
	}

	_, err = db.Exec("INSERT INTO notifications (user_id, notification, link, file_id, type, image) VALUES ($1, $2, $3, $4, $5, $6)", user_id, "Модераторы отклонили ваш пост, но он все еще доступен по прямой ссылке", "https://ehworld.ru/post/"+strconv.Itoa(postID), postID, "rejected", "https://ehworld.ru/static/img/rejected.svg")
	if err != nil {
		return err
	}

	LogModAction(userID, "Rejected post "+strconv.Itoa(postID))

	return nil
}

func DeletePost(userID, postID int) error {
	// Получаем имя файла для удаления
	var fileName, thumb string
	err := db.QueryRow("SELECT file_name, thumbnail FROM files WHERE id = $1", postID).Scan(&fileName, &thumb)
	if err != nil {
		return err
	}

	// Удаляем запись из БД
	_, err = db.Exec("DELETE FROM last_seen WHERE post_id = $1", postID)
	if err != nil {
		return err
	}

	_, err = db.Exec("DELETE FROM comments WHERE file_id = $1", postID)
	if err != nil {
		return err
	}

	_, err = db.Exec("DELETE FROM files WHERE id = $1", postID)
	if err != nil {
		return err
	}

	// Удаляем физический файл
	go func() {
		filePath := filepath.Join("static/uploads", fileName)
		os.Remove(filePath)

		// Если есть миниатюра, удаляем и её
		thumbPath := filepath.Join("static/uploads", thumb)
		os.Remove(thumbPath)
	}()

	LogModAction(userID, "Deleted post "+strconv.Itoa(postID))

	return nil
}

func BanUser(modID, userID int) error {
	_, err := db.Exec("DELETE FROM comments WHERE user_id = $1", userID)
	if err != nil {
		return err
	}

	posts, _, err := GetUserPosts(userID, 1, 100, "all", "newest", "")
	if err == nil {
		for _, element := range posts {
			DeletePost(modID, element.ID)
		}
	}

	_, err = db.Exec("UPDATE users SET is_banned = true WHERE id = $1", userID)
	if err != nil {
		return err
	}

	LogModAction(modID, "Banned user "+strconv.Itoa(userID))

	return nil
}

func UnbanUser(modID, userID int) error {
	_, err := db.Exec("UPDATE users SET is_banned = false WHERE id = $1", userID)
	if err != nil {
		return err
	}

	LogModAction(modID, "Unbanned user "+strconv.Itoa(userID))

	return nil
}

func IsBanned(userID int) bool {
	var is_banned bool
	err := db.QueryRow("SELECT is_banned FROM users WHERE id = $1", userID).Scan(&is_banned)
	if err != nil {
		return true
	}

	return is_banned
}

func AddModerator(username string) error {
	var user_id int
	var role string
	err := db.QueryRow("SELECT id, role FROM users WHERE login = $1", strings.ToLower(username)).Scan(&user_id, &role)
	if err != nil {
		return err
	}

	if role != "admin" {
		_, err = db.Exec("UPDATE users SET role = 'moderator' WHERE id = $1", user_id)
		if err != nil {
			return err
		}

		return nil
	} else {
		return errors.New("user's already admin")
	}
}

func DeleteModerator(username string) error {
	var user_id int
	err := db.QueryRow("SELECT id FROM users WHERE login = $1", strings.ToLower(username)).Scan(&user_id)
	if err != nil {
		return err
	}

	_, err = db.Exec("UPDATE users SET role = 'user' WHERE id = $1", user_id)
	if err != nil {
		return err
	}

	return nil
}

func GetBadges() []models.Badge {
	results := []models.Badge{}

	// Поиск баджей
	users, err := db.Query(`
			SELECT badges.id, badges.image, shop_items.title, shop_items.cost 
			FROM badges, shop_items
			WHERE shop_items.badge_id = badges.id
		`)
	if err == nil {
		defer users.Close()
		for users.Next() {
			var b models.Badge
			if err := users.Scan(&b.ID, &b.Image, &b.Title, &b.Cost); err == nil {
				results = append(results, b)
			}
		}
	} else {
		log.Println(err.Error())
	}

	return results
}

func GetStats() (models.Stats, error) {
	var usersCount, views, posts, postlikes, commentlikes int64
	err := db.QueryRow("SELECT COUNT(*) FROM users").Scan(&usersCount)
	if err != nil {
		var stat models.Stats
		return stat, err
	}

	err = db.QueryRow("SELECT SUM(views) FROM files").Scan(&views)
	if err != nil {
		var stat models.Stats
		return stat, err
	}

	err = db.QueryRow("SELECT COUNT(*) FROM files").Scan(&posts)
	if err != nil {
		var stat models.Stats
		return stat, err
	}

	err = db.QueryRow("SELECT COUNT(*) FROM likes").Scan(&postlikes)
	if err != nil {
		var stat models.Stats
		return stat, err
	}

	err = db.QueryRow("SELECT COUNT(*) FROM comments_likes").Scan(&commentlikes)
	if err != nil {
		var stat models.Stats
		return stat, err
	}

	var result models.Stats
	result.TotalUsers = usersCount
	result.TotalViews = views
	result.TotalPosts = posts
	result.TotalLikes = postlikes + commentlikes
	return result, nil
}

func HasNotifications(userID int) bool {
	var nots int
	err := db.QueryRow("SELECT COUNT(*) FROM notifications WHERE id = $1", userID).Scan(&nots)
	if err != nil {
		return false
	}

	if nots > 0 {
		return true
	} else {
		return false
	}
}

func HasDescription(postID int) bool {
	var desc string
	err := db.QueryRow("SELECT description FROM files WHERE id = $1", postID).Scan(&desc)
	if err != nil {
		return false
	}

	if desc != "" {
		return true
	} else {
		return false
	}
}

func GetBannedUsersList() ([]models.UserSearchResult, error) {
	results := []models.UserSearchResult{}

	// Поиск пользователей
	users, err := db.Query(`
			SELECT id, display_name, profile_image_url 
			FROM users
			WHERE is_banned = true
		`)
	if err == nil {
		defer users.Close()
		for users.Next() {
			var u models.UserSearchResult
			if err := users.Scan(&u.ID, &u.DisplayName, &u.ProfileImageURL); err == nil {
				results = append(results, u)
			}
		}
	} else {
		return nil, err
	}

	return results, nil
}

// TWITCH API

// возвращает Twitch ID пользователя по его логину
func getUserIdByLogin(login string) (string, error) {
	token, err := RefreshTwitchToken()
	if err != nil {
		return "", err
	}
	clientId := os.Getenv("TWITCH_CLIENT_ID_BOT")
	url := fmt.Sprintf("https://api.twitch.tv/helix/users?login=%s", url.QueryEscape(login))

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Client-Id", clientId)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API request failed: %s", resp.Status)
	}

	var result struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	if len(result.Data) == 0 {
		return "", errors.New("user not found")
	}

	return result.Data[0].ID, nil
}

func RefreshTwitchToken() (string, error) {
	tokens := GetTokens()
	// Формируем данные запроса
	data := url.Values{}
	data.Set("grant_type", "refresh_token")
	data.Set("refresh_token", tokens.RefreshToken)
	data.Set("client_id", os.Getenv("TWITCH_CLIENT_ID_BOT"))
	data.Set("client_secret", os.Getenv("TWITCH_CLIENT_SECRET_BOT"))

	// Создаем POST-запрос
	req, err := http.NewRequest(
		"POST",
		"https://id.twitch.tv/oauth2/token",
		bytes.NewBufferString(data.Encode()),
	)
	if err != nil {
		return "", fmt.Errorf("error creating request: %w", err)
	}

	// Устанавливаем заголовок
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	// Выполняем запрос
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("error sending request: %w", err)
	}
	defer resp.Body.Close()

	// Читаем ответ
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("error reading response: %w", err)
	}

	// Проверяем статус код
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API error: %s, body: %s", resp.Status, body)
	}

	// Парсим ответ
	var tokenResponse struct {
		AccessToken  string   `json:"access_token"`
		RefreshToken string   `json:"refresh_token"`
		Scopes       []string `json:"scope"`
		TokenType    string   `json:"token_type"`
	}

	if err := json.Unmarshal(body, &tokenResponse); err != nil {
		return "", fmt.Errorf("error parsing JSON: %w", err)
	}

	// Обновляем переменные окружения
	os.Setenv("TWITCH_BOT_TOKEN", tokenResponse.AccessToken)
	os.Setenv("TWITCH_BOT_REFRESH_TOKEN", tokenResponse.RefreshToken)

	SaveTokens(tokenResponse.AccessToken, tokenResponse.RefreshToken)

	return tokenResponse.AccessToken, nil
}

// Выдача VIP через Twitch API
func GrantVIP(username string) error {
	userID, err := getUserIdByLogin(username)
	if err != nil {
		return err
	}

	broadcasterID := os.Getenv("TWITCH_CHANNEL_ID")
	token, err := RefreshTwitchToken()
	if err != nil {
		return err
	}
	clientId := os.Getenv("TWITCH_CLIENT_ID_BOT")

	url := fmt.Sprintf(
		"https://api.twitch.tv/helix/channels/vips?broadcaster_id=%s&user_id=%s",
		broadcasterID,
		userID,
	)

	req, err := http.NewRequest("POST", url, nil)
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Client-Id", clientId)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		log.Println(string(body))
		return fmt.Errorf("failed to grant VIP: %s (%s)", resp.Status, string(body))
	}

	return nil
}

func RevokeVIP(username string) error {
	userID, err := getUserIdByLogin(username)
	if err != nil {
		return err
	}

	broadcasterID := os.Getenv("TWITCH_CHANNEL_ID")
	token, err := RefreshTwitchToken()
	if err != nil {
		return err
	}
	clientId := os.Getenv("TWITCH_CLIENT_ID_BOT")

	url := fmt.Sprintf(
		"https://api.twitch.tv/helix/channels/vips?broadcaster_id=%s&user_id=%s",
		broadcasterID,
		userID,
	)

	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Client-Id", clientId)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to revoke VIP: %s (%s)", resp.Status, string(body))
	}

	return nil
}

func GetItemById(itemID int) (models.ShopItem, error) {
	var item models.ShopItem
	err := db.QueryRow(`SELECT id, type, title, cost, image FROM shop_items WHERE id = $1`, itemID).Scan(&item.ID, &item.Type, &item.Title, &item.Cost, &item.Image)
	if err != nil {
		log.Println("Failed to select from shop_items")
		return item, err
	}

	if item.Type == "badge" {
		err = db.QueryRow(`SELECT badge_id FROM shop_items WHERE id = $1`, itemID).Scan(&item.BadgeId)
		if err != nil {
			log.Println("Failed to select badge_id from shop_items")
			return item, err
		}
		return item, nil
	} else {
		return item, nil
	}
}

func SubtractRating(userID, itemID int) error {
	user, err := GetUserByID(userID)
	if err != nil {
		log.Println("Failed to get user")
		return err
	}

	item, err := GetItemById(itemID)
	if err != nil {
		log.Println("Failed to get item")
		return err
	}

	if user.Rating < item.Cost {
		return errors.New("not enough balance")
	} else {
		switch item.Type {
		case "vip":
			err = GrantVIP(user.Login) // Выдать вип
			if err != nil {
				return err
			}
		case "badge":
			_, err = db.Exec("INSERT INTO users_items (user_id, item_id) VALUES ($1, $2)", user.ID, itemID)
			if err != nil {
				log.Println("Failed to add to users_items")
				return err
			}

			// Сразу применить купленный бадж
			_, err = db.Exec("UPDATE users SET badge_id = $1 WHERE id = $2", item.BadgeId, userID)
			if err != nil {
				log.Println("Failed to apply")
				return err
			}
		}
		_, err = db.Exec("UPDATE users SET rating = $1 WHERE id = $2", user.Rating-item.Cost, userID)
		if err != nil {
			log.Println("Failed to increase rating")
			return err
		}
	}
	return nil
}

func BuyItem(userID, itemID int) error {
	err := SubtractRating(userID, itemID)
	if err != nil {
		return err
	}
	return nil
}

func HasBadge(userID int) bool {
	var id int

	err := db.QueryRow(`
			SELECT badge_id
			FROM users
			WHERE id = $1
		`, userID).Scan(&id)
	if err != nil {
		return false
	} else {
		return true
	}
}

func GetShopItems(userID int) []models.ShopItem {
	results := []models.ShopItem{}

	// Поиск предметов
	items, err := db.Query(`
			SELECT id, type, title, cost, image
			FROM shop_items
		`)
	if err == nil {
		defer items.Close()
		for items.Next() {
			var i models.ShopItem
			if err := items.Scan(&i.ID, &i.Type, &i.Title, &i.Cost, &i.Image); err == nil {
				if i.Type == "badge" {
					// Проверка на владение
					var count int
					err := db.QueryRow(`SELECT COUNT(*) FROM users_items WHERE user_id = $1 AND item_id = $2`, userID, i.ID).Scan(&count)
					if err != nil {
						return nil
					}
					if count > 0 {
						i.Owned = true
					} else {
						i.Owned = false
					}

					err = db.QueryRow(`SELECT badge_id FROM shop_items WHERE id = $1`, i.ID).Scan(&i.BadgeId)
					if err != nil {
						return nil
					}
				}
				user, _ := GetUserByID(userID)
				i.Rating = user.Rating
				results = append(results, i)
			}
		}
	}

	return results
}

func GetCases(userID int) []models.Case {
	results := []models.Case{}

	// Поиск кейсов
	items, err := db.Query(`
			SELECT id, title, description, price, image
			FROM cases
		`)
	if err == nil {
		defer items.Close()
		for items.Next() {
			var i models.Case
			if err := items.Scan(&i.ID, &i.Title, &i.Description, &i.Price, &i.Image); err == nil {
				user, _ := GetUserByID(userID)
				i.Rating = user.Rating
				results = append(results, i)
			}
		}
	}

	return results
}

func IsVIP(item_type string) bool {
	return item_type == "vip"
}

func SaveTokens(access_token, refresh_token string) error {
	_, err := db.Exec("UPDATE admin_tokens SET access_token = $1", access_token)
	if err != nil {
		return err
	}

	_, err = db.Exec("UPDATE admin_tokens SET refresh_token = $1", refresh_token)
	if err != nil {
		return err
	}

	return nil
}

func GetTokens() models.Tokens {
	var tokens models.Tokens
	var access_token, refresh_token string
	err := db.QueryRow(`SELECT access_token, refresh_token FROM admin_tokens WHERE id = 1`).Scan(&access_token, &refresh_token)
	if err != nil {
		return tokens
	}
	tokens.AccessToken = access_token
	tokens.RefreshToken = refresh_token
	return tokens
}

func GetTotalLikes(userID int) int {
	var result int
	err := db.QueryRow(`SELECT SUM(likes) FROM files WHERE user_id = $1`, userID).Scan(&result)
	if err != nil {
		log.Println("Getting user's (id: " + strconv.Itoa(userID) + " likes error: " + err.Error())
		return 0
	}
	return result
}

func GetTotalFucks(userID int) int {
	var result int
	err := db.QueryRow(`SELECT SUM(fucks) FROM files WHERE user_id = $1`, userID).Scan(&result)
	if err != nil {
		log.Println("Getting user's fucks error: " + err.Error())
		return 0
	}
	return result
}

func GetTotalPosts(userID int) int {
	var result int
	err := db.QueryRow(`SELECT COUNT(*) FROM files WHERE user_id = $1`, userID).Scan(&result)
	if err != nil {
		log.Println("Getting user's posts count error: " + err.Error())
		return 0
	}
	return result
}

func GetUserBadges(userID int) ([]models.Badge, error) {
	var result []models.Badge

	// Поиск пользователей
	badges, err := db.Query(`
			SELECT badges.id, badges.image
			FROM (
				-- Основной значок профиля
				SELECT badge_id
				FROM users
				WHERE id = $1 AND badge_id IS NOT NULL

				UNION

				-- Значки из магазина (users_items)
				SELECT shop_items.badge_id
				FROM users_items
				JOIN shop_items ON users_items.item_id = shop_items.id
				WHERE 
					users_items.user_id = $1
					AND shop_items.type = 'badge'
					AND shop_items.badge_id IS NOT NULL

				UNION

				-- Значки из инвентаря (inventory)
				SELECT cases_rewards.badge_id
				FROM inventory
				JOIN cases_rewards ON inventory.reward_id = cases_rewards.id
				WHERE 
					inventory.user_id = $1
					AND cases_rewards.type = 'badge'
					AND cases_rewards.badge_id IS NOT NULL
			) AS combined_badges
			JOIN badges ON badges.id = combined_badges.badge_id;
		`, userID)
	if err == nil {
		defer badges.Close()
		for badges.Next() {
			var b models.Badge
			if err := badges.Scan(&b.ID, &b.Image); err == nil {
				result = append(result, b)
			}
		}
	} else {
		log.Println("Getting user's badges error: " + err.Error())
		return nil, err
	}

	return result, nil
}

func CheckFollow(userID, targetID int) bool {
	var following bool
	err := db.QueryRow(`
		SELECT EXISTS(
			SELECT 1 FROM follows 
			WHERE user_id = $1 AND target_id = $2
		)
	`, userID, targetID).Scan(&following)
	if err != nil {
		log.Println("Checking follow error: " + err.Error())
		return false
	}
	return following
}

func HasFollowings(userID int) bool {
	var following bool
	err := db.QueryRow(`
		SELECT EXISTS(
			SELECT 1 FROM follows 
			WHERE user_id = $1
		)
	`, userID).Scan(&following)
	if err != nil {
		log.Println("Checking followings error: " + err.Error())
		return false
	}
	return following
}

func GetUserPosts(userID int, page int, limit int, contentType string, sort string, search string) ([]models.File, int, error) {
	var result []models.File
	var total int

	// Поиск пользователей
	files, err := db.Query(`
			SELECT id, file_name, title, thumbnail, uploaded_at, views, likes, type
			FROM files
			WHERE user_id = $1
			AND is_public = true
			AND is_moderated = true
			AND (title ILIKE '%' || $5 || '%' OR $5 = '')
			ORDER BY
			CASE WHEN $4 = 'popular' THEN views END DESC,
			CASE WHEN $4 = 'newest' THEN uploaded_at END DESC
			LIMIT $2 OFFSET (($3 - 1) * $2);
		`, userID, limit, page, sort, search)
	if err == nil {
		defer files.Close()
		for files.Next() {
			var f models.File
			if err := files.Scan(&f.ID, &f.FileName, &f.Title, &f.Thumbnail, &f.UploadedAt, &f.Views, &f.Likes, &f.Type); err == nil {
				total++
				if contentType == "image" && IsImageFile(f.FileName) {
					result = append(result, f)
				} else if contentType == "video" && IsVideoFile(f.FileName) {
					result = append(result, f)
				} else if contentType == "clip" && f.Type == "clip" {
					result = append(result, f)
				} else if contentType == "all" {
					result = append(result, f)
				}
			}
		}
	} else {
		log.Println("Getting user's posts error: " + err.Error())
		return nil, 0, err
	}

	return result, total, nil
}

func Subscribe(userID, targetID int) error {
	if !CheckFollow(userID, targetID) {
		_, err := db.Exec("INSERT INTO follows (user_id, target_id) VALUES ($1, $2);", userID, targetID)
		if err != nil {
			return err
		}

		var user_display_name, user_profile_image string
		err = db.QueryRow(`
		SELECT display_name, profile_image_url FROM users
		WHERE id = $1
		`, userID).Scan(&user_display_name, &user_profile_image)
		if err != nil {
			return err
		}

		_, err = db.Exec(`
		INSERT INTO notifications (user_id, author_id, notification, image, link, type)
		VALUES ($1, $2, $3, $4, $5, 'like')
	`, targetID, userID, user_display_name+" подписался на Вас!", user_profile_image, "https://ehworld.ru/user/"+strconv.Itoa(userID))
		if err != nil {
			return err
		}

		_, err = db.Exec("UPDATE users SET followers = (SELECT COUNT(*) FROM follows WHERE target_id = $1) WHERE id = $1;", targetID)
		if err != nil {
			return err
		}

		return nil
	}
	return errors.New("already following")
}

func Unsubscribe(userID, targetID int) error {
	if CheckFollow(userID, targetID) {
		_, err := db.Exec("DELETE FROM follows WHERE user_id = $1 AND target_id = $2;", userID, targetID)
		if err != nil {
			return err
		}

		_, err = db.Exec("UPDATE users SET followers = (SELECT COUNT(*) FROM follows WHERE target_id = $1) WHERE id = $1;", targetID)
		if err != nil {
			return err
		}

		return nil
	}
	return errors.New("not following")
}

func SaveBadge(image, title string, cost int) error {
	var id int
	err := db.QueryRow(`
		INSERT INTO badges (image)
		VALUES ($1) RETURNING id
	`, image).Scan(&id)
	if err != nil {
		return err
	}

	_, err = db.Exec("INSERT INTO shop_items (type, badge_id, title, cost, image) VALUES ('badge', $1, $2, $3, $4)", id, title, cost, image)
	if err != nil {
		return err
	}

	return err
}

func SaveCase(image, title, description string, price int) (int, error) {
	var id int
	err := db.QueryRow(`
		INSERT INTO cases (image, title, description, price)
		VALUES ($1, $2, $3, $4) RETURNING id
	`, image, title, description, price).Scan(&id)
	if err != nil {
		return -1, err
	}

	return id, err
}

func AddReward(reward models.CaseReward) error {
	switch reward.Type {
	case "vip":
		_, err := db.Exec("INSERT INTO cases_rewards (type, probability, case_id) VALUES ('vip', $1, $2)", reward.Probability, reward.CaseID)
		if err != nil {
			return err
		}
	case "badge":
		_, err := db.Exec("INSERT INTO cases_rewards (type, probability, case_id, badge_id) VALUES ('badge', $1, $2, $3)", reward.Probability, reward.CaseID, reward.BadgeID)
		if err != nil {
			return err
		}
	case "auk":
		_, err := db.Exec("INSERT INTO cases_rewards (type, probability, case_id, auk_value ) VALUES ('auk', $1, $2, $3)", reward.Probability, reward.CaseID, reward.AukValue)
		if err != nil {
			return err
		}
	}

	return nil
}

func GetCaseRewards(caseID int) ([]models.CaseReward, error) {
	var result []models.CaseReward

	// Поиск пользователей
	rewards, err := db.Query(`
			SELECT id, type, probability
			FROM cases_rewards
			WHERE case_id = $1
		`, caseID)
	if err == nil {
		defer rewards.Close()
		for rewards.Next() {
			var r models.CaseReward
			if err := rewards.Scan(&r.ID, &r.Type, &r.Probability); err == nil {
				switch r.Type {
				case "badge":
					var title string
					err = db.QueryRow(`
					SELECT cases_rewards.badge_id, badges.image, shop_items.title
					FROM cases_rewards, badges, shop_items
					WHERE cases_rewards.id = $1 AND cases_rewards.badge_id = badges.id AND shop_items.badge_id = badges.id
					`, r.ID).Scan(&r.BadgeID, &r.Image, &title)
					if err != nil {
						return nil, err
					}
					r.Title = "Значок " + title
				case "auk":
					err = db.QueryRow(`
					SELECT auk_value
					FROM cases_rewards
					WHERE id = $1
					`, r.ID).Scan(&r.AukValue)
					if err != nil {
						return nil, err
					}
					r.Image = "../static/img/auk.png"
					r.Title = strconv.Itoa(r.AukValue) + " рублей для аука"
				case "vip":
					r.Image = "../static/img/vip.png"
					r.Title = "Статус VIP в чате"
				}

				result = append(result, r)
			}
		}
	} else {
		log.Println("Getting case rewards error: " + err.Error())
		return nil, err
	}

	return result, nil
}

func GetCaseByID(caseID int) (models.Case, error) {
	var result models.Case
	err := db.QueryRow(`SELECT id, title, price, image
						FROM cases
						WHERE id = $1`, caseID).Scan(&result.ID, &result.Title, &result.Price, &result.Image)
	if err != nil {
		return result, err
	}

	return result, nil
}

func OpenCase(caseID, userID int) (*models.CaseReward, error) {
	caseData, err := GetCaseByID(caseID)
	if err != nil {
		return nil, err
	}

	// Проверяем баланс
	user, err := GetUserByID(userID)
	if err != nil {
		return nil, err
	}

	if user.Rating < caseData.Price {
		return nil, errors.New("insufficient balance")
	}

	// Получаем все награды для кейса
	rewards, err := GetCaseRewards(caseID)
	if err != nil {
		return nil, err
	}

	// Выбираем случайную награду на основе вероятностей
	var selectedReward *models.CaseReward
	randValue := rand.Float64()

	currentProb := 0.0
	for _, reward := range rewards {
		currentProb += reward.Probability
		if randValue <= currentProb {
			selectedReward = &reward
			break
		}
	}

	if selectedReward == nil {
		return nil, errors.New("failed to select reward")
	}

	// Обновляем баланс
	IncrementRating(userID, caseData.Price*-1)

	// Добавляем в инвентарь
	_, err = db.Exec("INSERT INTO inventory (user_id, reward_id) VALUES ($1, $2)", userID, selectedReward.ID)
	if err != nil {
		return nil, err
	}

	return selectedReward, nil
}

func GetUserInventory(userID int) ([]models.CaseReward, error) {
	var result []models.CaseReward

	rewards, err := db.Query(`
			SELECT DISTINCT cr.id, cr.type
			FROM cases_rewards cr
			JOIN inventory inv ON inv.reward_id = cr.id
			WHERE inv.user_id = $1
		`, userID)
	if err == nil {
		defer rewards.Close()
		for rewards.Next() {
			var r models.CaseReward
			if err := rewards.Scan(&r.ID, &r.Type); err == nil {
				switch r.Type {
				case "badge":
					var title string
					err = db.QueryRow(`
					SELECT DISTINCT ON (cr.badge_id) cr.badge_id, b.image, si.title
					FROM cases_rewards cr
					JOIN badges b ON cr.badge_id = b.id
					JOIN shop_items si ON b.id = si.badge_id
					WHERE cr.id = $1
					ORDER BY cr.badge_id;
					`, r.ID).Scan(&r.BadgeID, &r.Image, &title)
					if err != nil {
						return nil, err
					}
					r.Title = "Значок " + title
				case "auk":
					err = db.QueryRow(`
					SELECT auk_value
					FROM cases_rewards
					WHERE id = $1
					`, r.ID).Scan(&r.AukValue)
					if err != nil {
						return nil, err
					}
					r.Image = "../static/img/auk.png"
					r.Title = strconv.Itoa(r.AukValue) + " рублей для аука"
				case "vip":
					r.Image = "../static/img/vip.png"
					r.Title = "Статус VIP в чате"
				}

				result = append(result, r)
			}
		}
	} else {
		log.Println("Getting user's inventory error: " + err.Error())
		return nil, err
	}

	return result, nil
}

func ApplyBadge(badgeID, userID int) error {
	_, err := db.Exec("UPDATE users SET badge_id = $1 WHERE id = $2", badgeID, userID)
	if err != nil {
		return err
	} else {
		return nil
	}
}

func UserHasItem(userID, itemID int) bool {
	var hasItem int
	err := db.QueryRow(`
					SELECT COUNT(*)
					FROM inventory
					WHERE user_id = $1 AND reward_id = $2
					`, userID, itemID).Scan(&hasItem)
	if err != nil {
		return false
	}

	if hasItem > 0 {
		return true
	}
	return false
}

func ApplyItem(itemID, userID int, lot_name string) error {
	var item models.CaseReward
	err := db.QueryRow(`
					SELECT type
					FROM cases_rewards
					WHERE id = $1
					`, itemID).Scan(&item.Type)
	if err != nil {
		return err
	}

	// Проверка на наличие предмета в инвентаре
	if UserHasItem(userID, itemID) {
		switch item.Type {
		case "vip":
			user, err := GetUserByID(userID)
			if err != nil {
				return err
			}
			err = GrantVIP(user.Login)
			if err != nil {
				return err
			}

			_, err = db.Exec("DELETE FROM inventory WHERE user_id = $1 AND reward_id = $2", userID, itemID)
			if err != nil {
				return err
			}
		case "auk":
			_, err = db.Exec("INSERT INTO auk_submissions (user_id, lot) VALUES ($1, $2)", userID, lot_name)
			if err != nil {
				return err
			}

			_, err = db.Exec("DELETE FROM inventory WHERE user_id = $1 AND reward_id = $2", userID, itemID)
			if err != nil {
				return err
			}
		}
	} else {
		return errors.New("item is not in inventory")
	}

	return nil
}
