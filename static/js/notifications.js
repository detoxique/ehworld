document.addEventListener('DOMContentLoaded', function() {
            const notificationButton = document.getElementById('notificationButton');
            const notificationDropdown = document.getElementById('notificationDropdown');
            
            if (notificationButton) {
                notificationButton.addEventListener('click', function(e) {
                    e.stopPropagation();
                    notificationDropdown.classList.toggle('show');
                    loadNotifications();
                });
            }
            
            // Закрытие при клике вне области
            document.addEventListener('click', function(e) {
                if (!notificationDropdown.contains(e.target) && 
                    !notificationButton.contains(e.target)) {
                    notificationDropdown.classList.remove('show');
                }
            });
            
            function loadNotifications() {
                const notificationList = document.getElementById('notificationList');
                
                // Очищаем предыдущие уведомления
                notificationList.innerHTML = '<div class="loading-notification">Загрузка...</div>';
                
                fetch('/api/notifications')
                    .then(response => response.json())
                    .then(notifications => {
                        notificationList.innerHTML = '';
                        
                        if (notifications.length === 0) {
                            notificationList.innerHTML = '<div class="empty-notification">Уведомлений нет</div>';
                            return;
                        }
                        
                        notifications.forEach(notification => {
                            const item = document.createElement('a');
                            item.href = notification.link;
                            item.className = 'notification-item';
                            
                            item.innerHTML = `
                                <img src="${notification.image}" alt="Notification" class="notification-image">
                                <div class="notification-content">
                                    <div class="notification-text">${notification.notification}</div>
                                    <div class="notification-time">${notification.time}</div>
                                </div>
                            `;
                            
                            notificationList.appendChild(item);
                        });
                    })
                    .catch(error => {
                        console.error('Ошибка загрузки уведомлений:', error);
                        notificationList.innerHTML = '<div class="error-notification">Уведомлений нет</div>';
                    });
            }
        });