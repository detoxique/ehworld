document.addEventListener('DOMContentLoaded', () => {
    const searchInput = document.getElementById('moderatorSearch');
    const searchBanInput = document.getElementById('banSearch');
    const searchResults = document.getElementById('searchResultsUsers');
    const searchBanResults = document.getElementById('searchResultsBanUsers');
    const addBtn = document.getElementById('addModeratorBtn');
    const banBtn = document.getElementById('banUserBtn');
    const moderatorsList = document.getElementById('moderatorsList');
    const bannedList = document.getElementById('bannedList');
    
    let searchTimeout;
    let selectedUser = null;

    let selectedBanUser = null;

    // Загрузка списка модераторов
    function loadModerators() {
        fetch('/api/admin/moderators')
            .then(response => response.json())
            .then(data => renderModerators(data))
            .catch(error => console.error('Error loading moderators:', error));
    }

    // Загрузка списка забаненных
    function loadBanned() {
        fetch('/api/admin/banned')
            .then(response => response.json())
            .then(data => renderBanned(data))
            .catch(error => console.error('Error loading banned users:', error));
    }
    
    // Отрисовка списка модераторов
    function renderModerators(moderators) {
        moderatorsList.innerHTML = '';
        
        moderators.forEach(user => {
            const row = document.createElement('tr');
            
            row.innerHTML = `
                <td>
                    <img src="${user.profile_image_url}" 
                         alt="Аватар" 
                         class="moderator-avatar">
                </td>
                <td>${user.display_name}</td>
                <td>
                    <select class="role-select" 
                            data-id="${user.id}"
                            data-username="${user.display_name}"
                            onchange="changeRole(this)">
                        <option value="moderator" ${user.role === 'moderator' ? 'selected' : ''}>moderator</option>
                        <option value="user" ${user.role === 'user' ? 'selected' : ''}>user</option>
                    </select>
                </td>
            `;
            
            moderatorsList.appendChild(row);
        });
    }

    // Отрисовка списка забаненных
    function renderBanned(banned) {
        bannedList.innerHTML = '';
        
        banned.forEach(user => {
            const row = document.createElement('tr');
            
            row.innerHTML = `
                <td>
                    <img src="${user.profile_image_url}" 
                         alt="Аватар" 
                         class="moderator-avatar">
                </td>
                <td>${user.display_name}</td>
                <td>
                    <select class="status-select" 
                            data-id="${user.id}"
                            data-username="${user.display_name}"
                            onchange="changeStatus(this)">
                        <option value="banned" ${user.is_banned === 'true' ? 'selected' : ''}>Заблокирован</option>
                        <option value="unbanned" ${user.is_banned === 'false' ? 'selected' : ''}>Разблокирован</option>
                    </select>
                </td>
            `;
            
            bannedList.appendChild(row);
        });
    }
    
    // Поиск пользователей
    searchInput.addEventListener('input', () => {
        const query = searchInput.value.trim();
        
        if (query.length < 2) {
            searchResults.innerHTML = '';
            searchResults.style.display = 'none';
            return;
        }
        
        clearTimeout(searchTimeout);
        searchTimeout = setTimeout(() => {
            fetch(`/api/admin/users?q=${encodeURIComponent(query)}`)
                .then(response => response.json())
                .then(users => {
                    if (users.length === 0) {
                        searchResults.innerHTML = '<div class="result-item">Пользователи не найдены</div>';
                        searchResults.style.display = 'block';
                        return;
                    }
                    
                    searchResults.innerHTML = '';
                    users.forEach(user => {
                        const item = document.createElement('div');
                        item.className = 'result-item';
                        item.innerHTML = `
                            <img src="${user.profile_image_url}" 
                                 alt="${user.display_name}" 
                                 class="result-avatar">
                            <span>${user.display_name}</span>
                        `;
                        
                        item.addEventListener('click', () => {
                            searchInput.value = user.display_name;
                            selectedUser = user;
                            searchResults.style.display = 'none';
                        });
                        
                        searchResults.appendChild(item);
                    });
                    
                    searchResults.style.display = 'block';
                });
        }, 300);
    });

    searchBanInput.addEventListener('input', () => {
        const query = searchBanInput.value.trim();
        
        if (query.length < 2) {
            searchBanResults.innerHTML = '';
            searchBanResults.style.display = 'none';
            return;
        }
        
        clearTimeout(searchTimeout);
        searchTimeout = setTimeout(() => {
            fetch(`/api/admin/users?q=${encodeURIComponent(query)}`)
                .then(response => response.json())
                .then(users => {
                    if (users.length === 0) {
                        searchBanResults.innerHTML = '<div class="result-item">Пользователи не найдены</div>';
                        searchBanResults.style.display = 'block';
                        return;
                    }
                    
                    searchBanResults.innerHTML = '';
                    users.forEach(user => {
                        const item = document.createElement('div');
                        item.className = 'result-item';
                        item.innerHTML = `
                            <img src="${user.profile_image_url}" 
                                 alt="${user.display_name}" 
                                 class="result-avatar">
                            <span>${user.display_name}</span>
                        `;
                        
                        item.addEventListener('click', () => {
                            searchBanInput.value = user.display_name;
                            selectedBanUser = user;
                            searchBanResults.style.display = 'none';
                            console.log(selectedBanUser.display_name);
                        });
                        
                        searchBanResults.appendChild(item);
                    });
                    
                    searchBanResults.style.display = 'block';
                });
        }, 300);
    });
    
    // Добавление модератора
    addBtn.addEventListener('click', () => {
        if (!selectedUser || !selectedUser.display_name) {
            alert('Пожалуйста, выберите пользователя из списка');
            return;
        }
        
        fetch(`/api/admin/moderatorrole/${selectedUser.display_name}`, {
            method: 'POST'
        })
        .then(response => {
            if (response.ok) {
                loadModerators();
                searchInput.value = '';
                selectedUser = null;
            } else {
                alert('Ошибка при добавлении модератора');
            }
        })
        .catch(error => console.error('Error adding moderator:', error));
    });

    // Блокировка
    banBtn.addEventListener('click', () => {
        if (!selectedBanUser || !selectedBanUser.display_name) {
            alert('Пожалуйста, выберите пользователя из списка');
            return;
        }
        
        fetch(`/api/moderation/banusername/${selectedBanUser.display_name}`, {
            method: 'POST'
        })
        .then(response => {
            if (response.ok) {
                loadBanned();
                searchBanInput.value = '';
                selectedBanUser = null;
            } else {
                alert('Ошибка при блокировке пользователя');
            }
        })
        .catch(error => console.error('Error ban user:', error));
    });
    
    // Изменение роли
    window.changeRole = function(select) {
        const id = select.getAttribute('data-id');
        const username = select.getAttribute('data-username');
        const role = select.value;
        
        if (role === 'user') {
            if (!confirm(`Вы уверены, что хотите удалить ${username} из модераторов?`)) {
                select.value = 'moderator';
                return;
            }
            
            fetch(`/api/admin/moderatorrole/${username}`, {
                method: 'DELETE'
            })
            .then(response => {
                if (response.ok) {
                    loadModerators();
                } else {
                    alert('Ошибка при изменении роли');
                    select.value = 'moderator';
                }
            })
            .catch(error => {
                console.error('Error changing role:', error);
                select.value = 'moderator';
            });
        }
    };

    // Разбан
    window.changeStatus = function(select) {
        const id = select.getAttribute('data-id');
        const username = select.getAttribute('data-username');
        const role = select.value;
        
        if (role === 'unbanned') {
            if (!confirm(`Вы уверены, что хотите разблокировать ${username}?`)) {
                select.value = 'banned';
                return;
            }
            
            fetch(`/api/moderation/banusername/${username}`, {
                method: 'DELETE'
            })
            .then(response => {
                if (response.ok) {
                    loadBanned();
                } else {
                    alert('Ошибка при разблокировке');
                    select.value = 'banned';
                }
            })
            .catch(error => {
                console.error('Error unban user:', error);
                select.value = 'banned';
            });
        }
    };
    
    // Скрытие результатов при клике вне области
    document.addEventListener('click', (e) => {
        if (!e.target.closest('.ban-user-search-container')) {
            searchResults.style.display = 'none';
        }
    });
    
    // Инициализация
    loadModerators();
    loadBanned();
});

