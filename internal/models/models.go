package models

import "time"

type User struct {
	ID              int       `json:"-"`
	TwitchID        string    `json:"id"`
	Login           string    `json:"login"`
	DisplayName     string    `json:"display_name"`
	ProfileImageURL string    `json:"profile_image_url"`
	Email           string    `json:"email"`
	CreatedAt       time.Time `json:"created_at"`
	Rating          int       `json:"rating"`
	Role            string    `json:"role"`
	IsBanned        bool      `json:"is_banned"`
	Followers       int       `json:"followers"`
	Badge           string    `json:"badge_image_url"`
	CurrentBadgeID  int       `json:"current_badge_id"`
}

type File struct {
	ID          int       `json:"id"`
	UserID      int       `json:"user_id"`
	Title       string    `json:"title"`
	FileName    string    `json:"file_name"`
	FileSize    int64     `json:"file_size"`
	Thumbnail   string    `json:"thumbnail"`
	UploadedAt  time.Time `json:"uploaded_at"`
	IsPublic    bool      `json:"is_public"`
	IsModerated bool      `json:"is_moderated"`
	Type        string    `json:"type"`
	Views       int64     `json:"views"`
	Likes       int64     `json:"likes"`
	Description string    `json:"description"`
	Fucks       int64     `json:"fucks"`
}

type MainFile struct {
	ID         int    `json:"id"`
	Title      string `json:"title"`
	Thumbnail  string `json:"thumbnail"`
	Views      string `json:"views"`
	AuthorName string `json:"author_name"`
	Type       string `json:"type"`
	IsVideo    bool   `json:"is_video"`
	FileName   string `json:"file_name"`
}

type FileWithAuthor struct {
	File
	AuthorID              int    `json:"author_id"`
	AuthorName            string `json:"author_name"`
	AuthorProfileImageURL string `json:"author_image"`
	Badge                 string `json:"badge_image_url"`
}

type FeedFile struct {
	File
	AuthorID              int    `json:"author_id"`
	AuthorName            string `json:"author_name"`
	AuthorProfileImageURL string `json:"author_image"`
	FormatTime            string `json:"format_time"`
	FormatViews           string `json:"format_views"`
	IsLiked               bool   `json:"is_liked"`
	IsFucked              bool   `json:"is_fucked"`
	Comments              int    `json:"comments"`
	Badge                 string `json:"badge_image_url"`
}

type CommentWithAuthor struct {
	Comment
	Replies               []CommentWithAuthor `json:"replies,omitempty"`
	AuthorProfileImageURL string              `json:"author_image"`
	AuthorID              int                 `json:"author_id"`
	HasLiked              bool                `json:"has_liked"`
	Badge                 string              `json:"badge_image_url"`
}

type Comment struct {
	ID         int       `json:"id"`
	UserID     int       `json:"user_id"`
	FileID     int       `json:"file_id"`
	ParentID   int       `json:"parent_id"`
	Text       string    `json:"text"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
	Likes      int       `json:"likes"`
	AuthorName string    `json:"author_name"`
}

type Like struct {
	UserID int `json:"user_id"`
	FileID int `json:"file_id"`
}

type SearchResult struct {
	ID          int    `json:"id"`
	Title       string `json:"title,omitempty"`
	DisplayName string `json:"display_name,omitempty"`
	Username    string `json:"username,omitempty"`
	Type        string `json:"type"` // "post" или "user"
}

type UserSearchResult struct {
	ID              int    `json:"id"`
	DisplayName     string `json:"display_name,omitempty"`
	ProfileImageURL string `json:"profile_image_url"`
}

type ModerationPost struct {
	ID           int       `json:"id"`
	AuthorID     int       `json:"author_id"`
	AuthorName   string    `json:"author_name"`
	AuthorAvatar string    `json:"author_avatar"`
	UploadedAt   time.Time `json:"uploaded_at"`
	FileName     string    `json:"file_name"`
	Title        string    `json:"title"`
	Type         string    `json:"type"`
	Description  string    `json:"description"`
}

type ModerationResponse struct {
	Posts   []ModerationPost `json:"posts"`
	HasMore bool             `json:"has_more"`
}

type ClipResponse struct {
	Data []struct {
		ThumbnailURL string `json:"thumbnail_url"`
	} `json:"data"`
}

type TwitchTokenResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`
	TokenType   string `json:"token_type"`
}

type Badge struct {
	ID    int    `json:"id"`
	Image string `json:"image"`
	Title string `json:"title"`
	Cost  int    `json:"cost"`
}

type Stats struct {
	TotalUsers int64 `json:"total_users"`
	TotalViews int64 `json:"total_views"`
	TotalPosts int64 `json:"total_posts"`
	TotalLikes int64 `json:"total_likes"`
}

type Notification struct {
	ID            int       `json:"id"`
	Notification  string    `json:"notification"`
	Image         string    `json:"image"`
	Link          string    `json:"link"`
	FormattedTime string    `json:"time"`
	CreatedAt     time.Time `json:"created_at"`
}

type ShopItem struct {
	ID      int    `json:"id"`
	Type    string `json:"type"`
	BadgeId int    `json:"badge_id"`
	Title   string `json:"title"`
	Cost    int    `json:"cost"`
	Image   string `json:"image_url"`
	Owned   bool   `json:"owned"`
	Rating  int    `json:"rating"`
}

type Tokens struct {
	AccessToken  string
	RefreshToken string
}

type Case struct {
	ID          int    `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Price       int    `json:"Price"`
	Image       string `json:"image"`
	Rating      int    `json:"rating"`
}

type CaseReward struct {
	ID          int     `json:"id"`
	CaseID      int     `json:"case_id"`
	Type        string  `json:"type"`
	Probability float64 `json:"probability"`
	BadgeID     int     `json:"badge_id"`
	AukValue    int     `json:"auk_value"`
	Image       string  `json:"image"`
	Title       string  `json:"title"`
}
