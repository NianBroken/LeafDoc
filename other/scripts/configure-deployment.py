#!/usr/bin/env python3
# 生成两种部署共用的私有配置、证书与网关配置，重复执行保留密钥和密码。
import argparse
import ipaddress
import json
import os
from pathlib import Path
import secrets
import shutil
import subprocess


def main():
    parser = argparse.ArgumentParser(description='Configure LeafDoc Ubuntu deployment')
    parser.add_argument('--mode', choices=['native', 'docker'], required=True)
    parser.add_argument('--directory', required=True)
    parser.add_argument('--hosts', default='localhost,127.0.0.1')
    parser.add_argument('--domain', default='')
    parser.add_argument('--tls-directory')
    args = parser.parse_args()
    root = Path(__file__).resolve().parents[2]
    ports = json.loads((root / 'backend/internal/core/serviceports/ports.json').read_text(encoding='utf-8'))
    directory = Path(args.directory).resolve()
    directory.mkdir(parents=True, exist_ok=True)
    os.chmod(str(directory), 0o755 if args.mode == 'native' else 0o750)
    environment = directory / 'environment'
    values = {}
    if environment.exists():
        for line in environment.read_text(encoding='utf-8').splitlines():
            if line and not line.startswith('#'):
                key, value = line.split('=', 1)
                values[key] = value
    defaults = {
        'LEAFDOC_MYSQL_HOST': '127.0.0.1', 'LEAFDOC_MYSQL_PORT': '3306',
        'LEAFDOC_MYSQL_DATABASE': 'leafdoc', 'LEAFDOC_MYSQL_USER': 'leafdoc',
        'LEAFDOC_MYSQL_PASSWORD': secrets.token_hex(32),
        'LEAFDOC_MYSQL_ROOT_PASSWORD': secrets.token_hex(32),
        'LEAFDOC_JWT_SECRET': secrets.token_hex(64), 'LEAFDOC_ADMIN_PASSWORD': secrets.token_hex(20),
        'LEAFDOC_BASE_PATH': '', 'LEAFDOC_MYSQL_IMAGE': 'mysql:8.0',
        'LEAFDOC_IMAGE': 'leafdoc:local',
        'LEAFDOC_FRONTEND_PORT': str(ports['frontend']), 'LEAFDOC_BACKEND_PORT': str(ports['backend']),
    }
    for key, value in defaults.items():
        values.setdefault(key, value)
    values['LEAFDOC_TLS_MODE'] = 'public' if args.domain else 'local'
    values['LEAFDOC_PUBLIC_DOMAIN'] = args.domain
    if args.domain and not all(c.isalnum() or c in '.-' for c in args.domain):
        raise ValueError('Invalid public domain')
    tls = directory / 'tls'
    tls.mkdir(exist_ok=True)
    if args.tls_directory:
        for name in ['server.crt', 'server.key', 'local-ca.crt']:
            shutil.copyfile(str(Path(args.tls_directory) / name), str(tls / name))
    elif not (tls / 'server.crt').exists():
        hosts = list(dict.fromkeys(['localhost', '127.0.0.1'] + args.hosts.split(',')))
        names = []
        for host in hosts:
            try:
                ipaddress.ip_address(host)
                names.append('IP:' + host)
            except ValueError:
                if not host or not all(c.isalnum() or c in '.-' for c in host):
                    raise ValueError('Invalid certificate hostname')
                names.append('DNS:' + host)
        extension = tls / 'extensions.cnf'
        extension.write_text('subjectAltName=' + ','.join(names) + '\nextendedKeyUsage=serverAuth\n', encoding='utf-8')
        subprocess.run(['openssl', 'req', '-x509', '-newkey', 'rsa:3072', '-nodes', '-sha256', '-days', '3650', '-subj', '/CN=LeafDoc Local CA', '-keyout', str(tls / 'local-ca.key'), '-out', str(tls / 'local-ca.crt')], check=True)
        subprocess.run(['openssl', 'req', '-newkey', 'rsa:2048', '-nodes', '-subj', '/CN=LeafDoc', '-keyout', str(tls / 'server.key'), '-out', str(tls / 'server.csr')], check=True)
        subprocess.run(['openssl', 'x509', '-req', '-sha256', '-days', '825', '-in', str(tls / 'server.csr'), '-CA', str(tls / 'local-ca.crt'), '-CAkey', str(tls / 'local-ca.key'), '-CAcreateserial', '-extfile', str(extension), '-out', str(tls / 'server.crt')], check=True)
    for key in tls.glob('*.key'):
        os.chmod(str(key), 0o640)
    certificate_dir = '/run/leafdoc-tls' if args.mode == 'docker' else str(tls)
    values['LEAFDOC_TLS_CERT'] = certificate_dir + '/server.crt'
    values['LEAFDOC_TLS_KEY'] = certificate_dir + '/server.key'
    values['LEAFDOC_BACKEND_URL'] = 'http://{}:{}'.format('app' if args.mode == 'docker' else '127.0.0.1', ports['backend'])
    values['LEAFDOC_FRONTEND_PORT'] = str(ports['frontend'])
    values['LEAFDOC_BACKEND_PORT'] = str(ports['backend'])
    environment.write_text('# LeafDoc private deployment environment\n' + ''.join('{}={}\n'.format(k, v) for k, v in values.items()), encoding='utf-8')
    os.chmod(str(environment), 0o640)
    print('Configuration ready: ' + str(directory))


if __name__ == '__main__':
    main()
