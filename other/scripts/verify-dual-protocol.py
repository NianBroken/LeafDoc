#!/usr/bin/env python3
"""使用真实证书信任和独立测试文档验收四个固定协议入口，不输出私密配置。"""
import argparse
import http.cookiejar
import json
import re
import ssl
import sys
import time
import urllib.error
import urllib.parse
import urllib.request
import uuid
from pathlib import Path

TIMEOUT = 60
SEARCH_ATTEMPTS = 20


class NoRedirect(urllib.request.HTTPRedirectHandler):
    """直接保留重定向响应，以便识别协议强制升级。"""
    def redirect_request(self, req, fp, code, msg, headers, newurl):
        return None


class Client:
    """每个验收客户端保存独立 Cookie，切换协议时按当前协议选择 CSRF。"""
    def __init__(self, context):
        self.cookies = http.cookiejar.CookieJar()
        self.opener = urllib.request.build_opener(
            urllib.request.ProxyHandler({}), NoRedirect(),
            urllib.request.HTTPSHandler(context=context),
            urllib.request.HTTPCookieProcessor(self.cookies))
        self.base = ''

    def request(self, path, data=None, method=None, csrf=True, content_type=None, expected=None):
        url = self.base + path
        headers = {}
        if data is not None:
            if not isinstance(data, bytes):
                data = json.dumps(data, ensure_ascii=False).encode('utf-8')
                content_type = 'application/json'
            headers['Content-Type'] = content_type
        name = '__Host-leafdoc_csrf' if self.base.startswith('https:') else 'leafdoc_csrf'
        if csrf:
            for cookie in self.cookies:
                if cookie.name == name:
                    headers['X-CSRF-Token'] = urllib.parse.unquote(cookie.value)
        request = urllib.request.Request(url, data=data, headers=headers, method=method)
        try:
            response = self.opener.open(request, timeout=TIMEOUT)
        except urllib.error.HTTPError as error:
            response = error
        with response:
            body = response.read()
            status = response.status
            if expected is not None:
                assert status == expected, f'{method or "GET"} {path}: {status}, expected {expected}'
            else:
                assert 200 <= status < 300, f'{method or "GET"} {path}: {status}'
            if 'application/json' in response.headers.get('Content-Type', '') and body:
                return json.loads(body)
            return body


def environment(path):
    """仅解析赋值文本，不执行 shell 内容。"""
    result = {}
    for line in Path(path).read_text(encoding='utf-8-sig').splitlines():
        line = line.strip()
        if not line or line.startswith('#'):
            continue
        key, value = line.split('=', 1)
        result[key.strip()] = value.strip().strip('"\'')
    return result


def check(client, identifier, password):
    """验证读取、认证、写入、附件、搜索、刷新与注销，并清理测试文档。"""
    client.request('/api/health')
    html = client.request('/')
    assert b'<html' in html.lower(), 'missing HTML document'
    resources = re.findall(rb'(?:src|href)=["\']([^"\']+\.(?:js|css)(?:\?[^"\']*)?)["\']', html)
    assert resources, 'missing static resources'
    resource = resources[0].decode('utf-8')
    base_path = urllib.parse.urlsplit(client.base).path
    if resource.startswith(base_path + '/') and base_path:
        resource = resource[len(base_path):]
    client.request(resource if resource.startswith('/') else '/' + resource)
    client.request('/api/config')
    client.request('/api/auth/login', {'identifier': identifier, 'password': password})
    client.request('/api/auth/refresh-token', {})
    marker = 'dualprotocol' + uuid.uuid4().hex
    title = '双协议验收 ' + marker
    page = client.request('/api/pages', {'title': title, 'slug': marker, 'parentId': None, 'kind': 'page'})
    page_id = page['id']
    route = '/api/pages/' + page_id
    try:
        page = client.request(route)
        payload = {'title': title, 'slug': marker, 'version': page['version'], 'content': '# 中文验收\n\n' + marker}
        client.request(route, payload, method='PUT', csrf=False, expected=403)
        client.request(route, payload, method='PUT')
        saved = client.request(route)
        assert marker in saved['content'], 'saved content mismatch'
        for attempt in range(SEARCH_ATTEMPTS):
            found = client.request('/api/search?q=' + marker)
            if page_id in json.dumps(found):
                break
            time.sleep(0.5)
        else:
            raise AssertionError('search index did not contain saved document')
        boundary = 'leafdoc' + uuid.uuid4().hex
        content = '中文附件验收'.encode('utf-8')
        multipart = ('--' + boundary + '\r\nContent-Disposition: form-data; name="file"; filename="protocol-check.txt"\r\nContent-Type: text/plain\r\n\r\n').encode() + content + ('\r\n--' + boundary + '--\r\n').encode()
        asset = client.request(route + '/assets', multipart, content_type='multipart/form-data; boundary=' + boundary)
        file = asset['file']
        if base_path and file.startswith(base_path + '/'):
            file = file[len(base_path):]
        assert client.request(file) == content, 'downloaded attachment mismatch'
        client.request(route + '/assets')
        client.request(route + '/assets/protocol-check.txt', method='DELETE')
    finally:
        primary_error = sys.exc_info()[1]
        try:
            current = client.request(route)
            client.request(route + '?version=' + urllib.parse.quote(current['version'], safe=''), method='DELETE')
        except Exception as cleanup_error:
            if primary_error is None:
                raise
            primary_error.add_note('测试文档清理失败: ' + str(cleanup_error))
    client.request(route, expected=404)
    client.request('/api/auth/logout', {})
    print(client.base + ': 健康、页面资源、登录、刷新、CSRF、文档读写搜索、附件和注销通过')


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--environment', required=True)
    parser.add_argument('--ca', required=True)
    parser.add_argument('--host', default='localhost')
    args = parser.parse_args()
    env = environment(args.environment)
    identifier = env.get('LEAFDOC_ADMIN_USERNAME') or env.get('LEAFDOC_ADMIN_EMAIL') or 'admin'
    password = env['LEAFDOC_ADMIN_PASSWORD']
    prefix = env.get('LEAFDOC_BASE_PATH', '').rstrip('/')
    ports = json.loads((Path(__file__).resolve().parents[2] / 'backend/internal/core/serviceports/ports.json').read_text(encoding='utf-8'))
    context = ssl.create_default_context(cafile=args.ca)
    client = Client(context)
    # 同一 Cookie 容器按 HTTPS、HTTP 交错访问两个端口，覆盖协议切换。
    for name in ('frontend', 'backend'):
        for scheme in ('https', 'http', 'https', 'http'):
            client.base = f'{scheme}://{args.host}:{ports[name]}{prefix}'
            check(client, identifier, password)


if __name__ == '__main__':
    main()
