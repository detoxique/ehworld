document.addEventListener('DOMContentLoaded', function() {
            // Обработка применения значка
            const badgeItems = document.querySelectorAll('.badge-item');
            badgeItems.forEach(item => {
                item.addEventListener('click', function() {
                    const badgeId = this.dataset.id;
                    const badgeImage = this.dataset.image;
                    
                    // Отправляем запрос на сервер
                    fetch(`/api/apply-badge/${badgeId}`, {
                        method: 'POST',
                        headers: {
                            'Content-Type': 'application/json'
                        }
                    })
                    .then(response => response.status)
                    .then(data => {
                        // Обновляем UI
                            badgeItems.forEach(badge => {
                                badge.classList.remove('active-badge');
                                const check = badge.querySelector('.badge-check');
                                if (check) check.remove();
                            });
                            
                            this.classList.add('active-badge');
                            
                            // Добавляем галочку
                            const checkMark = document.createElement('div');
                            checkMark.className = 'badge-check';
                            checkMark.innerHTML = '✓';
                            this.appendChild(checkMark);
                            
                            // Обновляем текущий значок в шапке
                            const currentBadge = document.querySelector('.current-badge');
                            let badgeImg = currentBadge.querySelector('img');
                            
                            if (!badgeImg) {
                                badgeImg = document.createElement('img');
                                badgeImg.className = 'user-badge';
                                badgeImg.width = 40;
                                badgeImg.height = 40;
                                badgeImg.alt = 'Badge';
                                currentBadge.appendChild(badgeImg);
                            }
                            
                            badgeImg.src = badgeImage;
                    })
                    .catch(error => {
                        console.error('Ошибка:', error);
                        window.location.reload();
                    });
                });
            });

            // Обработка применения предметов
            const applyButtons = document.querySelectorAll('.apply-button');
            const auctionModal = document.getElementById('auctionModal');
            const lotNameInput = document.getElementById('lotName');
            const cancelAuctionBtn = document.getElementById('cancelAuction');
            const submitAuctionBtn = document.getElementById('submitAuction');
            
            let currentItemId = null;
            let currentItemType = null;
            
            applyButtons.forEach(button => {
                button.addEventListener('click', function() {
                    const itemCard = this.closest('.item-card');
                    currentItemId = itemCard.dataset.id;
                    currentItemType = itemCard.dataset.type;
                    
                    // Для предметов типа "auk" показываем модальное окно
                    if (currentItemType === 'auk') {
                        lotNameInput.value = '';
                        auctionModal.style.display = 'flex';
                    } else {
                        // Для других типов сразу отправляем запрос
                        applyItem(currentItemId);
                    }
                });
            });
            
            // Обработка отправки аукциона
            submitAuctionBtn.addEventListener('click', function() {
                const lotName = lotNameInput.value.trim();
                if (!lotName) {
                    alert('Пожалуйста, введите название лота');
                    return;
                }
                
                applyItem(currentItemId, { lot_name: lotName });
                auctionModal.style.display = 'none';
            });
            
            // Отмена аукциона
            cancelAuctionBtn.addEventListener('click', function() {
                auctionModal.style.display = 'none';
            });
            
            // Закрытие модального окна при клике вне его
            window.addEventListener('click', function(event) {
                if (event.target === auctionModal) {
                    auctionModal.style.display = 'none';
                }
            });
            
            // Функция применения предмета
            function applyItem(itemId, data = {}) {
                const url = `/api/apply-item/${itemId}`;
                
                fetch(url, {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json'
                    },
                    body: Object.keys(data).length ? JSON.stringify(data) : null
                })
                .then(response => response.json())
                .then(result => {
                    if (result.success) {
                        alert(result.message || 'Предмет успешно применен!');
                        // Обновляем страницу для отображения изменений
                        setTimeout(() => location.reload(), 1500);
                    } else {
                        alert('Ошибка: ' + (result.message || 'Неизвестная ошибка'));
                    }
                })
                .catch(error => {
                    console.error('Ошибка:', error);
                    alert('Произошла ошибка при применении предмета');
                });
            }
        });