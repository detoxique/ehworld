class VideoPlayer {
    constructor(container) {
        // Основные элементы
        this.container = container;
        this.video = container.querySelector('video');
        this.playBtn = container.querySelector('.play-pause-btn');
        this.videoPlayBtn = container.querySelector('.video-play-btn');
        this.progressContainer = container.querySelector('.progress-container');
        this.progressBar = container.querySelector('.progress-bar');
        this.volumeBtn = container.querySelector('.volume-btn');
        this.volumeSlider = container.querySelector('.volume-slider');
        this.volumeLevel = container.querySelector('.volume-level');
        this.currentTimeEl = container.querySelector('.current-time');
        this.durationEl = container.querySelector('.duration');
        this.fullscreenBtn = container.querySelector('.fullscreen-btn');
        this.loadingIndicator = container.querySelector('.video-loading');
        
        // Состояние
        this.isDraggingProgress = false;
        this.isDraggingVolume = false;
        this.controlsTimeout = null;
        this.wasPlayingBeforeDrag = false;
        
        // Инициализация
        this.init();
    }
    
    init() {
        // Обработчики событий
        this.video.addEventListener('click', () => this.togglePlay());
        this.video.addEventListener('play', () => this.updatePlayState());
        this.video.addEventListener('pause', () => this.updatePlayState());
        this.video.addEventListener('timeupdate', () => this.updateProgress());
        this.video.addEventListener('waiting', () => this.showLoading());
        this.video.addEventListener('playing', () => this.hideLoading());
        this.video.addEventListener('loadedmetadata', () => this.setDuration());
        this.video.addEventListener('ended', () => this.handleVideoEnd());
        
        this.playBtn.addEventListener('click', () => this.togglePlay());
        this.videoPlayBtn.addEventListener('click', () => this.togglePlay());
        
        this.progressContainer.addEventListener('mousedown', (e) => this.startProgressDrag(e));
        document.addEventListener('mousemove', (e) => this.handleProgressDrag(e));
        document.addEventListener('mouseup', () => this.stopProgressDrag());
        
        this.volumeBtn.addEventListener('click', () => this.toggleMute());
        this.volumeSlider.addEventListener('mousedown', (e) => this.startVolumeDrag(e));
        document.addEventListener('mousemove', (e) => this.handleVolumeDrag(e));
        document.addEventListener('mouseup', () => this.stopVolumeDrag());
        
        this.fullscreenBtn.addEventListener('click', () => this.toggleFullscreen());
        
        this.container.addEventListener('mousemove', () => this.showControls());
        this.container.addEventListener('mouseleave', () => this.hideControls());
        
        // Инициализация громкости
        this.video.volume = 0.7;
        this.updateVolumeUI();
        
        // Установка длительности видео
        this.setDuration();
    }
    
    togglePlay() {
        if (this.video.paused) {
            this.video.play();
            this.container.classList.add('playing');
        } else {
            this.video.pause();
            this.container.classList.remove('playing');
        }
    }
    
    updatePlayState() {
        const icon = this.playBtn.querySelector('svg');
        if (this.video.paused) {
            icon.innerHTML = '<path d="M8 5v14l11-7z"/>';
        } else {
            icon.innerHTML = '<path d="M6 19h4V5H6v14zm8-14v14h4V5h-4z"/>';
        }
    }
    
    showLoading() {
        this.loadingIndicator.classList.add('visible');
    }
    
    hideLoading() {
        this.loadingIndicator.classList.remove('visible');
    }
    
    setDuration() {
        const duration = this.video.duration;
        if (!isNaN(duration)) {
            this.durationEl.textContent = this.formatTime(duration);
        }
    }
    
    updateProgress() {
        if (this.isDraggingProgress) return;
        
        const percent = (this.video.currentTime / this.video.duration) * 100;
        this.progressBar.style.width = `${percent}%`;
        
        this.currentTimeEl.textContent = this.formatTime(this.video.currentTime);
    }
    
    startProgressDrag(e) {
        this.isDraggingProgress = true;
        this.wasPlayingBeforeDrag = !this.video.paused;
        if (this.wasPlayingBeforeDrag) this.video.pause();
        
        this.progressContainer.classList.add('dragging');
        this.updateProgressPosition(e);
    }
    
    handleProgressDrag(e) {
        if (!this.isDraggingProgress) return;
        this.updateProgressPosition(e);
    }
    
    stopProgressDrag() {
        if (!this.isDraggingProgress) return;
        
        this.isDraggingProgress = false;
        this.progressContainer.classList.remove('dragging');
        
        if (this.wasPlayingBeforeDrag) this.video.play();
    }
    
    updateProgressPosition(e) {
        const rect = this.progressContainer.getBoundingClientRect();
        const percent = Math.min(Math.max((e.clientX - rect.left) / rect.width, 0), 1);
        this.progressBar.style.width = `${percent * 100}%`;
        
        this.video.currentTime = percent * this.video.duration;
        this.currentTimeEl.textContent = this.formatTime(this.video.currentTime);
    }
    
    toggleMute() {
        this.video.muted = !this.video.muted;
        this.updateVolumeUI();
    }
    
    startVolumeDrag(e) {
        this.isDraggingVolume = true;
        this.updateVolumePosition(e);
    }
    
    handleVolumeDrag(e) {
        if (!this.isDraggingVolume) return;
        this.updateVolumePosition(e);
    }
    
    stopVolumeDrag() {
        this.isDraggingVolume = false;
    }
    
    updateVolumePosition(e) {
        const rect = this.volumeSlider.getBoundingClientRect();
        const percent = Math.min(Math.max((e.clientX - rect.left) / rect.width, 0), 1);
        this.volumeLevel.style.width = `${percent * 100}%`;
        
        this.video.volume = percent;
        this.video.muted = percent === 0;
        this.updateVolumeUI();
    }
    
    updateVolumeUI() {
        const volume = this.video.muted ? 0 : this.video.volume;
        this.volumeLevel.style.width = `${volume * 100}%`;
        
        const icon = this.volumeBtn.querySelector('svg');
        if (volume === 0) {
            icon.innerHTML = '<path d="M3.63 3.63a.996.996 0 0 0 0 1.41L7.29 9H6c-1.1 0-2 .9-2 2v2c0 1.1.9 2 2 2h3.29l3.66 3.66c.39.39 1.02.39 1.41 0 .39-.39.39-1.02 0-1.41L5.05 3.63c-.39-.39-1.02-.39-1.42 0zm13.74 2.74c-.25-.33-.7-.4-1.03-.15-.33.25-.4.7-.15 1.03.25.33.7.4 1.03.15.33-.25.4-.7.15-1.03zM19.03 7c.27.3.24.78-.06 1.06-.3.27-.78.24-1.06-.06-.27-.3-.24-.78.06-1.06.3-.27.78-.24 1.06.06zm0 4c.27.3.24.78-.06 1.06-.3.27-.78.24-1.06-.06-.27-.3-.24-.78.06-1.06.3-.27.78-.24 1.06.06z"/>';
        } else if (volume < 0.5) {
            icon.innerHTML = '<path d="M18.5 12c0-1.77-1.02-3.29-2.5-4.03v8.05c1.48-.73 2.5-2.25 2.5-4.02zM5 9v6h4l5 5V4L9 9H5z"/>';
        } else {
            icon.innerHTML = '<path d="M3 9v6h4l5 5V4L7 9H3zm13.5 3c0-1.77-1.02-3.29-2.5-4.03v8.05c1.48-.73 2.5-2.25 2.5-4.02zM14 3.23v2.06c2.89.86 5 3.54 5 6.71s-2.11 5.85-5 6.71v2.06c4.01-.91 7-4.49 7-8.77s-2.99-7.86-7-8.77z"/>';
        }
    }
    
    toggleFullscreen() {
        if (!document.fullscreenElement) {
            if (this.container.requestFullscreen) {
                this.container.requestFullscreen();
            } else if (this.container.mozRequestFullScreen) { /* Firefox */
                this.container.mozRequestFullScreen();
            } else if (this.container.webkitRequestFullscreen) { /* Chrome, Safari and Opera */
                this.container.webkitRequestFullscreen();
            } else if (this.container.msRequestFullscreen) { /* IE/Edge */
                this.container.msRequestFullscreen();
            }
            
            // Обновляем иконку
            this.fullscreenBtn.querySelector('.fullscreen-open').style.display = 'none';
            this.fullscreenBtn.querySelector('.fullscreen-close').style.display = 'block';
        } else {
            if (document.exitFullscreen) {
                document.exitFullscreen();
            } else if (document.mozCancelFullScreen) { /* Firefox */
                document.mozCancelFullScreen();
            } else if (document.webkitExitFullscreen) { /* Chrome, Safari and Opera */
                document.webkitExitFullscreen();
            } else if (document.msExitFullscreen) { /* IE/Edge */
                document.msExitFullscreen();
            }
            
            // Обновляем иконку
            this.fullscreenBtn.querySelector('.fullscreen-open').style.display = 'block';
            this.fullscreenBtn.querySelector('.fullscreen-close').style.display = 'none';
        }
    }
    
    showControls() {
        clearTimeout(this.controlsTimeout);
        this.container.querySelector('.video-controls').style.opacity = '1';
        this.controlsTimeout = setTimeout(() => {
            if (!this.video.paused) {
                this.container.querySelector('.video-controls').style.opacity = '0';
            }
        }, 3000);
    }
    
    hideControls() {
        clearTimeout(this.controlsTimeout);
        if (!this.video.paused) {
            this.container.querySelector('.video-controls').style.opacity = '0';
        }
    }
    
    handleVideoEnd() {
        this.container.classList.remove('playing');
        this.video.currentTime = 0;
        this.updateProgress();
    }
    
    formatTime(seconds) {
        const minutes = Math.floor(seconds / 60);
        seconds = Math.floor(seconds % 60);
        return `${minutes}:${seconds < 10 ? '0' : ''}${seconds}`;
    }
}

//Инициализация плеера
document.addEventListener('DOMContentLoaded', () => {
    const videoContainers = document.querySelectorAll('.video-container');
    videoContainers.forEach(container => {
        console.log('new video container');
        new VideoPlayer(container);
    });
});