# 在系统公共目录维护本机 CA 与服务证书，私钥不进入源码或发行包。
[CmdletBinding()]
param([string]$Directory = (Join-Path ([Environment]::GetFolderPath('CommonApplicationData')) 'LeafDoc/tls'))
$ErrorActionPreference = 'Stop'
$certPath = Join-Path $Directory 'server.crt'
$keyPath = Join-Path $Directory 'server.key'
$caPath = Join-Path $Directory 'local-ca.crt'
if (!(Test-Path -LiteralPath $certPath) -or !(Test-Path -LiteralPath $keyPath)) {
    New-Item -ItemType Directory -Path $Directory -Force | Out-Null
    $caKey = [Security.Cryptography.RSA]::Create(3072)
    $serverKey = [Security.Cryptography.RSA]::Create(2048)
    try {
        $hash = [Security.Cryptography.HashAlgorithmName]::SHA256
        $padding = [Security.Cryptography.RSASignaturePadding]::Pkcs1
        $caRequest = [Security.Cryptography.X509Certificates.CertificateRequest]::new('CN=LeafDoc Local CA', $caKey, $hash, $padding)
        $caRequest.CertificateExtensions.Add([Security.Cryptography.X509Certificates.X509BasicConstraintsExtension]::new($true,$false,0,$true))
        $caRequest.CertificateExtensions.Add([Security.Cryptography.X509Certificates.X509KeyUsageExtension]::new([Security.Cryptography.X509Certificates.X509KeyUsageFlags]::KeyCertSign,$true))
        $ca = $caRequest.CreateSelfSigned([DateTimeOffset]::UtcNow.AddMinutes(-5), [DateTimeOffset]::UtcNow.AddYears(10))
        try {
            $request = [Security.Cryptography.X509Certificates.CertificateRequest]::new('CN=LeafDoc', $serverKey, $hash, $padding)
            $san = [Security.Cryptography.X509Certificates.SubjectAlternativeNameBuilder]::new()
            @('localhost', [Net.Dns]::GetHostName()) | Select-Object -Unique | ForEach-Object { $san.AddDnsName($_) }
            $addresses = @([Net.IPAddress]::Loopback, [Net.IPAddress]::IPv6Loopback) + @([Net.Dns]::GetHostAddresses([Net.Dns]::GetHostName()))
            $addresses | Sort-Object IPAddressToString -Unique | ForEach-Object { $san.AddIpAddress($_) }
            $request.CertificateExtensions.Add($san.Build())
            $eku = [Security.Cryptography.OidCollection]::new()
            [void]$eku.Add([Security.Cryptography.Oid]::new('1.3.6.1.5.5.7.3.1'))
            $request.CertificateExtensions.Add([Security.Cryptography.X509Certificates.X509EnhancedKeyUsageExtension]::new($eku,$false))
            $certificate = $request.Create($ca,[DateTimeOffset]::UtcNow.AddMinutes(-5),[DateTimeOffset]::UtcNow.AddYears(2),[Security.Cryptography.RandomNumberGenerator]::GetBytes(16))
            try {
                [IO.File]::WriteAllText($certPath,$certificate.ExportCertificatePem(),[Text.UTF8Encoding]::new($false))
                [IO.File]::WriteAllText($keyPath,$serverKey.ExportPkcs8PrivateKeyPem(),[Text.UTF8Encoding]::new($false))
                [IO.File]::WriteAllText($caPath,$ca.ExportCertificatePem(),[Text.UTF8Encoding]::new($false))
            } finally { $certificate.Dispose() }
        } finally { $ca.Dispose() }
    } finally { $caKey.Dispose(); $serverKey.Dispose() }
}
# 已受信任的证书直接复用。需要安装新信任时写入系统根存储。
if (Test-Path -LiteralPath $caPath) {
    $certificate = [Security.Cryptography.X509Certificates.X509Certificate2]::CreateFromPem([IO.File]::ReadAllText($caPath,[Text.Encoding]::UTF8))
    $chain = [Security.Cryptography.X509Certificates.X509Chain]::new()
    try {
        $chain.ChainPolicy.RevocationMode = 'NoCheck'
        if ($chain.Build($certificate)) { $certificate.Dispose(); Write-Output $Directory; return }
    } finally { $chain.Dispose() }
    $store = [Security.Cryptography.X509Certificates.X509Store]::new('Root','LocalMachine')
    try { $store.Open('ReadWrite'); $store.Add($certificate) } finally { $store.Dispose(); $certificate.Dispose() }
}
Write-Output $Directory
