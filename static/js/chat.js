// Чат
const chatButton = document.getElementById('chatButton');
const chatContainer = document.getElementById('chatContainer');
const closeChat = document.getElementById('closeChat');
        
if (chatButton && chatContainer) {
    chatButton.addEventListener('click', function() {
        chatContainer.classList.toggle('active');
    });
            
    if (closeChat) {
        closeChat.addEventListener('click', function() {
            chatContainer.classList.remove('active');
        });
    }
}