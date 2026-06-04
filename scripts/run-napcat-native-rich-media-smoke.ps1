param(
    [string]$WebSocketUrl = "ws://127.0.0.1:3001",
    [string]$AccessToken = "NcatBot",
    [ValidateSet("group", "private")][string]$ConversationType = "group",
    [string]$ChatId = "",
    [string]$GroupId = "27234224"
)

$ErrorActionPreference = "Stop"

function Invoke-OneBotAction {
    param(
        [Parameter(Mandatory = $true)][string]$Uri,
        [Parameter(Mandatory = $true)][string]$Token,
        [Parameter(Mandatory = $true)][string]$Action,
        [hashtable]$Params = @{}
    )

    $ws = [System.Net.WebSockets.ClientWebSocket]::new()
    try {
        $ws.Options.SetRequestHeader("Authorization", "Bearer $Token")
        [void]$ws.ConnectAsync([Uri]$Uri, [Threading.CancellationToken]::None).GetAwaiter().GetResult()

        $echo = [Guid]::NewGuid().ToString()
        $payload = @{
            action = $Action
            params = $Params
            echo   = $echo
        } | ConvertTo-Json -Depth 32 -Compress
        $bytes = [Text.Encoding]::UTF8.GetBytes($payload)
        $segment = [ArraySegment[byte]]::new($bytes)
        [void]$ws.SendAsync(
            $segment,
            [System.Net.WebSockets.WebSocketMessageType]::Text,
            $true,
            [Threading.CancellationToken]::None
        ).GetAwaiter().GetResult()

        $buffer = New-Object byte[] 65536
        $segmentIn = [ArraySegment[byte]]::new($buffer)
        while ($true) {
            $stream = New-Object System.IO.MemoryStream
            try {
                do {
                    $result = $ws.ReceiveAsync($segmentIn, [Threading.CancellationToken]::None).GetAwaiter().GetResult()
                    if ($result.MessageType -eq [System.Net.WebSockets.WebSocketMessageType]::Close) {
                        throw "websocket closed by peer"
                    }
                    if ($result.Count -gt 0) {
                        $stream.Write($buffer, 0, $result.Count)
                    }
                } while (-not $result.EndOfMessage)

                $stream.Position = 0
                $reader = New-Object System.IO.StreamReader($stream, [Text.Encoding]::UTF8)
                $text = $reader.ReadToEnd()
                $json = $text | ConvertFrom-Json
                if ($json.echo -eq $echo) {
                    return $json
                }
            }
            finally {
                $stream.Dispose()
            }
        }
    }
    finally {
        try {
            if ($ws.State -eq [System.Net.WebSockets.WebSocketState]::Open) {
                [void]$ws.CloseAsync(
                    [System.Net.WebSockets.WebSocketCloseStatus]::NormalClosure,
                    "done",
                    [Threading.CancellationToken]::None
                ).GetAwaiter().GetResult()
            }
        }
        catch {
        }
        $ws.Dispose()
    }
}

function Upload-FileStreamLikeAkashic {
    param(
        [Parameter(Mandatory = $true)][string]$Uri,
        [Parameter(Mandatory = $true)][string]$Token,
        [Parameter(Mandatory = $true)][string]$SourcePath
    )

    $raw = [IO.File]::ReadAllBytes($SourcePath)
    $streamId = "native-" + [Guid]::NewGuid().ToString("N")
    $sha = [System.BitConverter]::ToString(
        ([System.Security.Cryptography.SHA256]::Create().ComputeHash($raw))
    ).Replace("-", "").ToLowerInvariant()
    $chunkSize = 64KB
    $totalChunks = [Math]::Ceiling($raw.Length / [double]$chunkSize)
    if ($totalChunks -lt 1) {
        $totalChunks = 1
    }

    for ($i = 0; $i -lt $totalChunks; $i++) {
        $start = $i * $chunkSize
        $length = [Math]::Min($chunkSize, $raw.Length - $start)
        $chunk = New-Object byte[] $length
        [Array]::Copy($raw, $start, $chunk, 0, $length)
        $response = Invoke-OneBotAction -Uri $Uri -Token $Token -Action "upload_file_stream" -Params @{
            stream_id       = $streamId
            chunk_data      = [Convert]::ToBase64String($chunk)
            chunk_index     = $i
            total_chunks    = $totalChunks
            file_size       = $raw.Length
            expected_sha256 = $sha
            filename        = [IO.Path]::GetFileName($SourcePath)
            file_retention  = 30000
        }
        if ($response.retcode -ne 0) {
            return $response
        }
    }

    return Invoke-OneBotAction -Uri $Uri -Token $Token -Action "upload_file_stream" -Params @{
        stream_id   = $streamId
        is_complete = $true
    }
}