document.addEventListener('DOMContentLoaded', () => {
    // Элементы управления
    const uploadArea = document.getElementById('uploadArea');
    const fileInput = document.getElementById('fileInput');
    const badgePreview = document.querySelector('.preview .user-badge');
    const titleInput = document.getElementById('titleInput');
    const costInput = document.getElementById('costInput');
    const addBadgeBtn = document.getElementById('addBadgeBtn');

    // Кейс
    const uploadCaseArea = document.getElementById('uploadCaseArea');
    const caseFileInput = document.getElementById('caseFileInput');
    const casePreview = document.querySelector('.case-image .image-case');
    const caseTitle = document.getElementById('caseTitle');
    const caseTitleInput = document.getElementById('titleCaseInput');
    const caseDescription = document.getElementById('caseDescription');
    const caseDescriptionInput = document.getElementById('descriptionCaseInput');
    const casePrice = document.getElementById('casePrice');
    const costCaseInput = document.getElementById('costCaseInput');
    const addCaseBtn = document.getElementById('addCaseBtn');
    
    // Изначально скрываем превью (если нет файла)
    badgePreview.style.display = 'none';

    casePreview.style.display = 'none';

    // Обработчик клика по области загрузки
    uploadArea.addEventListener('click', () => {
        fileInput.click();
    });

    uploadCaseArea.addEventListener('click', () => {
        caseFileInput.click();
    });

    // Обработчики для Drag-and-Drop
    ['dragover', 'dragenter'].forEach(eventName => {
        uploadArea.addEventListener(eventName, (e) => {
            e.preventDefault();
            uploadArea.classList.add('dragover');
        });
    });

    ['dragleave', 'drop'].forEach(eventName => {
        uploadArea.addEventListener(eventName, (e) => {
            e.preventDefault();
            uploadArea.classList.remove('dragover');
        });
    });

    // Кейс
    ['dragover', 'dragenter'].forEach(eventName => {
        uploadCaseArea.addEventListener(eventName, (e) => {
            e.preventDefault();
            uploadCaseArea.classList.add('dragover');
        });
    });

    ['dragleave', 'drop'].forEach(eventName => {
        uploadCaseArea.addEventListener(eventName, (e) => {
            e.preventDefault();
            uploadCaseArea.classList.remove('dragover');
        });
    });

    // Обработка сброса файла
    uploadArea.addEventListener('drop', (e) => {
        const file = e.dataTransfer.files[0];
        if (file) processFile(file);
    });

    uploadCaseArea.addEventListener('drop', (e) => {
        const file = e.dataTransfer.files[0];
        if (file) processFile(file);
    });

    // Обработчик выбора файла
    fileInput.addEventListener('change', () => {
        if (fileInput.files.length > 0) {
            processFile(fileInput.files[0]);
        }
    });

    caseFileInput.addEventListener('change', () => {
        if (caseFileInput.files.length > 0) {
            processCaseFile(caseFileInput.files[0]);
        }
    });

    // Ввод данных кейса
    caseTitleInput.addEventListener('input', () => {
        caseTitle.textContent = caseTitleInput.value.trim();
    });

    caseDescriptionInput.addEventListener('input', () => {
        caseDescription.textContent = caseDescriptionInput.value.trim();
    });

    costCaseInput.addEventListener('input', () => {
        casePrice.textContent = costCaseInput.value.trim() + ' э';
    });

    // Функция обработки файла
    function processFile(file) {
        // Проверка типа файла
        if (!file.type.match('image.*')) {
            alert('Пожалуйста, выберите изображение (JPG, PNG, GIF)');
            return;
        }

        // Создание превью
        const reader = new FileReader();
        reader.onload = (e) => {
            badgePreview.src = e.target.result;
            badgePreview.style.display = 'inline';
        };
        reader.readAsDataURL(file);
    }

    function processCaseFile(file) {
        // Проверка типа файла
        if (!file.type.match('image.*')) {
            alert('Пожалуйста, выберите изображение (JPG, PNG, GIF)');
            return;
        }

        // Создание превью
        const reader = new FileReader();
        reader.onload = (e) => {
            casePreview.src = e.target.result;
            casePreview.style.display = 'inline';
        };
        reader.readAsDataURL(file);
    }

    // Обработчик кнопки добавления
    addBadgeBtn.addEventListener('click', async () => {
        // Валидация полей
        if (!fileInput.files || fileInput.files.length === 0) {
            alert('Пожалуйста, загрузите изображение значка');
            return;
        }

        const title = titleInput.value.trim();
        if (!title) {
            alert('Введите название значка');
            titleInput.focus();
            return;
        }

        const cost = parseFloat(costInput.value);
        if (isNaN(cost) || cost < 0) {
            alert('Введите корректную стоимость (число больше или равно 0)');
            costInput.focus();
            return;
        }

        // Подготовка данных для отправки
        const formData = new FormData();
        formData.append('file', fileInput.files[0]);
        formData.append('title', title);
        formData.append('cost', cost.toString());

        try {
            // Отправка запроса на сервер
            const response = await fetch('/api/admin/uploadbadge', {
                method: 'POST',
                body: formData
            });

            const result = await response.status;

            if (response.ok) {
                alert('Значок успешно добавлен!');
                // Сброс формы
                fileInput.value = '';
                titleInput.value = '';
                costInput.value = '';
                badgePreview.style.display = 'none';
            } else {
                throw new Error(result.message || 'Ошибка сервера');
            }
        } catch (error) {
            alert(`Ошибка: ${error.message}`);
            console.error('Ошибка добавления значка:', error);
        }
    });

    // Элементы управления наградами
    const rewardTypeSelect = document.getElementById('rewardType');
    const rewardDetails = document.getElementById('rewardDetails');
    const addRewardBtn = document.getElementById('addRewardBtn');
    const rewardsList = document.getElementById('rewardsList');

    // Хранилище для наград кейса
    let caseRewards = [];

    // Загрузка значков для выбора
    let badges = [];

    async function loadBadges() {
        try {
            console.log('загружаем значки');
            const response = await fetch('/api/admin/badges');
            badges = await response.json();
        } catch (error) {
            console.error('Ошибка загрузки значков:', error);
        }
    }

    // Обработчик изменения типа награды
    rewardTypeSelect.addEventListener('change', () => {
        const type = rewardTypeSelect.value;
        let html = '';
        
        switch (type) {
            case 'vip':
                html = `
                    <div class="vip-reward">
                        <img src="../static/img/vip.png" alt="VIP" class="reward-preview" width="50">
                        <div class="reward-probability">
                            <label>Вероятность (0-1):</label>
                            <input type="number" id="vipProbability" step="0.01" min="0" max="1" value="0.1" class="form-control">
                        </div>
                    </div>
                `;
                break;
                
            case 'badge':
                let options = '<option value="">Выберите значок</option>';
                badges.forEach(badge => {
                    options += `
                        <option value="${badge.id}">
                            ${badge.title} (${badge.cost} э)
                        </option>
                    `;
                });
                
                html = `
                    <div class="badge-reward">
                        <div class="badge-select-container">
                            <select id="badgeSelect" class="form-control">
                                ${options}
                            </select>
                        </div>
                        <div class="reward-probability">
                            <label>Вероятность (0-1):</label>
                            <input type="number" id="badgeProbability" step="0.01" min="0" max="1" value="0.1" class="form-control">
                        </div>
                    </div>
                `;
                break;
                
            case 'auk':
                html = `
                    <div class="auk-reward">
                        <img src="../static/img/auk.png" alt="AUK" class="reward-preview" width="50">
                        <div>
                            <label>Количество рублей:</label>
                            <input type="number" id="aukValue" min="1" value="100" class="form-control">
                        </div>
                        <div class="reward-probability">
                            <label>Вероятность (0-1):</label>
                            <input type="number" id="aukProbability" step="0.01" min="0" max="1" value="0.1" class="form-control">
                        </div>
                    </div>
                `;
                break;
        }
        
        rewardDetails.innerHTML = html;
    });

    // Добавление награды
    addRewardBtn.addEventListener('click', () => {
        const type = rewardTypeSelect.value;
        let reward;
        
        switch (type) {
            case 'vip':
                const vipProb = parseFloat(document.getElementById('vipProbability').value);
                if (isNaN(vipProb)) {
                    alert('Введите корректную вероятность');
                    return;
                }
                reward = {
                    type: 'vip',
                    probability: vipProb
                };
                break;
                
            case 'badge':
                const badgeId = document.getElementById('badgeSelect').value;
                if (!badgeId) {
                    alert('Выберите значок');
                    return;
                }
                const badgeProb = parseFloat(document.getElementById('badgeProbability').value);
                if (isNaN(badgeProb)) {
                    alert('Введите корректную вероятность');
                    return;
                }
                
                const badge = badges.find(b => b.id == badgeId);
                reward = {
                    type: 'badge',
                    badgeId: badgeId,
                    badgeTitle: badge.title,
                    badgeImage: badge.image,
                    probability: badgeProb
                };
                break;
                
            case 'auk':
                const aukValue = parseInt(document.getElementById('aukValue').value);
                if (isNaN(aukValue)) {
                    alert('Введите корректное количество рублей');
                    return;
                }
                const aukProb = parseFloat(document.getElementById('aukProbability').value);
                if (isNaN(aukProb)) {
                    alert('Введите корректную вероятность');
                    return;
                }
                reward = {
                    type: 'auk',
                    aukValue: aukValue,
                    probability: aukProb
                };
                break;
        }
        
        caseRewards.push(reward);
        renderRewardsList();
    });

    // Отрисовка списка наград
    function renderRewardsList() {
        rewardsList.innerHTML = '';
        
        caseRewards.forEach((reward, index) => {
            const card = document.createElement('div');
            card.className = 'reward-card';
            
            let content = '';
            switch (reward.type) {
                case 'vip':
                    content = `
                        <img src="../static/img/vip.png" alt="VIP" class="reward-image">
                        <div class="reward-info">
                            <div>VIP</div>
                            <div class="reward-probability">
                                Вероятность: <strong>${reward.probability}</strong>
                            </div>
                        </div>
                    `;
                    break;
                    
                case 'badge':
                    content = `
                        <img src="${reward.badgeImage}" alt="${reward.badgeTitle}" class="reward-image">
                        <div class="reward-info">
                            <div>${reward.badgeTitle}</div>
                            <div class="reward-probability">
                                Вероятность: <strong>${reward.probability}</strong>
                            </div>
                        </div>
                    `;
                    break;
                    
                case 'auk':
                    content = `
                        <img src="../static/img/auk.png" alt="AUK" class="reward-image">
                        <div class="reward-info">
                            <div>${reward.aukValue} рублей</div>
                            <div class="reward-probability">
                                Вероятность: <strong>${reward.probability}</strong>
                            </div>
                        </div>
                    `;
                    break;
            }
            
            card.innerHTML = content + `<button class="remove-reward" data-index="${index}">×</button>`;
            rewardsList.appendChild(card);
        });
        
        // Обработчики удаления
        document.querySelectorAll('.remove-reward').forEach(btn => {
            btn.addEventListener('click', (e) => {
                const index = e.target.getAttribute('data-index');
                caseRewards.splice(index, 1);
                renderRewardsList();
            });
        });
    }

    // Функция добавления наград к кейсу
    async function addRewardsToCase(caseId) {
        for (const reward of caseRewards) {
            const formData = new FormData();
            formData.append('case_id', caseId);
            formData.append('type', reward.type);
            formData.append('probability', reward.probability);
            
            if (reward.type === 'badge') {
                formData.append('badge_id', reward.badgeId);
            } else if (reward.type === 'auk') {
                formData.append('auk_value', reward.aukValue);
            }
            
            try {
                const response = await fetch('/api/admin/add-rewards', {
                    method: 'POST',
                    body: formData
                });
                
                if (!response.ok) {
                    console.error('Ошибка при добавлении награды:', await response.text());
                }
            } catch (error) {
                console.error('Ошибка при добавлении награды:', error);
            }
        }
    }

    // Обработчик кнопки добавления
    addCaseBtn.addEventListener('click', async () => {
        // Валидация полей
        if (!caseFileInput.files || caseFileInput.files.length === 0) {
            alert('Пожалуйста, загрузите изображение кейса');
            return;
        }

        const title = caseTitleInput.value.trim();
        if (!title) {
            alert('Введите название кейса');
            titleInput.focus();
            return;
        }

        const description = caseDescriptionInput.value.trim();

        const price = parseFloat(costCaseInput.value);
        if (isNaN(price) || price < 0) {
            alert('Введите корректную стоимость (число больше или равно 0)');
            costInput.focus();
            return;
        }

        // Подготовка данных для отправки
        const formData = new FormData();
        formData.append('file', caseFileInput.files[0]);
        formData.append('title', title);
        formData.append('description', description);
        formData.append('price', price.toString());

        try {
            // Отправка запроса на сервер
            const response = await fetch('/api/admin/add-case', {
                method: 'POST',
                body: formData
            });

            if (response.ok) {
                alert('Кейс успешно добавлен!');
                // Сброс формы
                caseFileInput.value = '';
                caseTitleInput.value = '';
                caseDescriptionInput.value = '';
                caseDescriptionInput.value = '';
                casePreview.style.display = 'none';

                const caseData = await response.json();
                console.log(caseData.id);
                const caseId = caseData.id;

                // Отправка наград
                await addRewardsToCase(caseId);
            } else {
                throw new Error(result.message || 'Ошибка сервера');
            }
        } catch (error) {
            alert(`Ошибка: ${error.message}`);
            console.error('Ошибка добавления кейса:', error);
        }
    });

    loadBadges();
    rewardTypeSelect.dispatchEvent(new Event('change'));
});