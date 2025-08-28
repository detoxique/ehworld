// Загрузка топа пользователей
async function loadTopUsers() {
    const topUsersList = document.getElementById('topUsersList');
    
    topUsersList.innerHTML = '<div class="loading-users">Загрузка пользователей...</div>';
    
    try {
        const response = await fetch(`/api/top-authors`);
        const users = await response.json();
        
        if (users.length === 0) {
            topUsersList.innerHTML = '<div class="no-users">Нет данных о пользователях</div>';
            return;
        }
        
        let usersHtml = '';
        users.forEach(user => {
            usersHtml += `
                <div class="user-item" onclick="window.open('https://ehworld.ru/user/${user.display_name}', '_blank')">
                    <img src="${user.profile_image_url}" alt="${user.display_name}" class="user-avatar-sm">
                    <div class="user-info">
                        <a href="/user/${user.display_name}" class="user-name" target="_blank">${user.display_name}</a>
                        <div class="user-stats">
                            Подписчиков: ${user.followers}
                        </div>
                    </div>
                </div>
            `;
        });
        
        topUsersList.innerHTML = usersHtml;
    } catch (error) {
        console.error('Ошибка при загрузке топа пользователей:', error);
        topUsersList.innerHTML = '<div class="users-error">Ошибка загрузки</div>';
    }
}

// Инициализация при загрузке страницы
document.addEventListener('DOMContentLoaded', function() {
    // Загружаем топ пользователей при загрузке страницы
    loadTopUsers();
    
    // Обновляем при изменении сортировки
    document.getElementById('topUsersSort').addEventListener('change', loadTopUsers);
    
    // Обновляем каждую минуту
    setInterval(loadTopUsers, 60000);
});