$tempDir = Join-Path $env:TEMP ("akashic-native-napcat-rich-media-" + [guid]::NewGuid().ToString("N"))
New-Item -ItemType Directory -Force -Path $tempDir | Out-Null
$imagePath = Join-Path $tempDir "native-smoke.png"
$filePath = Join-Path $tempDir "native-smoke.txt"
[IO.File]::WriteAllBytes(
    $imagePath,
    [Convert]::FromBase64String("iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAwMCAO2+nWQAAAAASUVORK5CYII=")
)
Set-Content -LiteralPath $filePath -Value ("native napcat smoke {0}" -f (Get-Date).ToString("s")) -Encoding UTF8

$uri = $WebSocketUrl
if ($uri -notmatch "access_token=" -and -not [string]::IsNullOrWhiteSpace($AccessToken)) {
    if ($uri.Contains("?")) {
        $uri = "{0}&access_token={1}" -f $uri, $AccessToken
    }
    else {
        $uri = "{0}?access_token={1}" -f $uri, $AccessToken
    }
}

$targetChatId = if (-not [string]::IsNullOrWhiteSpace($ChatId)) {
    $ChatId
} else {
    $GroupId
}
if ([string]::IsNullOrWhiteSpace($targetChatId)) {
    throw "ChatId or GroupId is required"
}

$login = Invoke-OneBotAction -Uri $uri -Token $AccessToken -Action "get_login_info"
$imageUpload = Upload-FileStreamLikeAkashic -Uri $uri -Token $AccessToken -SourcePath $imagePath
$imageSend = $null
if ($imageUpload.data.file_path) {
    $imageParams = @{
        message = @(
            @{ type = "text"; data = @{ text = "native onebot image smoke $(Get-Date -Format s)" } },
            @{ type = "image"; data = @{ file = $imageUpload.data.file_path } }
        )
    }
    $imageAction = "send_group_msg"
    if ($ConversationType -eq "private") {
        $imageAction = "send_private_msg"
        $imageParams["user_id"] = [long]$targetChatId
    }
    else {
        $imageParams["group_id"] = [long]$targetChatId
    }
    $imageSend = Invoke-OneBotAction -Uri $uri -Token $AccessToken -Action $imageAction -Params $imageParams
}

$fileUpload = Upload-FileStreamLikeAkashic -Uri $uri -Token $AccessToken -SourcePath $filePath
$fileSend = $null
if ($fileUpload.data.file_path) {
    $fileParams = @{
        file = $fileUpload.data.file_path
        name = "native-smoke.txt"
    }
    $fileAction = "upload_group_file"
    if ($ConversationType -eq "private") {
        $fileAction = "upload_private_file"
        $fileParams["user_id"] = [long]$targetChatId
    }
    else {
        $fileParams["group_id"] = [long]$targetChatId
    }
    $fileSend = Invoke-OneBotAction -Uri $uri -Token $AccessToken -Action $fileAction -Params $fileParams
}

[ordered]@{
    websocket_url     = $uri
    conversation_type = $ConversationType
    chat_id           = $targetChatId
    temp_dir          = $tempDir
    image_path        = $imagePath
    file_path         = $filePath
    login             = $login
    image_upload      = $imageUpload
    image_send        = $imageSend
    file_upload       = $fileUpload
    file_send         = $fileSend
} | ConvertTo-Json -Depth 32
