// Поиск
        // Функция для задержки запросов
        function debounce(func, wait) {
            let timeout;
            return function(...args) {
                clearTimeout(timeout);
                timeout = setTimeout(() => func.apply(this, args), wait);
            };
        }

        document.addEventListener('DOMContentLoaded', function() {
            const searchInput = document.getElementById('searchInput');
            const searchResults = document.getElementById('searchResults');
            
            const performSearch = debounce(async function() {
                const query = searchInput.value.trim();
                if (query.length < 2) {
                    searchResults.style.display = 'none';
                    return;
                }
                
                try {
                    const response = await fetch(`/api/search?q=${encodeURIComponent(query)}`);
                    const results = await response.json();
                    
                    searchResults.innerHTML = '';
                    if (results.length === 0) {
                        searchResults.style.display = 'none';
                        return;
                    }
                    
                    results.forEach(item => {
                        const resultElement = document.createElement('a');
                        resultElement.href = item.type === 'post' 
                            ? `/post/${item.id}` 
                            : `/user/${item.username}`;
                        resultElement.className = 'search-result-item';
                        
                        resultElement.innerHTML = `
                            ${item.title || item.display_name}
                            <span class="search-result-type">
                                ${item.type === 'post' ? 'Пост' : 'Пользователь'}
                            </span>
                        `;
                        searchResults.appendChild(resultElement);
                    });
                    
                    searchResults.style.display = 'block';
                } catch (error) {
                    console.error('Search error:', error);
                    searchResults.style.display = 'none';
                }
            }, 300);

            searchInput.addEventListener('input', performSearch);
            
            // Скрываем результаты при клике вне поля поиска
            document.addEventListener('click', function(e) {
                if (!searchInput.contains(e.target) && !searchResults.contains(e.target)) {
                    searchResults.style.display = 'none';
                }
            });
        });