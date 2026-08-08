$ErrorActionPreference = 'Stop'

$root = Join-Path $PSScriptRoot '../..'
$pathDirectory = Join-Path $root 'contracts/openapi/paths'
$source = Join-Path $root 'docs/specs/technical_specifications/02_api_openapi_specifications_v1.0.md'

$sourceCount = @(Get-Content $source | Where-Object { $_ -match '^\| FR-' }).Count
$pathFiles = @(Get-ChildItem $pathDirectory -Filter '*.yaml' | Where-Object { $_.Name -ne 'catalog.yaml' })
$operationLines = @(Select-String -Path $pathFiles.FullName -Pattern '^  (get|post|put|patch|delete):')
$operationIds = @(Select-String -Path $pathFiles.FullName -Pattern '^    operationId:' | ForEach-Object { $_.Line.Trim() })
$owners = @(Select-String -Path $pathFiles.FullName -Pattern '^    x-owner-capability:' | ForEach-Object { $_.Line.Split(':', 2)[1].Trim() })
$stateChanging = @(Select-String -Path $pathFiles.FullName -Pattern '^  (post|put|patch|delete):')
$requiredIdempotency = @(Select-String -Path $pathFiles.FullName -Pattern 'Idempotency-Key-Required')

if ($sourceCount -ne $operationLines.Count) {
    throw "Contract operation count $($operationLines.Count) does not match authoritative catalog count $sourceCount."
}
if (($operationIds | Sort-Object -Unique).Count -ne $operationIds.Count) {
    throw 'Duplicate operationId values found.'
}
if ($owners.Count -ne $operationIds.Count) {
    throw 'Every operation must declare x-owner-capability.'
}
if ($requiredIdempotency.Count -ne $stateChanging.Count) {
    throw "Every state-changing operation must reference the required idempotency header ($($stateChanging.Count) operations, $($requiredIdempotency.Count) references)."
}

foreach ($file in $pathFiles) {
    $content = Get-Content $file.FullName
    for ($index = 0; $index -lt $content.Count; $index++) {
        if ($content[$index] -match '^  get:') {
            $end = $content.Count
            for ($next = $index + 1; $next -lt $content.Count; $next++) {
                if ($content[$next] -match '^/|^  (get|post|put|patch|delete):') { $end = $next; break }
            }
            if ($content[$index..($end - 1)] -match "^      '201':") {
                throw "GET operation in $($file.Name) advertises HTTP 201."
            }
        }
    }
}
if (@(Select-String -Path $pathFiles.FullName -Pattern "^      '201':").Count -gt 0) {
    throw 'User Story 1 contract must not advertise unsupported blanket HTTP 201 responses.'
}

$allowed = @(
    'master-data', 'general-ledger', 'accounts-payable', 'accounts-receivable',
    'payroll', 'invoicing', 'payments', 'reporting', 'intercompany',
    'revenue-recognition', 'fixed-assets', 'multi-currency', 'fiscal-periods',
    'coa-segments', 'bank-reconciliation', 'tax', 'approvals', 'identity-access',
    'audit-integrity'
)
$invalid = @($owners | Where-Object { $_ -notin $allowed })
if ($invalid.Count -gt 0) {
    throw "Unknown x-owner-capability value(s): $($invalid -join ', ')"
}

$requiredFiles = @(
    'contracts/openapi/openapi.yaml',
    'contracts/openapi/components/common.yaml',
    'contracts/openapi/paths/catalog.yaml',
    'contracts/openapi/examples/command-request.yaml',
    'contracts/openapi/examples/established-result.yaml',
    'contracts/openapi/examples/problem-details.yaml'
)
foreach ($file in $requiredFiles) {
    if (-not (Test-Path (Join-Path $root $file))) { throw "Missing committed contract file: $file" }
}
if ($pathFiles.Count -ne 19) { throw "Expected 19 capability path files, found $($pathFiles.Count)." }

$catalogLines = Get-Content (Join-Path $pathDirectory 'catalog.yaml')
$catalogPath = $null
foreach ($line in $catalogLines) {
    if ($line -match '^(/[^:]+):$') {
        $catalogPath = $matches[1]
        continue
    }
    if ($line -match '^\s+\$ref: \./([^#]+)#/(.+)$') {
        $targetFile = Join-Path $pathDirectory $matches[1]
        $pointer = $matches[2].Replace('~1', '/').Replace('~0', '~')
        if (-not (Test-Path $targetFile)) { throw "Catalog reference target does not exist: $targetFile" }
        $targetPattern = '^' + [regex]::Escape($pointer) + ':'
        if (-not (Select-String -Path $targetFile -Pattern $targetPattern)) {
            throw "Catalog reference target does not resolve to $pointer in $targetFile"
        }
    }
}

if ((Get-Content (Join-Path $root 'contracts/openapi/examples/command-request.yaml') -Raw) -match '(password|secret|token|accountNumber)') {
    throw 'Command example contains a credential or sensitive field.'
}

Write-Output "OpenAPI User Story 1 structure verified: $($operationIds.Count) operations, $($owners | Sort-Object -Unique | Measure-Object | Select-Object -ExpandProperty Count) capability owners."
