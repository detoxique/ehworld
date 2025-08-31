// Управление чатом
class EhchoChat {
    constructor() {
        this.chatButton = document.getElementById('chatButton');
        this.chatContainer = document.getElementById('chatContainer');
        this.closeChat = document.getElementById('closeChat');
        this.messagesContainer = document.getElementById('messagesContainer');
        this.messageInput = document.getElementById('messageInput');
        this.fileInput = document.getElementById('fileInput');
        this.sendMessageBtn = document.getElementById('sendMessageBtn');
        this.filePreview = document.getElementById('filePreview');
        
        this.attachedFiles = [];
        this.isOpen = false;
        this.isAutoScrollEnabled = false;
        
        this.init();
    }
    
    init() {
        // Обработчики событий
        this.chatButton.addEventListener('click', () => this.toggleChat());
        this.closeChat.addEventListener('click', () => this.hideChat());
        this.sendMessageBtn.addEventListener('click', () => this.sendMessage());
        this.messageInput.addEventListener('keypress', (e) => {
            if (e.key === 'Enter') this.sendMessage();
        });
        this.fileInput.addEventListener('change', (e) => this.handleFileSelect(e));
        
        // Загружаем историю сообщений
        this.loadMessageHistory();
        
        // Интервал для обновления сообщений
        setInterval(() => this.loadMessageHistory(), 5000);
    }
    
    toggleChat() {
        if (this.isOpen) {
            this.hideChat();
        } else {
            this.showChat();
        }
    }
    
    showChat() {
        this.chatContainer.style.display = 'flex';
        this.isOpen = true;
        // Фокусируемся на поле ввода при открытии
        setTimeout(() => this.messageInput.focus(), 100);
    }
    
    hideChat() {
        this.chatContainer.style.display = 'none';
        this.isOpen = false;
    }
    
    async loadMessageHistory() {
        try {
            const response = await fetch('/api/chat/messages?limit=50');
            const messages = await response.json();
            
            this.messagesContainer.innerHTML = '';
            messages.forEach(message => {
                this.appendMessage(message);
            });
            
            // Прокручиваем к последнему сообщению
            if (this.isAutoScrollEnabled) {
                this.scrollToBottom();
            }
            
        } catch (error) {
            console.error('Ошибка при загрузке истории чата:', error);
        }
    }
    
    async sendMessage() {
        const text = this.messageInput.value.trim();
        
        // Если нет текста и нет прикрепленных файлов - ничего не делаем
        if (!text && this.attachedFiles.length === 0) return;
        
        // Создаем FormData для отправки
        const formData = new FormData();
        formData.append('message', text);
        
        // Добавляем файлы
        this.attachedFiles.forEach(file => {
            formData.append('files', file);
        });
        
        try {
            const response = await fetch('/api/chat/send', {
                method: 'POST',
                body: formData
            });
            
            if (response.ok) {
                // Очищаем поле ввода и прикрепленные файлы
                this.messageInput.value = '';
                this.clearAttachedFiles();
                
                // Перезагружаем историю сообщений
                this.loadMessageHistory();
            } else {
                console.error('Ошибка при отправке сообщения');
            }
        } catch (error) {
            console.error('Ошибка при отправке сообщения:', error);
        }
    }
    
    handleFileSelect(event) {
        const files = event.target.files;
        if (!files.length) return;
        
        for (let i = 0; i < files.length; i++) {
            const file = files[i];
            
            // Проверяем тип файла (только изображения и аудио)
            if (!file.type.match('image.*') && !file.type.match('audio.*')) {
                alert('Можно загружать только изображения и аудио файлы');
                continue;
            }
            
            // Проверяем размер файла (максимум 10MB)
            if (file.size > 10 * 1024 * 1024) {
                alert('Файл слишком большой. Максимальный размер: 10MB');
                continue;
            }
            
            this.attachedFiles.push(file);
            this.displayFilePreview(file);
        }
        
        // Сбрасываем значение input, чтобы можно было выбрать тот же файл снова
        this.fileInput.value = '';
    }
    
