// Обработка покупки значков
document.addEventListener('DOMContentLoaded', function() {
    const buyButtons = document.querySelectorAll('.buy-button:not(:disabled)');
    
    buyButtons.forEach(button => {
        button.addEventListener('click', function() {
            const badgeId = this.getAttribute('data-id');
                    
            fetch('/api/buy_item/'+badgeId, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json'
                },
                body: JSON.stringify({ badge_id: badgeId })
            })
            .then(response => response.status)
            .then(data => {
                if (data.success) {
                    location.reload(); // Обновляем страницу
                } else {
                    location.reload();
                    alert('Ошибка: ' + data.message);
                }
            })
            .catch(error => {
                console.error('Ошибка:', error);
                //location.reload();
                alert('Не удалось купить предмет');
            });
        });
    });

    const caseButtons = document.querySelectorAll('.badge-card.case-card .open-case-button:not(:disabled)');
    const modal = document.getElementById('caseModal');
    const closeBtn = document.querySelector('.close');
    const openCaseBtn = document.getElementById('openCaseBtn');
    const rewardPopup = document.getElementById('rewardPopup');
    const closeRewardBtn = document.getElementById('closeRewardBtn');
    
    let currentCaseId = null;
    
    // Открытие модального окна кейса
    caseButtons.forEach(button => {
        button.addEventListener('click', function() {
            currentCaseId = this.getAttribute('data-id');
            const card = this.closest('.badge-card');
            
            // Заполнение данных о кейсе
            document.getElementById('modalCaseTitle').textContent = 
                card.querySelector('.badge-title').textContent;
                
            document.getElementById('modalCaseImage').src = 
                card.querySelector('.badge-image img').src;
                
            document.getElementById('modalCaseDescription').textContent = 
                card.querySelector('.case-description').textContent;
                
            const price = card.querySelector('.badge-price').textContent;
            document.getElementById('casePriceValue').textContent = 
                price.replace(' э', '');
            
            // Загрузка наград для кейса
            fetch(`/api/case-rewards/${currentCaseId}`)
                .then(response => response.json())
                .then(data => {
                    const rewardsGrid = document.getElementById('rewardsGrid');
                    rewardsGrid.innerHTML = '';
                    
                    data.forEach(reward => {
                        const rewardCard = document.createElement('div');
                        rewardCard.className = 'reward-card';
                        rewardCard.innerHTML = `
                            <img src="${reward.image}" alt="${reward.title}">
                            <h4>${reward.title}</h4>
                        `;
                        rewardsGrid.appendChild(rewardCard);
                    });
                })
                .catch(error => {
                    console.error('Ошибка загрузки наград:', error);
                    rewardsGrid.innerHTML = '<p>Не удалось загрузить награды</p>';
                });
            
            modal.style.display = 'block';
        });
    });
    
    // Закрытие модального окна
    closeBtn.addEventListener('click', function() {
        modal.style.display = 'none';
    });
    
    window.addEventListener('click', function(event) {
        if (event.target === modal) {
            modal.style.display = 'none';
        }
    });
    
    // Открытие кейса
    openCaseBtn.addEventListener('click', async() => {
        try {
            const response = await fetch(`/api/case-open/${currentCaseId}`, {
                method: 'POST',
            });

            if (response.ok) {
                const respData = await response.json();
                // Показываем выпавшую награду
                document.getElementById('rewardImage').src = respData.image;
                document.getElementById('rewardTitle').textContent = respData.title;
                
                // Скрываем основное окно, показываем награду
                document.querySelector('.modal-content').style.display = 'none';
                rewardPopup.style.display = 'flex';
            }
        } catch (error) {
            alert('Не удалось открыть кейс');
        }


        // fetch(`/api/case-open/${currentCaseId}`, {
        //     method: 'POST'
        // })
        // .then(response => response.json())
        // .then(data => {
        //     if (data.success) {
                
        //     } else {
        //         alert('Ошибка: ' + response.message);
        //     }
        // })
        // .catch(error => {
        //     console.error('Ошибка открытия кейса:', error);
        //     alert('Не удалось открыть кейс');
        // });
    });
    
    // Закрытие окна с наградой
    closeRewardBtn.addEventListener('click', function() {
        rewardPopup.style.display = 'none';
        modal.style.display = 'none';
        document.querySelector('.modal-content').style.display = 'block';
        location.reload(); // Обновляем страницу для обновления баланса
    });
});