"""在一次性临时服务上验证启动流程和共享设置的实际行为。"""
from pathlib import Path
import argparse
import http.cookiejar
import json
import os
import secrets
import socket
import subprocess
import sys
import tempfile
import threading
import time
import urllib.error
import urllib.request

parser = argparse.ArgumentParser(description='Validate global settings against an isolated LeafDoc process.')
parser.add_argument('--binary', required=True, type=Path)
parser.add_argument('--report', type=Path, default=Path('acceptance-results.json'))
arguments = parser.parse_args()
BINARY = arguments.binary.resolve(strict=True)
REPORT = arguments.report.resolve()
checks = []
with tempfile.TemporaryDirectory(prefix='leafdoc-acceptance-') as data:
    with socket.socket() as s:
        s.bind(('127.0.0.1',0)); port=s.getsockname()[1]
    env=os.environ.copy(); env['LEAFDOC_JWT_SECRET']=secrets.token_urlsafe(64)
    output=(Path(data)/'acceptance-server.log').open('wb')
    env['LEAFDOC_ADMIN_USERNAME']='NianBroken'
    env['LEAFDOC_ADMIN_PASSWORD']='Nian123456'
    process=subprocess.Popen([str(BINARY),'--host','127.0.0.1','--port',str(port),'--data-dir',data,'--allow-insecure','--enable-revision'],env=env,stdout=output,stderr=output)
    base=f'http://127.0.0.1:{port}'
    jars=[http.cookiejar.CookieJar() for _ in range(3)]
    clients=[urllib.request.build_opener(urllib.request.HTTPCookieProcessor(j)) for j in jars]
    def request(path,method='GET',body=None,client=0):
        headers={'Content-Type':'application/json','Origin':base}
        for c in jars[client]:
            if c.name.endswith('csrf'): headers['X-CSRF-Token']=c.value
        req=urllib.request.Request(base+path,data=None if body is None else json.dumps(body).encode(),headers=headers,method=method)
        try:
            with clients[client].open(req,timeout=10) as r:return r.status,json.loads(r.read() or 'null')
        except urllib.error.HTTPError as e:
            raw=e.read()
            try:value=json.loads(raw)
            except ValueError:value=raw.decode(errors='replace')
            return e.code,value
    def check(condition,label,detail=None):
        assert condition,f'{label}: {detail}'
        checks.append(label)
    try:
        for attempt in range(100):
            try: status,config=request('/api/config'); break
            except OSError: time.sleep(.1)
        check(config['preferences']=={'language':'zh','dateFormat':'iso','timeFormat':'24h'},'新实例默认偏好',config)
        check(config['publicAccess'] is True,'默认公开访问')
        status,login=request('/api/auth/login','POST',{'identifier':'NianBroken','password':'Nian123456'})
        check(status==200 and login['user']['mustChangePassword'],'固定账户首次登录受限',login)
        for path in ['/api/tree','/api/users','/api/admin/settings/preferences']:
            status,result=request(path); check(status==403,'首次改密限制 '+path,result)
        check(request('/api/tree',client=1)[0]==200,'匿名公开读取独立于受限会话')
        password='Acceptance-'+secrets.token_urlsafe(12)
        status,result=request('/api/users/me/password','PUT',{'oldPassword':'Nian123456','newPassword':password})
        check(status in [200,204],'修改本人密码',result)
        status,login=request('/api/auth/login','POST',{'identifier':'NianBroken','password':password})
        check(status==200 and not login['user'].get('mustChangePassword',False),'改密后登录解除限制',login)
        check(request('/api/auth/login','POST',{'identifier':'NianBroken','password':'Nian123456'},client=2)[0]==401,'初始密码失效')
        received=threading.Event()
        def events():
            try:
                with clients[1].open(base+'/api/settings/events',timeout=10) as response:
                    initial=True
                    for raw in response:
                        if raw.startswith(b'data:'):
                            if initial: initial=False
                            else: received.set(); return
            except OSError: pass
        thread=threading.Thread(target=events,daemon=True); thread.start(); time.sleep(.3)
        status,result=request('/api/admin/settings/preferences','PUT',{'language':'fr','dateFormat':'dmy_dot','timeFormat':'12h'})
        check(status==200,'管理员写入全局偏好',result)
        check(received.wait(5),'匿名客户端实时收到设置通知')
        check(request('/api/config',client=1)[1]['preferences']==result,'所有访问者读取同一偏好')
        status,user=request('/api/users','POST',{'username':'验收用户','email':'acceptance@example.test','password':'UserAcceptance123','role':'viewer'})
        check(status in [200,201],'创建普通用户',user)
        request('/api/config',client=2)
        status,user=request('/api/auth/login','POST',{'identifier':'验收用户','password':'UserAcceptance123'},client=2)
        check(status==200,'普通用户正常登录',user)
        check(request('/api/config',client=2)[1]['preferences']==result,'普通用户与匿名访客读取同一全局偏好')
        check(request('/api/admin/settings/preferences','PUT',{'language':'zh'},client=2)[0]==403,'普通用户不能修改全局偏好')
        check(request('/api/admin/settings/preferences','PUT',{'dateFormat':'locale'})[0]==400,'拒绝跟随语言格式')
        check(request('/api/admin/settings/preferences','PUT',{'language':'zh'},client=1)[0] in [401,403],'匿名不能写全局设置')
        for mode in ['auto','always','never']:
            status,result=request('/api/admin/settings/toc-display','PUT',{'mode':mode})
            check(status==200 and request('/api/config',client=1)[1]['tocMode']==mode,'全局目录模式 '+mode,result)
        status,result=request('/api/admin/settings/public-access','PUT',{'enabled':False})
        check(status==200 and request('/api/tree',client=1)[0]==401,'关闭公开访问立即阻止匿名读取',result)
        check(request('/api/admin/settings/public-access','PUT',{'enabled':True})[0]==200 and request('/api/tree',client=1)[0]==200,'重新开放匿名读取')
        status,page=request('/api/pages','POST',{'title':'功能验收','slug':'acceptance-document','kind':'page'})
        check(status==201,'实际创建文档',page)
        page_id=page['id']
        content='# 功能验收\n\n## 标题一\n\nAcceptanceSearchToken\n\n## 标题二\n\n## 标题三\n'
        status,page=request('/api/pages/'+page_id,'PUT',{'version':page['version'],'title':'功能验收','slug':'acceptance-document','content':content})
        check(status==200 and page['content']==content,'实际保存文档',page)
        check(request('/api/pages/'+page_id,client=1)[1]['content']==content,'匿名实际读取文档')
        check(request('/api/pages/'+page_id,'PUT',{'version':page['version'],'title':'禁止修改','slug':'acceptance-document','content':'blocked'},client=2)[0]==403,'只读用户不能编辑文档')
        boundary='leafdoc-'+secrets.token_hex(10)
        body=(f'--{boundary}\r\nContent-Disposition: form-data; name="file"; filename="acceptance.txt"\r\nContent-Type: text/plain\r\n\r\nattachment acceptance\r\n--{boundary}--\r\n').encode()
        headers={'Content-Type':f'multipart/form-data; boundary={boundary}'}
        for cookie in jars[0]:
            if cookie.name.endswith('csrf'):headers['X-CSRF-Token']=cookie.value
        with clients[0].open(urllib.request.Request(base+'/api/pages/'+page_id+'/assets',data=body,headers=headers,method='POST'),timeout=10) as uploaded:
            check(uploaded.status in [200,201],'实际上传附件')
        check(request('/api/pages/'+page_id+'/assets',client=1)[0]==200,'匿名读取附件列表')
        for attempt in range(100):
            status,result=request('/api/search?q=AcceptanceSearchToken',client=1)
            if 'acceptance-document' in json.dumps(result):break
            time.sleep(.1)
        check(status==200 and 'acceptance-document' in json.dumps(result),'实际全文搜索',result)
        previous_snapshots={item['id'] for item in request('/api/admin/snapshot')[1].get('snapshots',[])}
        status,result=request('/api/admin/snapshot','POST',{})
        check(status==202,'实际创建快照',result)
        for attempt in range(100):
            result=request('/api/admin/snapshot')[1]
            if any(item['id'] not in previous_snapshots for item in result.get('snapshots',[])):break
            time.sleep(.1)
        check(bool(result.get('snapshots')),'快照文件创建完成',result)
        snapshot=next(item['id'] for item in result['snapshots'] if item['id'] not in previous_snapshots)
        request('/api/admin/settings/preferences','PUT',{'language':'de'})
        status,result=request('/api/admin/restore/'+snapshot,'POST',{})
        check(status==202,'实际执行快照恢复',result)
        for attempt in range(200):
            result=request('/api/config',client=1)[1]
            if result.get('preferences',{}).get('language')=='fr':break
            time.sleep(.1)
        check(result.get('preferences',{}).get('language')=='fr','快照恢复全局配置',result)
        status,result=request('/api/auth/login','POST',{'identifier':'NianBroken','password':password})
        check(status==200,'恢复后账户与修改后的密码有效',result)
        for path in ['/api/auth/totp/setup','/api/auth/totp/verify','/api/admin/backup/pull','/api/users/me/settings']:
            check(request(path,'POST')[0] in [404,405],'旧接口不可调用 '+path)
        check(json.loads((Path(data)/'preferences.json').read_text())['language']=='fr','全局配置已持久化')
        process.terminate();process.wait(15)
        process=subprocess.Popen([str(BINARY),'--host','127.0.0.1','--port',str(port),'--data-dir',data,'--allow-insecure','--enable-revision'],env=env,stdout=output,stderr=output)
        for attempt in range(100):
            try: request('/api/config');break
            except OSError:time.sleep(.1)
        status,login=request('/api/auth/login','POST',{'identifier':'NianBroken','password':password})
        check(status==200 and not login['user'].get('mustChangePassword',False),'重启后使用新密码且不重复强制改密',login)
        check(request('/api/config',client=1)[1]['preferences']['language']=='fr','重启后所有访问者仍读取已保存配置')
        REPORT.write_text(json.dumps(checks,ensure_ascii=False,indent=2),encoding='utf-8')
        print(json.dumps({'passed':len(checks),'checks':checks},ensure_ascii=False,indent=2))
    except BaseException:
        output.flush()
        print((Path(data)/'acceptance-server.log').read_text(encoding='utf-8'),file=sys.stderr)
        raise
    finally:
        process.terminate(); process.wait(15); output.close()