    displayFilePreview(file) {
        const previewItem = document.createElement('div');
        previewItem.className = 'preview-item';
        
        if (file.type.match('image.*')) {
            const reader = new FileReader();
            reader.onload = (e) => {
                const img = document.createElement('img');
                img.src = e.target.result;
                previewItem.appendChild(img);
                
                const removeBtn = document.createElement('button');
                removeBtn.className = 'remove-preview';
                removeBtn.innerHTML = '×';
                removeBtn.onclick = () => this.removeAttachedFile(file, previewItem);
                previewItem.appendChild(removeBtn);
            };
            reader.readAsDataURL(file);
        } else if (file.type.match('audio.*')) {
            const audio = document.createElement('audio');
            audio.controls = true;
            audio.src = URL.createObjectURL(file);
            previewItem.appendChild(audio);
            
            const removeBtn = document.createElement('button');
            removeBtn.className = 'remove-preview';
            removeBtn.innerHTML = '×';
            removeBtn.onclick = () => this.removeAttachedFile(file, previewItem);
            previewItem.appendChild(removeBtn);
        }
        
        this.filePreview.appendChild(previewItem);
    }
    
    removeAttachedFile(file, previewElement) {
        this.attachedFiles = this.attachedFiles.filter(f => f !== file);
        this.filePreview.removeChild(previewElement);
        URL.revokeObjectURL(previewElement.querySelector('audio')?.src);
    }
    
    clearAttachedFiles() {
        this.attachedFiles = [];
        this.filePreview.innerHTML = '';
    }
    
    appendMessage(message) {
        const messageElement = document.createElement('div');
        messageElement.className = 'message';
        
        const header = document.createElement('div');
        header.className = 'message-header';

        const badge = document.createElement('img');
        badge.className = 'message-badge';
        badge.src = message.badge_url;
        
        const userSpan = document.createElement('span');
        userSpan.className = 'message-user';
        userSpan.textContent = message.display_name || 'Аноним';
        userSpan.onclick = () => {
            window.open(`/user/${message.display_name}`, '_blank');
        };
        
        header.appendChild(badge);
        header.appendChild(userSpan);
        
        const content = document.createElement('div');
        content.className = 'message-content';
        content.innerHTML = DOMPurify.sanitize(message.content);
        
        messageElement.appendChild(header);
        messageElement.appendChild(content);
        
        // Добавляем медиафайлы, если есть
        if (message.files_urls && message.files_urls.length > 0) {
            message.files_urls.forEach(fileUrl => {
                // Определяем тип файла по расширению
                const extension = fileUrl.split('.').pop().toLowerCase();
                
                if (['png', 'gif', 'jpg', 'jpeg', 'bmp', 'hevc', 'webp'].includes(extension)) {
                    const img = document.createElement('img');
                    img.className = 'message-image';
                    img.src = fileUrl;
                    img.alt = 'Прикрепленное изображение';
                    messageElement.appendChild(img);
                } else if (['ogg', 'mp3', 'wav', 'm4a', 'aac'].includes(extension)) {
                    const audio = document.createElement('audio');
                    audio.className = 'message-audio';
                    audio.controls = true;
                    audio.src = fileUrl;
                    messageElement.appendChild(audio);
                } else {
                    // Для других типов файлов создаем ссылку для скачивания
                    const link = document.createElement('a');
                    link.className = 'message-file';
                    link.href = fileUrl;
                    link.target = '_blank';
                    link.textContent = 'Скачать файл';
                    messageElement.appendChild(link);
                }
            });
        }
        
        this.messagesContainer.appendChild(messageElement);
    }
    
    scrollToBottom() {
        this.messagesContainer.scrollTop = this.messagesContainer.scrollHeight;
    }
}

// Инициализация чата при загрузке страницы
document.addEventListener('DOMContentLoaded', function() {
    new EhchoChat();
});