#!/usr/bin/env python3
# 使用真实 HTTPS、前端资源和管理员登录验证部署，不以进程存在作为成功依据。
import argparse
import http.cookiejar
import json
from pathlib import Path
import re
import ssl
import time
import urllib.error
import urllib.request
import urllib.parse


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument('--url', required=True)
    parser.add_argument('--ca', required=True)
    parser.add_argument('--environment', required=True)
    args = parser.parse_args()
    values = dict(line.split('=', 1) for line in Path(args.environment).read_text(encoding='utf-8').splitlines() if line and not line.startswith('#'))
    opener = urllib.request.build_opener(urllib.request.ProxyHandler({}), urllib.request.HTTPSHandler(context=ssl.create_default_context(cafile=args.ca)), urllib.request.HTTPCookieProcessor(http.cookiejar.CookieJar()))
    url = args.url.rstrip('/') + values.get('LEAFDOC_BASE_PATH', '')
    last = None
    for attempt in range(120):
        try:
            with opener.open(url + '/api/health', timeout=5) as response:
                assert response.status == 200
            break
        except (OSError, urllib.error.URLError) as error:
            last = error
            time.sleep(1)
    else:
        raise RuntimeError('HTTPS readiness failed: ' + str(last))
    with opener.open(url + '/', timeout=10) as response:
        html = response.read().decode('utf-8')
    assets = re.findall(r'(?:src|href)="([^"]+\.(?:js|css))"', html)
    assert assets, 'Frontend assets are missing'
    for asset in assets:
        with opener.open(urllib.parse.urljoin(url + '/', asset), timeout=15) as response:
            assert response.status == 200
            assert response.read(), 'Frontend asset is empty'
    with opener.open(url + '/api/config', timeout=10) as response:
        json.load(response)
    data = json.dumps({'identifier': values.get('LEAFDOC_ADMIN_USERNAME', 'Nian'), 'password': values['LEAFDOC_ADMIN_PASSWORD']}).encode('utf-8')
    request = urllib.request.Request(url + '/api/auth/login', data=data, headers={'Content-Type': 'application/json'})
    with opener.open(request, timeout=15) as response:
        assert response.status == 200
    print('HTTPS, frontend assets, configuration and administrator login verified: ' + url)


if __name__ == '__main__':
    main()
