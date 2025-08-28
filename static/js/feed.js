class FeedManager {
    constructor() {
        this.postsContainer = document.getElementById('feed-posts');
        this.loadingIndicator = document.getElementById('loading-indicator');
        this.offset = 0;
        this.limit = 12;
        this.isLoading = false;
        this.hasMore = true;
        
        // Обработчик скролла
        window.addEventListener('scroll', () => this.handleScroll());
    }
    
    async loadInitialPosts() {
        this.showLoading();
        await this.loadPosts();
        this.hideLoading();
    }
    
    async loadPosts() {
        if (this.isLoading || !this.hasMore) return;
        
        this.isLoading = true;
        this.showLoading();
        
        try {
            const response = await fetch(`/api/feed?offset=${this.offset}&limit=${this.limit}`);
            const posts = await response.json();
            
            if (posts.length === 0) {
                this.hasMore = false;
            } else {
                this.renderPosts(posts);
                this.offset += posts.length;
            }
        } catch (error) {
            console.error('Ошибка загрузки постов:', error);
        } finally {
            this.isLoading = false;
            this.hideLoading();
        }
    }

    async toggleLike(postId, buttonElement) {
        const isLiked = buttonElement.dataset.liked === 'true';
        const likeCountElement = buttonElement.nextElementSibling;
        let currentLikes = parseInt(likeCountElement.textContent);

        try {
            const response = await fetch(`/api/like/${postId}`, {
                method: isLiked ? 'DELETE' : 'POST',
                headers: {
                    'Content-Type': 'application/json'
                }
            });

            if (response.ok) {
                // Обновляем состояние кнопки
                buttonElement.dataset.liked = !isLiked;
                buttonElement.classList.toggle('liked');
                
                // Обновляем счетчик
                likeCountElement.textContent = isLiked ? currentLikes - 1 : currentLikes + 1;
            } else {
                console.error('Ошибка при обновлении лайка');
            }
        } catch (error) {
            console.error('Сетевая ошибка:', error);
        }
    }

    async toggleFuck(postId, buttonElement) {
        const isFucked = buttonElement.dataset.fucked === 'true';
        const fuckyouCountElement = buttonElement.nextElementSibling;
        let currentFucks = parseInt(fuckyouCountElement.textContent);

        try {
            const response = await fetch(`/api/fuckyou/${postId}`, {
                method: isFucked ? 'DELETE' : 'POST',
                headers: {
                    'Content-Type': 'application/json'
                }
            });

            if (response.ok) {
                // Обновляем состояние кнопки
                buttonElement.dataset.fucked = !isFucked;
                buttonElement.classList.toggle('fuckyou');
                
                // Обновляем счетчик
                fuckyouCountElement.textContent = isFucked ? currentFucks - 1 : currentFucks + 1;
            } else {
                console.error('Ошибка при обновлении фака');
            }
        } catch (error) {
            console.error('Сетевая ошибка:', error);
        }
    }
    
    renderPosts(posts) {
        posts.forEach(post => {
            const postElement = this.createPostElement(post);
            this.postsContainer.appendChild(postElement);
        });
    }
    
    createPostElement(post) {
        const postElement = document.createElement('div');
        postElement.className = 'feed-post';
        postElement.dataset.postId = post.id;
        
        // Определяем тип медиа
        const isVideo = post.file_name.toLowerCase().endsWith('.mp4') || post.file_name.toLowerCase().endsWith('.mov') || post.file_name.toLowerCase().endsWith('.avi');
        const isClip = post.type === 'clip';
        let descriptionContent;
        
        if (post.description != '') {
            descriptionContent = `
                    <div class="post-description">
                        ${DOMPurify.sanitize(post.description)}
                    </div>`;
        } else {
            descriptionContent = ``;
        }

        // Генерируем медиа-контент в зависимости от типа
        let mediaContent;
        if (isClip) {
            // Для клипов используем специальный контейнер
            mediaContent = `
                <div class="clip-embed">
                    ${DOMPurify.sanitize(post.file_name)}
                </div>
            `;
        } else if (isVideo) {
            // Обычное видео
            mediaContent = `
                <div class="video-container">
                    <video id="mainVideo" poster="../static/uploads/${DOMPurify.sanitize(post.thumbnail)}">
                        <source src="../static/uploads/${DOMPurify.sanitize(post.file_name)}" type="video/mp4">
                    </video>
                    
                    <div class="video-loading">
                        <div class="video-loading-spinner"></div>
                    </div>
                    
                    <button class="video-play-btn"></button>
                    
                    <div class="video-controls">
                        <div class="progress-container">
                            <div class="progress-bar"></div>
                        </div>
                        
                        <div class="control-buttons">
                            <div class="left-controls">
                                <button class="control-btn play-pause-btn">
                                    <svg viewBox="0 0 24 24"><path d="M8 5v14l11-7z"/></svg>
                                </button>
                                
                                <div class="volume-container">
                                    <button class="control-btn volume-btn">
                                        <svg viewBox="0 0 24 24"><path d="M3 9v6h4l5 5V4L7 9H3zm13.5 3c0-1.77-1.02-3.29-2.5-4.03v8.05c1.48-.73 2.5-2.25 2.5-4.02zM14 3.23v2.06c2.89.86 5 3.54 5 6.71s-2.11 5.85-5 6.71v2.06c4.01-.91 7-4.49 7-8.77s-2.99-7.86-7-8.77z"/></svg>
                                    </button>
                                    <div class="volume-slider">
                                        <div class="volume-level"></div>
                                    </div>
                                </div>
                                
                                <div class="video-time">
                                    <span class="current-time">0:00</span>
                                    <span class="time-separator">/</span>
                                    <span class="duration">0:00</span>
                                </div>
                            </div>
                            
                            <div class="right-controls">
                                <button class="control-btn fullscreen-btn">
                                    <svg class="fullscreen-open" viewBox="0 0 24 24"><path d="M7 14H5v5h5v-2H7v-3zm-2-4h2V7h3V5H5v5zm12 7h-3v2h5v-5h-2v3zM14 5v2h3v3h2V5h-5z"/></svg>
                                    <svg class="fullscreen-close" viewBox="0 0 24 24" style="display:none;"><path d="M5 16h3v3h2v-5H5v2zm3-8H5v2h5V5H8v3zm6 11h2v-3h3v-2h-5v5zm2-11V5h-2v5h5V8h-3z"/></svg>
                                </button>
                            </div>
                        </div>
                    </div>
                </div>
            `;
        } else {
            // Изображение
            mediaContent = `<img src="../static/uploads/${DOMPurify.sanitize(post.file_name)}" alt="${DOMPurify.sanitize(post.title)}">`;
        }

        let badge = '';

        if (post.badge_image_url != '') {
            badge = `<img src="${post.badge_image_url}" alt="Badge" class="user-badge" width="20" height="20">`;
        }
        
        postElement.innerHTML = `
            <div class="post-card" data-post-id="${post.id}">
                <div class="post-header">
                    <img src="${post.author_image}" alt="Аватар" class="post-avatar" onclick="window.location.href = '/user/${post.author_name}'">
                    <div>
                        <div class="username-badge">
                            <div class="post-author" onclick="window.location.href = '/user/${post.author_name}'">${post.author_name}</div>
                            ${badge}
                        </div>
                        <span class="post-time">${post.format_time}</span>
                    </div>
                </div>
                
                <div class="media-container">
                    ${mediaContent}
                </div>
                
                <div class="post-footer">
                    <div class="post-caption">
                        ${DOMPurify.sanitize(post.title)}
                    </div>

                    ${DOMPurify.sanitize(descriptionContent)}

                    <div class="post-stats">
                        <div class="reactions-container">
                            <div class="like-container" data-post-id="${post.id}">
                                <button class="like-btn  ${post.is_liked ? 'liked' : ''}" data-liked="${post.is_liked}" onclick="toggleLike(this)">
                                    <svg width="24" height="24" viewBox="0 0 24 24">
                                        <path d="M16.5,3C13.605,3,12,5.09,12,5.09S10.395,3,7.5,3C4.462,3,2,5.462,2,8.5c0,4.171,4.912,8.8,10,12.5 c5.088-3.7,10-8.329,10-12.5C22,5.462,19.538,3,16.5,3z" fill="#65676B"/>
                                    </svg>
                                </button>
                                <span class="like-count">${post.likes}</span>
                            </div>

                            <div class="fuckyou-container" id="fuckyou-container">
                                <button id="fuckyouBtn" class="fuckyou-btn ${post.is_fucked ? 'fuckyou' : ''}" data-fucked="${post.is_fucked}" onclick="toggleFuck(this)">
                                    <svg width="24" height="24" viewBox="0 0 128 128">
                                        <path d="M50.98 122.96c-3.16 0-6.86-1.72-6.86-6.57c0-5.09-.42-10.72-5.45-13.19c-3.46-1.7-12.44-13.2-13.11-22.8c-.4-5.7 1.91-6.53 6.09-8.03c.42-.15.85-.3 1.28-.46c1.07-.4 1.07-.4.93-8.93c0-5.96 2.32-9.28 7.1-10.13c.47-.08.98-.12 1.53-.12c1.71 0 3.47.41 4.75.7c.89.21 1.53.35 2.05.35a2.68 2.68 0 0 0 2.75-2.32c.01-.06.01-.11.01-.17l.07-38c0-3.62 2.15-7.52 6.87-7.52c4.44 0 7.09 2.8 7.09 7.5l.84 37.15c0 .09.01.18.03.27c.33 1.59.68 2.51 1.61 2.79c.3.09.61.13.94.13c.59 0 1.15-.13 1.79-.29c.83-.2 1.86-.45 3.29-.45c.87 0 1.81.09 2.81.28c2.92.54 4.55 2.27 5.86 3.65c.94.99 1.75 1.85 2.87 2.07c.35.07.69.1 1.05.1c.68 0 1.32-.12 2.07-.26c1.19-.23 2.67-.51 5.26-.51c4.33 0 7.99 3.34 7.99 7.28l-.34 29.73c0 3.31-2.94 6.81-4.14 8.17c-3.05 3.43-4.99 7.81-4.99 12.33v2.03c0 4.14-3.98 5.03-7.38 5.05l-34.66.17z" fill="#65676B"/>
                                    </svg>
                                </button>
                                <span class="fuckyou-count">${post.fucks}</span>
                            </div>

                            <div class="comments-count-container">
                                <button id="commentsBtn" class="fuckyou-btn" onclick="window.location.href = '/post/${post.id}'">
                                    <img src="../static/img/comments.svg" width="24" height="24">
                                </button>
                                <span class="fuckyou-count">${post.comments}</span>
                            </div>
                        </div>
                        <div class="view-count">${post.format_views}</div>
                    </div>
                    
                    <div class="post-actions">
                        <button class="goto-post-btn" onclick="window.location.href = '/post/${post.id}'">Перейти к посту</button>
                    </div>
                    
                    
                </div>
            </div>
        `;

        const likeBtn = postElement.querySelector('.like-btn');
        likeBtn.addEventListener('click', () => this.toggleLike(post.id, likeBtn));

        const fuckyouBtn = postElement.querySelector('.fuckyou-btn');
        fuckyouBtn.addEventListener('click', () => this.toggleFuck(post.id, fuckyouBtn));

        // Инициализируем видео-плеер только для обычных видео (не клипов)
        if (isVideo && !isClip) {
            postElement.querySelectorAll('.video-container').forEach(container => {
                new VideoPlayer(container);
            });
        }

        return postElement;
    }
    
    handleScroll() {
        const scrollPosition = window.scrollY + window.innerHeight;
        const pageHeight = document.documentElement.scrollHeight;
        const threshold = 300;
        
        if (pageHeight - scrollPosition < threshold) {
            this.loadPosts();
        }
    }
    
    showLoading() {
        this.loadingIndicator.style.display = 'flex';
    }
    
    hideLoading() {
        this.loadingIndicator.style.display = 'none';
    }
}