document.addEventListener('alpine:init', () => {
    Alpine.data('qrGenerator', () => ({
        // State
        qrType: 'text',
        content: '',
        wifi: {
            ssid: '',
            password: '',
            encryption: 'WPA'
        },
        vcard: {
            name: '',
            phone: '',
            email: ''
        },
        size: 512,
        errorCorrection: 1, // Medium
        foregroundColor: '#000000',
        backgroundColor: '#ffffff',
        generating: false,
        error: '',
        qrCode: null,
        qrBlob: null,
        copied: false,

        // Generate QR code
        async generate() {
            this.generating = true;
            this.error = '';
            this.qrCode = null;
            this.copied = false;

            try {
                // Build form data based on type
                const formData = new FormData();
                formData.append('type', this.qrType);
                formData.append('size', this.size);
                formData.append('error_correction', this.errorCorrection);
                formData.append('foreground_color', this.foregroundColor);
                formData.append('background_color', this.backgroundColor);

                // Add type-specific data
                if (this.qrType === 'text') {
                    if (!this.content) {
                        throw new Error('Please enter content to encode');
                    }
                    formData.append('content', this.content);
                } else if (this.qrType === 'wifi') {
                    if (!this.wifi.ssid) {
                        throw new Error('Please enter Wi-Fi network name');
                    }
                    formData.append('ssid', this.wifi.ssid);
                    formData.append('password', this.wifi.password);
                    formData.append('encryption', this.wifi.encryption);
                } else if (this.qrType === 'vcard') {
                    if (!this.vcard.name) {
                        throw new Error('Please enter at least a name');
                    }
                    formData.append('name', this.vcard.name);
                    formData.append('phone', this.vcard.phone);
                    formData.append('email', this.vcard.email);
                }

                const response = await fetch('/api/tools/qr/generate', {
                    method: 'POST',
                    body: formData
                });

                if (!response.ok) {
                    const errorText = await response.text();
                    throw new Error(errorText || 'Failed to generate QR code');
                }

                // Get the QR code image as a blob
                const blob = await response.blob();
                this.qrBlob = blob;

                // Create a data URL for display
                this.qrCode = URL.createObjectURL(blob);

            } catch (error) {
                this.error = error.message;
            } finally {
                this.generating = false;
            }
        },

        // Download the QR code
        download() {
            if (!this.qrBlob) return;

            const url = URL.createObjectURL(this.qrBlob);
            const a = document.createElement('a');
            a.href = url;

            // Generate filename based on type
            let filename = 'qr-code';
            if (this.qrType === 'wifi') {
                filename = `wifi-${this.wifi.ssid}`;
            } else if (this.qrType === 'vcard') {
                filename = `contact-${this.vcard.name.replace(/\s+/g, '-')}`;
            }
            a.download = `${filename}.png`;

            document.body.appendChild(a);
            a.click();
            document.body.removeChild(a);

            setTimeout(() => URL.revokeObjectURL(url), 100);
        },

        // Copy image to clipboard
        async copyToClipboard() {
            if (!this.qrBlob) return;

            try {
                // Check if clipboard API supports images
                if (navigator.clipboard && navigator.clipboard.write) {
                    const item = new ClipboardItem({ 'image/png': this.qrBlob });
                    await navigator.clipboard.write([item]);

                    this.copied = true;
                    setTimeout(() => {
                        this.copied = false;
                    }, 2000);
                } else {
                    // Fallback: just notify user to right-click and save
                    alert('Your browser doesn\'t support copying images. Right-click the image and select "Copy Image" instead.');
                }
            } catch (error) {
                console.error('Failed to copy:', error);
                this.error = 'Failed to copy to clipboard';
            }
        }
    }));
});