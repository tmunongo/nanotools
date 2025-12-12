document.addEventListener('alpine:init', () => {
    Alpine.data('uuidGenerator', () => ({
        // State
        count: 5,
        uppercase: false,
        withHyphens: true,
        uuids: [],
        copied: false,

        // Generate UUIDs by calling our API
        async generate() {
            try {
                const params = new URLSearchParams({
                    count: this.count,
                    uppercase: this.uppercase,
                    hyphens: this.withHyphens
                });

                const response = await fetch(`/api/tools/uuid/generate?${params}`);

                if (!response.ok) {
                    throw new Error('Failed to generate UUIDs');
                }

                const data = await response.json();
                this.uuids = data.uuids;
                this.copied = false;
            } catch (error) {
                console.error('Error generating UUIDs:', error);
                alert('Failed to generate UUIDs. Please try again.');
            }
        },

        // Copy all UUIDs to clipboard
        async copyAll() {
            const text = this.uuids.join('\n');
            try {
                await navigator.clipboard.writeText(text);
                this.copied = true;

                // Reset the "Copied!" text after 2 seconds
                setTimeout(() => {
                    this.copied = false;
                }, 2000);
            } catch (error) {
                console.error('Failed to copy:', error);
                alert('Failed to copy to clipboard');
            }
        },

        // Copy a single UUID
        async copySingle(uuid) {
            try {
                await navigator.clipboard.writeText(uuid);
                // You could add visual feedback here
            } catch (error) {
                console.error('Failed to copy:', error);
            }
        }
    }));
});