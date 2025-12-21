document.addEventListener('alpine:init', () => {
    Alpine.data('imageConverter', () => ({
        // State
        file: null,
        fileName: '',
        fileSize: '',
        outputFormat: 'jpeg',
        quality: 85,
        converting: false,
        error: '',
        result: null,
        resultFormat: '',
        resultSize: '',
        resultBlob: null,

        // Handle file selection
        handleFileSelect(event) {
            const file = event.target.files[0];
            if (!file) return;

            // Validate file size (10MB limit)
            if (file.size > 10 * 1024 * 1024) {
                this.error = 'File too large. Maximum size is 10MB.';
                return;
            }

            this.file = file;
            this.fileName = file.name;
            this.fileSize = this.formatBytes(file.size);
            this.error = '';
            this.result = null;
        },

        // Convert the image
        async convert() {
            if (!this.file) {
                this.error = 'Please select an image first';
                return;
            }

            this.converting = true;
            this.error = '';
            this.result = null;

            try {
                // Create FormData and append our file
                const formData = new FormData();
                formData.append('image', this.file);
                formData.append('format', this.outputFormat);
                formData.append('quality', this.quality);

                const response = await fetch('/api/tools/image/convert', {
                    method: 'POST',
                    body: formData
                });

                if (!response.ok) {
                    const errorText = await response.text();
                    throw new Error(errorText || 'Conversion failed');
                }

                // Get the converted image as a blob
                const blob = await response.blob();
                this.resultBlob = blob;

                // Create a data URL for preview
                this.result = URL.createObjectURL(blob);
                this.resultFormat = this.outputFormat.toUpperCase();
                this.resultSize = this.formatBytes(blob.size);

            } catch (error) {
                this.error = error.message;
            } finally {
                this.converting = false;
            }
        },

        // Download the converted image
        download() {
            if (!this.resultBlob) return;

            const url = URL.createObjectURL(this.resultBlob);
            const a = document.createElement('a');
            a.href = url;
            
            // Generate filename
            const originalName = this.fileName.replace(/\.[^/.]+$/, '');
            a.download = `${originalName}_converted.${this.outputFormat}`;
            
            document.body.appendChild(a);
            a.click();
            document.body.removeChild(a);
            
            // Clean up the URL
            setTimeout(() => URL.revokeObjectURL(url), 100);
        },

        // Format bytes to human-readable string
        formatBytes(bytes) {
            if (bytes === 0) return '0 Bytes';
            const k = 1024;
            const sizes = ['Bytes', 'KB', 'MB'];
            const i = Math.floor(Math.log(bytes) / Math.log(k));
            return Math.round(bytes / Math.pow(k, i) * 100) / 100 + ' ' + sizes[i];
        }
    }));
});