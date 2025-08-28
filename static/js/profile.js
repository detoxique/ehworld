function getSubscribersString(num) {
  let lastTwo = num % 100;
  let lastOne = num % 10;
  
  if (lastTwo >= 11 && lastTwo <= 19) {
    return num + ' подписчиков';
  }
  
  switch (lastOne) {
    case 1:
      return num + ' подписчик';
    case 2:
    case 3:
    case 4:
      return num + ' подписчика';
    default:
      return num + ' подписчиков';
  }
}

function getViewsString(num) {
  let lastTwo = num % 100;
  let lastOne = num % 10;
  
  if (lastTwo >= 11 && lastTwo <= 19) {
    return num + ' просмотров';
  }
  
  switch (lastOne) {
    case 1:
      return num + ' просмотр';
    case 2:
    case 3:
    case 4:
      return num + ' просмотра';
    default:
      return num + ' просмотров';
  }
}

document.addEventListener('DOMContentLoaded', () => {
    const profileData = document.getElementById('profileData');
    const userId = profileData.dataset.userId;
    const isOwner = profileData.dataset.isOwner === 'true';
    
    // Элементы управления
    const postsGrid = document.getElementById('postsGrid');
    const paginationContainer = document.getElementById('pagination');
    const filterButtons = document.querySelectorAll('.filter-btn');
    const sortSelect = document.getElementById('sortSelect');
    const searchInput = document.getElementById('postSearchInput');
    const followButton = document.getElementById('followButton');
    const followersCounter = document.getElementById('followersCount');

    // Параметры загрузки
    let currentPage = 1;
    const itemsPerPage = 12;
    let currentType = 'all';
    let currentSort = 'newest';
    let currentSearch = '';

    // Загрузка постов
    async function loadPosts() {
        try {
            const url = new URL(`/api/posts/${userId}`, window.location.origin);
            url.searchParams.append('page', currentPage);
            url.searchParams.append('limit', itemsPerPage);
            url.searchParams.append('type', currentType);
            url.searchParams.append('sort', currentSort);
            url.searchParams.append('search', currentSearch);
            
            const response = await fetch(url);
            if (!response.ok) throw new Error('Ошибка загрузки постов');
            
            const data = await response.json();
            renderPosts(data.Posts);
            renderPagination(data.Total, data.TotalPages);
        } catch (error) {
            console.error('Ошибка:', error);
            postsGrid.innerHTML = '<p class="text-center">Не удалось загрузить посты</p>';
        }
    }

    // Рендеринг постов
    function renderPosts(posts) {
        postsGrid.innerHTML = '';
        
        if (posts.length === 0) {
            postsGrid.innerHTML = '<p class="text-center">Посты не найдены</p>';
            return;
        }
        
        posts.forEach(post => {
            const postCard = document.createElement('div');
            postCard.className = 'post-card';

            let content = '';
            if (post.type === 'clip') content = `
                        <img src="${DOMPurify.sanitize(post.thumbnail)}" alt="${DOMPurify.sanitize(post.title)}">
                        <div class="play-icon">▶</div>`;
            else if (post.file_name.match(/\.(mp4|mov|avi)$/i)) content = `
                        <img src="../static/uploads/${DOMPurify.sanitize(post.thumbnail)}" alt="${DOMPurify.sanitize(post.title)}">
                        <div class="play-icon">▶</div>
            `;
            else if (post.type === 'file') content = `<img src="../static/uploads/${DOMPurify.sanitize(post.file_name)}" alt="${DOMPurify.sanitize(post.title)}">`;
            
            postCard.innerHTML = `
                <a href="/post/${post.id}" class="post-card">
                    <div class="post-thumbnail">
                        ${content}
                    </div>
                    <div class="post-content">
                        <h3 class="post-title">${DOMPurify.sanitize(post.title)}</h3>
                        <div class="post-stats">
                            <span class="views">${getViewsString(post.views)}</span>
                        </div>
                    </div>
                </a>
            `;
            postsGrid.appendChild(postCard);
        });
    }

    // Пагинация
    function renderPagination(total, totalPages) {
        paginationContainer.innerHTML = '';
        
        if (totalPages <= 1) return;
        
        const prevButton = document.createElement('button');
        prevButton.className = 'page-btn';
        prevButton.innerHTML = '&larr;';
        prevButton.disabled = currentPage === 1;
        prevButton.addEventListener('click', () => {
            currentPage--;
            loadPosts();
        });
        paginationContainer.appendChild(prevButton);
        
        for (let i = 1; i <= totalPages; i++) {
            const pageButton = document.createElement('button');
            pageButton.className = `page-btn ${i === currentPage ? 'active' : ''}`;
            pageButton.textContent = i;
            pageButton.addEventListener('click', () => {
                currentPage = i;
                loadPosts();
            });
            paginationContainer.appendChild(pageButton);
        }
        
        const nextButton = document.createElement('button');
        nextButton.className = 'page-btn';
        nextButton.innerHTML = '&rarr;';
        nextButton.disabled = currentPage === totalPages;
        nextButton.addEventListener('click', () => {
            currentPage++;
            loadPosts();
        });
        paginationContainer.appendChild(nextButton);
    }

    // Обработчики событий
    filterButtons.forEach(btn => {
        btn.addEventListener('click', () => {
            filterButtons.forEach(b => b.classList.remove('active'));
            btn.classList.add('active');
            currentType = btn.dataset.type;
            currentPage = 1;
            loadPosts();
        });
    });
    
    sortSelect.addEventListener('change', () => {
        currentSort = sortSelect.value;
        currentPage = 1;
        loadPosts();
    });
    
    let searchTimeout;
    searchInput.addEventListener('input', () => {
        clearTimeout(searchTimeout);
        searchTimeout = setTimeout(() => {
            currentSearch = searchInput.value;
            currentPage = 1;
            loadPosts();
        }, 500);
    });
    
    // Подписка/отписка
    if (followButton && !isOwner) {
        followButton.addEventListener('click', async () => {
            try {
                const isFollowing = followButton.dataset.following === 'true';
                const method = isFollowing ? 'DELETE' : 'POST';
                
                const response = await fetch(`/api/follow/${userId}`, {
                    method: method,
                    headers: { 'Content-Type': 'application/json' }
                });
                
                if (response.ok) {
                    followButton.dataset.following = isFollowing ? 'false' : 'true';
                    followButton.textContent = isFollowing ? 'Подписаться' : 'Отписаться';
                    followersCount.textContent = isFollowing ? getSubscribersString(parseInt(followersCount.textContent) - 1) : getSubscribersString(parseInt(followersCount.textContent) + 1);
                }
            } catch (error) {
                console.error('Ошибка:', error);
            }
        });
    }

    // Инициализация
    loadPosts();
});