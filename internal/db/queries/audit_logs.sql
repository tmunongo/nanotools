-- name: CreateAuditLog :one
INSERT INTO audit_logs (
    tool_name,
    ip_address,
    user_agent,
    input_size_bytes,
    output_size_bytes,
    processing_time_ms,
    status,
    error_message
) VALUES (
    ?, ?, ?, ?, ?, ?, ?, ?
)
RETURNING *;

-- name: GetRecentLogs :many
SELECT * FROM audit_logs
ORDER BY created_at DESC
LIMIT ?;

-- name: GetLogsByTool :many
SELECT * FROM audit_logs
WHERE tool_name = ?
ORDER BY created_at DESC
LIMIT ?;

-- name: GetToolStats :one
SELECT 
    COUNT(*) as total_uses,
    SUM(CASE WHEN status = 'success' THEN 1 ELSE 0 END) as successful_uses,
    AVG(processing_time_ms) as avg_processing_time_ms,
    SUM(input_size_bytes) as total_input_bytes,
    SUM(output_size_bytes) as total_output_bytes
FROM audit_logs
WHERE tool_name = ?;

-- Most downloaded platforms
SELECT 
    CASE 
        WHEN tool_name LIKE '%youtube%' THEN 'YouTube'
        ELSE 'Other'
    END as platform,
    COUNT(*) as downloads,
    AVG(processing_time_ms) as avg_time
FROM audit_logs
WHERE tool_name = 'video_downloader'
GROUP BY platform;

-- Failed downloads
SELECT error_message, COUNT(*) 
FROM audit_logs
WHERE tool_name = 'video_downloader' AND status = 'error'
GROUP BY error_message
ORDER BY COUNT(*) DESC;