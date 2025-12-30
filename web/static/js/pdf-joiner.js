document.addEventListener('alpine:init', () => {
    Alpine.data('pdfJoiner', () => ({
        // State
        files: [],
        joining: false,
        error: '',

        // Handle file selection
        handleFileSelect(event) {
            const newFiles = Array.from(event.target.files);

            for (const file of newFiles) {
                // Validate file type
                if (file.type !== 'application/pdf') {
                    this.error = 'Please upload valid PDF files.';
                    continue;
                }

                // Validate file size (individual file limit, say 50MB)
                if (file.size > 50 * 1024 * 1024) {
                    this.error = `File ${file.name} is too large. Maximum size is 50MB.`;
                    continue;
                }

                this.files.push({
                    id: Math.random().toString(36).substr(2, 9),
                    file: file,
                    name: file.name,
                    size: this.formatBytes(file.size)
                });
            }

            // Clear input value to allow selecting the same file again if needed
            event.target.value = '';
            this.error = '';
        },

        removeFile(index) {
            this.files.splice(index, 1);
        },

        moveUp(index) {
            if (index > 0) {
                const temp = this.files[index];
                this.files[index] = this.files[index - 1];
                this.files[index - 1] = temp;
            }
        },

        moveDown(index) {
            if (index < this.files.length - 1) {
                const temp = this.files[index];
                this.files[index] = this.files[index + 1];
                this.files[index + 1] = temp;
            }
        },

        // Join PDFs
        async join() {
            if (this.files.length < 2) {
                this.error = 'Please select at least 2 PDF files to join';
                return;
            }

            this.joining = true;
            this.error = '';

            try {
                const formData = new FormData();
                this.files.forEach(f => {
                    formData.append('pdf', f.file);
                });

                const response = await fetch('/api/tools/pdf/join', {
                    method: 'POST',
                    body: formData
                });

                if (!response.ok) {
                    const errorText = await response.text();
                    throw new Error(errorText || 'Join failed');
                }

                // Handle file download
                const blob = await response.blob();
                const url = window.URL.createObjectURL(blob);
                const a = document.createElement('a');
                a.href = url;
                a.download = 'joined.pdf';
                document.body.appendChild(a);
                a.click();
                window.URL.revokeObjectURL(url);
                document.body.removeChild(a);

            } catch (error) {
                console.error(error);
                this.error = error.message;
            } finally {
                this.joining = false;
            }
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
