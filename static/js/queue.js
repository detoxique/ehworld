// Функция для загрузки данных очереди
        async function loadQueue() {
            try {
                const response = await fetch('/api/admin/queue');
                if (!response.ok) {
                    throw new Error('Ошибка при загрузке данных');
                }
                
                const requests = await response.json();
                displayRequests(requests);
                
                // Обновляем время последнего обновления
                document.getElementById('refreshInfo').textContent = 
                    `Данные обновлены: ${new Date().toLocaleTimeString()}. Обновление каждые 5 секунд`;
            } catch (error) {
                console.error('Ошибка:', error);
                document.getElementById('refreshInfo').textContent = 
                    `Ошибка загрузки данных: ${error.message}. Попытка повторной загрузки через 5 секунд`;
            }
        }

        function showStatusMessage(message, isSuccess = true) {
            const statusElement = document.getElementById('statusMessage');
            statusElement.innerHTML = `
                <div class="status-message ${isSuccess ? 'status-success' : 'status-error'}">
                    ${message}
                </div>
            `;
            
            // Автоматически скрыть сообщение через 5 секунд
            setTimeout(() => {
                statusElement.innerHTML = '';
            }, 5000);
        }
        
        // Функция для отображения запросов
        function displayRequests(requests) {
            const container = document.getElementById('requestsQueue');
            
            if (!requests || requests.length === 0) {
                container.innerHTML = `
                    <div class="no-requests">
                        <h4>Очередь запросов пуста</h4>
                        <p>Новых запросов на модерацию нет</p>
                    </div>
                `;
                return;
            }
            
            container.innerHTML = requests.map(request => `
                <div class="request-card">
                    <div class="request-header">
                        <img src="${request.profile_image_url}" alt="Аватар" class="request-avatar">
                        <div class="request-user-info">
                            <h3 class="request-username">${request.display_name}</h3>
                        </div>
                        <div class="request-value">${request.auk_value} ₽</div>
                    </div>
                    <div class="request-content">
                        <div class="request-submission">${DOMPurify.sanitize(request.submission)}</div>
                    </div>
                    <div class="request-actions">
                        <button class="btn-delete" data-id="${request.id}" onclick="deleteRequest(${request.id})">
                            Удалить
                        </button>
                    </div>
                </div>
            `).join('');
        }

        // Функция для удаления запроса
        async function deleteRequest(id) {
            // Найти кнопку и отключить её
            const deleteButton = document.querySelector(`button[data-id="${id}"]`);
            if (deleteButton) {
                deleteButton.disabled = true;
                deleteButton.textContent = 'Удаление...';
            }
            
            try {
                const response = await fetch(`/api/admin/queue/${id}`, {
                    method: 'DELETE'
                });
                
                if (!response.ok) {
                    throw new Error('Ошибка при удалении запроса');
                }
                
                // Показать сообщение об успехе
                showStatusMessage('Запрос успешно удален');
                
                // Перезагрузить очередь
                loadQueue();
            } catch (error) {
                console.error('Ошибка:', error);
                showStatusMessage('Ошибка при удалении запроса: ' + error.message, false);
                
                // Восстановить кнопку
                if (deleteButton) {
                    deleteButton.disabled = false;
                    deleteButton.textContent = 'Удалить';
                }
            }
        }
        
        // Загружаем данные сразу при загрузке страницы
        document.addEventListener('DOMContentLoaded', () => {
            loadQueue();
            
            // Устанавливаем интервал для автоматического обновления каждые 5 секунд
            setInterval(loadQueue, 5000);
        });