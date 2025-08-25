// Функция для загрузки информации о каналах
async function loadLiveChannels() {
    const channelsContainer = document.getElementById('liveChannels');
    
    try {
        // Получаем данные с сервера
        const sortOrder = document.getElementById('viewersSort').value;
        const response = await fetch(`/api/live-channels?sort=${sortOrder}`);
        const channels = await response.json();
        
        // Отображаем каналы
        if (channels.length > 0) {
            displayChannels(channels);
        } else {
            channelsContainer.innerHTML = '<div class="no-live-channels">Сейчас никто не в эфире</div>';
        }
    } catch (error) {
        console.error('Ошибка при загрузке каналов:', error);
        channelsContainer.innerHTML = '<div class="channels-error">Ошибка при загрузке каналов</div>';
    }
}

        // Функция для отображения каналов
        function displayChannels(channels) {
            const channelsContainer = document.getElementById('liveChannels');
            channelsContainer.innerHTML = '';
            
            channels.forEach(channel => {
                const isLive = channel.is_live;
                const channelElement = document.createElement('div');
                channelElement.className = `channel-item ${isLive ? '' : 'offline'}`;
                channelElement.onclick = () => window.open(`https://twitch.tv/${channel.user_login}`, '_blank');
                
                channelElement.innerHTML = `
                    <img src="${channel.profile_image_url}" alt="${channel.user_name}" class="channel-avatar">
                    <div class="channel-info">
                        <div class="channel-name">${channel.user_name}</div>
                        <div class="channel-category">${isLive ? (channel.game_name || 'Just Chatting') : 'Не в сети'}</div>
                    </div>
                    ${isLive ? `
                        <div class="channel-viewers">
                            <span class="viewer-dot"></span>
                            <span class="viewer-count">${formatViewers(channel.viewer_count)}</span>
                        </div>
                    ` : ''}
                `;
                
                channelsContainer.appendChild(channelElement);
            });
        }

        // Форматирование числа зрителей
        function formatViewers(count) {
            if (count >= 1000000) {
                return (count / 1000000).toFixed(1) + 'M';
            } else if (count >= 1000) {
                return (count / 1000).toFixed(1) + 'K';
            }
            return count;
        }