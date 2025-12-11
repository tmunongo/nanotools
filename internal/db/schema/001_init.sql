-- Audit log table
-- log every tool usage for analytics and debugging
CREATE TABLE IF NOT EXISTS audit_logs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    
    -- Which tool was used (e.g., 'json_formatter', 'image_converter')
    tool_name TEXT NOT NULL,
    
    -- Client information
    ip_address TEXT NOT NULL,
    user_agent TEXT,
    
    -- Request details
    input_size_bytes INTEGER,
    output_size_bytes INTEGER,
    
    -- Processing time in milliseconds
    processing_time_ms INTEGER,
    
    -- Success or error
    status TEXT NOT NULL, -- 'success' or 'error'
    error_message TEXT
);

-- Index for querying logs by tool
CREATE INDEX IF NOT EXISTS idx_audit_logs_tool ON audit_logs(tool_name);

-- Index for querying logs by time
CREATE INDEX IF NOT EXISTS idx_audit_logs_created ON audit_logs(created_at);