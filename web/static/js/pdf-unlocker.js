document.addEventListener('alpine:init', () => {
    Alpine.data('pdfUnlocker', () => ({
        // State
        file: null,
        fileName: '',
        fileSize: '',
        password: '',
        showPassword: false,
        unlocking: false,
        error: '',
        success: false,

        handleFileSelect(event) {
            const selected = event.target.files[0];
            this.error = '';
            this.success = false;

            if (!selected) {
                this.file = null;
                return;
            }

            if (selected.type !== 'application/pdf' && !selected.name.toLowerCase().endsWith('.pdf')) {
                this.error = 'Please upload a valid PDF file.';
                event.target.value = '';
                this.file = null;
                return;
            }

            if (selected.size > 50 * 1024 * 1024) {
                this.error = 'File is too large. Maximum size is 50 MB.';
                event.target.value = '';
                this.file = null;
                return;
            }

            this.file = selected;
            this.fileName = selected.name;
            this.fileSize = this.formatBytes(selected.size);
        },

        async unlock() {
            if (!this.file) {
                this.error = 'Please select a PDF file.';
                return;
            }
            if (!this.password) {
                this.error = 'Please enter the PDF password.';
                return;
            }

            this.unlocking = true;
            this.error = '';
            this.success = false;

            try {
                const formData = new FormData();
                formData.append('pdf', this.file);
                formData.append('password', this.password);

                const response = await fetch('/api/tools/pdf/unlock', {
                    method: 'POST',
                    body: formData,
                });

                if (!response.ok) {
                    const errorText = await response.text();
                    throw new Error(errorText || 'Failed to unlock the PDF.');
                }

                // Trigger download
                const blob = await response.blob();
                const url = window.URL.createObjectURL(blob);
                const a = document.createElement('a');
                a.href = url;
                a.download = 'unlocked-' + this.file.name;
                document.body.appendChild(a);
                a.click();
                window.URL.revokeObjectURL(url);
                document.body.removeChild(a);

                this.success = true;

            } catch (err) {
                console.error(err);
                this.error = err.message;
            } finally {
                this.unlocking = false;
                // Clear password from memory immediately after the request
                this.password = '';
            }
        },

        formatBytes(bytes) {
            if (bytes === 0) return '0 Bytes';
            const k = 1024;
            const sizes = ['Bytes', 'KB', 'MB', 'GB'];
            const i = Math.floor(Math.log(bytes) / Math.log(k));
            return Math.round(bytes / Math.pow(k, i) * 100) / 100 + ' ' + sizes[i];
        },
    }));
});

