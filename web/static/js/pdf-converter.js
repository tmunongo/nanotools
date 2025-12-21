document.addEventListener('alpine:init', () => {
    Alpine.data('pdfConverter', () => ({
        // State
        file: null,
        fileName: '',
        fileSize: '',
        outputFormat: 'jpeg',
        dpi: 150,
        quality: 85,
        converting: false,
        error: '',
        results: [],

        // Handle file selection
        handleFileSelect(event) {
            const file = event.target.files[0];
            if (!file) return;

            // Validate file type
            if (file.type !== 'application/pdf') {
                this.error = 'Please upload a valid PDF file.';
                return;
            }

            // Validate file size (50MB limit)
            if (file.size > 50 * 1024 * 1024) {
                this.error = 'File too large. Maximum size is 50MB.';
                return;
            }

            this.file = file;
            this.fileName = file.name;
            this.fileSize = this.formatBytes(file.size);
            this.error = '';
            this.results = [];
        },

        // Convert the PDF
        async convert() {
            if (!this.file) {
                this.error = 'Please select a PDF file first';
                return;
            }

            this.converting = true;
            this.error = '';
            this.results = [];

            try {
                const formData = new FormData();
                formData.append('pdf', this.file);
                formData.append('format', this.outputFormat);
                formData.append('dpi', this.dpi);
                formData.append('quality', this.quality);

                const response = await fetch('/api/tools/pdf/to-images', {
                    method: 'POST',
                    body: formData
                });

                if (!response.ok) {
                    const errorText = await response.text();
                    throw new Error(errorText || 'Conversion failed');
                }

                const data = await response.json();

                if (data.success && data.images) {
                    this.results = data.images;
                } else {
                    throw new Error('Invalid response from server');
                }

            } catch (error) {
                console.error(error);
                this.error = error.message;
            } finally {
                this.converting = false;
            }
        },

        // Download a single image
        downloadImage(img) {
            const link = document.createElement('a');
            link.href = 'data:image/' + img.Format + ';base64,' + img.ImageData;
            const originalName = this.fileName.replace(/\.pdf$/i, '');
            link.download = `${originalName}_page_${img.PageNumber}.${img.Format}`;
            document.body.appendChild(link);
            link.click();
            document.body.removeChild(link);
        },

        // Format bytes to human-readable string
        formatBytes(bytes) {
            if (bytes === 0) return '0 Bytes';
            const k = 1024;
            const sizes = ['Bytes', 'KB', 'MB', 'GB'];
            const i = Math.floor(Math.log(bytes) / Math.log(k));
            return Math.round(bytes / Math.pow(k, i) * 100) / 100 + ' ' + sizes[i];
        }
    }));
});
