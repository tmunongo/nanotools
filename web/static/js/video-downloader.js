function videoDownloader() {
    return {
        url: '',
        isYouTube: false,
        loading: false,
        downloading: false,
        error: '',
        videoInfo: null,
        selectedQuality: '720p',

        checkPlatform() {
            this.isYouTube = this.url.includes('youtube.com') || this.url.includes('youtu.be');
        },

        async getVideoInfo() {
            if (!this.url) {
                this.error = 'Please enter a video URL';
                return;
            }

            this.loading = true;
            this.error = '';
            this.videoInfo = null;

            try {
                const formData = new FormData();
                formData.append('url', this.url);

                const response = await fetch('/api/tools/video/info', {
                    method: 'POST',
                    body: formData
                });

                if (!response.ok) {
                    const errorText = await response.text();
                    throw new Error(errorText || 'Failed to get video info');
                }

                this.videoInfo = await response.json();
            } catch (error) {
                this.error = error.message;
            } finally {
                this.loading = false;
            }
        },

        async downloadVideo() {
            this.downloading = true;
            this.error = '';

            try {
                const formData = new FormData();
                formData.append('url', this.url);
                formData.append('quality', this.selectedQuality);
                formData.append('format', 'mp4');

                const response = await fetch('/api/tools/video/download', {
                    method: 'POST',
                    body: formData
                });

                if (!response.ok) {
                    const errorText = await response.text();
                    throw new Error(errorText || 'Download failed');
                }

                const blob = await response.blob();
                const url = URL.createObjectURL(blob);
                const a = document.createElement('a');
                a.href = url;

                const disposition = response.headers.get('Content-Disposition');
                let filename = 'video.mp4';
                if (disposition) {
                    const matches = /filename="?([^"]+)"?/.exec(disposition);
                    if (matches) filename = matches[1];
                }

                a.download = filename;
                document.body.appendChild(a);
                a.click();
                document.body.removeChild(a);
                URL.revokeObjectURL(url);

            } catch (error) {
                this.error = error.message;
            } finally {
                this.downloading = false;
            }
        },

        formatDuration(seconds) {
            if (!seconds) return 'Unknown';
            const hours = Math.floor(seconds / 3600);
            const minutes = Math.floor((seconds % 3600) / 60);
            const secs = seconds % 60;

            if (hours > 0) {
                return `${hours}:${minutes.toString().padStart(2, '0')}:${secs.toString().padStart(2, '0')}`;
            }
            return `${minutes}:${secs.toString().padStart(2, '0')}`;
        },

        formatBytes(bytes) {
            if (bytes === 0 || !bytes) return 'Unknown';
            const k = 1024;
            const sizes = ['Bytes', 'KB', 'MB', 'GB'];
            const i = Math.floor(Math.log(bytes) / Math.log(k));
            return Math.round(bytes / Math.pow(k, i) * 100) / 100 + ' ' + sizes[i];
        }
    };
}