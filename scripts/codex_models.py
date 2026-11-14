#!/usr/bin/env python3
"""List the isolated account's catalog. This does not establish inference eligibility."""
import asyncio,json,os
from pathlib import Path

async def main():
    home=os.getenv('JARVIS_CODEX_HOME','')
    if not home or not Path(home).is_absolute():raise SystemExit('Set an isolated absolute JARVIS_CODEX_HOME first')
    env={k:v for k,v in os.environ.items() if k in ['PATH','HOME','USER','LANG','HTTP_PROXY','HTTPS_PROXY','NO_PROXY','http_proxy','https_proxy','no_proxy','SSL_CERT_FILE','SSL_CERT_DIR','NODE_EXTRA_CA_CERTS','CODEX_PROXY_CERT']}
    env['CODEX_HOME']=home
    proc=await asyncio.create_subprocess_exec(os.getenv('JARVIS_CODEX_BIN','codex'),'app-server',stdin=asyncio.subprocess.PIPE,stdout=asyncio.subprocess.PIPE,stderr=asyncio.subprocess.DEVNULL,env=env,cwd=home)
    async def send(value):proc.stdin.write((json.dumps(value)+'\n').encode());await proc.stdin.drain()
    async def request(n,method,params):
        await send({'id':n,'method':method,'params':params})
        while True:
            line=await asyncio.wait_for(proc.stdout.readline(),30)
            if not line:raise RuntimeError('App Server disconnected')
            value=json.loads(line)
            if value.get('id')!=n:continue
            if value.get('error'):raise RuntimeError('Protocol/account error (details withheld)')
            return value['result']
    try:
        await request(1,'initialize',{'clientInfo':{'name':'jarvis-catalog','version':'0.1.0'}})
        await send({'method':'initialized'})
        account=await request(2,'account/read',{})
        if (account.get('account') or {}).get('type')!='chatgpt':raise RuntimeError('ChatGPT login required')
        catalog=await request(3,'model/list',{})
        print('Catalog only. Test each selected model with check_providers.py before calling it eligible.')
        for model in catalog.get('data',[]):print(json.dumps({k:model.get(k) for k in ['id','model','displayName','isDefault']},ensure_ascii=False))
        if catalog.get('nextCursor'):print('More catalog entries exist; this page is not the complete catalog.')
    finally:
        if proc.returncode is None:proc.terminate()
        await proc.wait()

if __name__=='__main__':asyncio.run(main())
