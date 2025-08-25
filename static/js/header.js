const hamburger = document.getElementById('hamburger');
const navLinks = document.getElementById('navLinks');
        
if (hamburger && navLinks) {
    hamburger.addEventListener('click', function() {
        hamburger.classList.toggle('active');
        navLinks.classList.toggle('active');
    });
}