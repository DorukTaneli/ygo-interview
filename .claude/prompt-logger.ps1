# Prompt log writer for Claude Code hooks.
# Invoked by UserPromptSubmit (arg "prompt") and Stop (arg "response") hooks.
# Reads the hook payload as JSON on stdin and appends to prompt-log.md at the
# project root (the parent of this script's .claude directory).
# Designed to never block the session: any failure exits 0.

try {
    $raw = [Console]::In.ReadToEnd()
    if ([string]::IsNullOrWhiteSpace($raw)) { exit 0 }
    $data = $raw | ConvertFrom-Json

    # Log lives at the project root = parent of the .claude folder holding this script.
    $projectRoot = Split-Path -Parent $PSScriptRoot
    $logFile = Join-Path $projectRoot 'prompt-log.md'
    $ts = Get-Date -Format 'yyyy-MM-dd HH:mm:ss zzz'
    $enc = New-Object System.Text.UTF8Encoding($false)

    # Short session id so concurrent or resumed sessions stay distinguishable in one log.
    $sid = 'unknown'
    if ($data.session_id) {
        $s = [string]$data.session_id
        $sid = if ($s.Length -ge 8) { $s.Substring(0, 8) } else { $s }
    }

    # Decide mode from the hook event name, falling back to the CLI arg.
    $event = $data.hook_event_name
    $arg = if ($args.Count -gt 0) { $args[0] } else { '' }
    $isPrompt = ($event -eq 'UserPromptSubmit') -or ($arg -eq 'prompt')
    $isResponse = ($event -eq 'Stop') -or ($arg -eq 'response')

    if ($isPrompt) {
        $prompt = [string]$data.prompt
        $entry = "`n# PROMPT - $ts - session $sid`n`n$prompt`n"
        [System.IO.File]::AppendAllText($logFile, $entry, $enc)
    }
    elseif ($isResponse) {
        $transcript = [string]$data.transcript_path
        if ($transcript -and (Test-Path $transcript)) {
            $lines = @(Get-Content -Path $transcript -Encoding UTF8)
            $responseText = $null
            for ($i = $lines.Count - 1; $i -ge 0; $i--) {
                $line = $lines[$i]
                if ([string]::IsNullOrWhiteSpace($line)) { continue }
                try { $obj = $line | ConvertFrom-Json } catch { continue }
                if ($obj.type -ne 'assistant' -or -not $obj.message) { continue }
                $content = $obj.message.content
                if ($content -is [string]) {
                    if (-not [string]::IsNullOrWhiteSpace($content)) {
                        $responseText = $content
                        break
                    }
                    continue
                }
                $texts = @()
                foreach ($block in $content) {
                    if ($block.type -eq 'text' -and $block.text) { $texts += [string]$block.text }
                }
                if ($texts.Count -gt 0) {
                    $responseText = ($texts -join "`n")
                    break
                }
            }
            if ($responseText) {
                $entry = "`n# RESPONSE - $ts - session $sid`n`n$responseText`n"
                [System.IO.File]::AppendAllText($logFile, $entry, $enc)
            }
        }
    }
    exit 0
}
catch {
    exit 0
}
