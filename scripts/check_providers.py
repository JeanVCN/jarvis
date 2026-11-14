#!/usr/bin/env python3
"""Run real inference through the Core; a catalog is never reported as a passed test."""
import argparse
import json
import os
import time
import urllib.request
import uuid


def main():
    p=argparse.ArgumentParser();p.add_argument("--provider",required=True,choices=["scripted","groq","codex","ollama"]);p.add_argument("--model",default="");p.add_argument("--core",default="http://127.0.0.1:18080");args=p.parse_args()
    def call(path,data=None):
        request=urllib.request.Request(args.core+path,data=json.dumps(data).encode() if data is not None else None,headers={"Content-Type":"application/json"})
        with urllib.request.urlopen(request,timeout=200) as r:return json.load(r)
    conversation=call('/api/v2/conversations',{})['conversation_id']
    for message in ["Diga uma frase curta em português brasileiro.","Use a ferramenta calculator para calcular 25 * 4."]:
        turn=uuid.uuid4().hex;start=time.monotonic();first=None;tools=[];text='';status=None
        call(f'/api/v2/conversations/{conversation}/turns',{"turn_id":turn,"message":message,"provider":args.provider,"model":args.model})
        with urllib.request.urlopen(args.core+f'/api/v2/turns/{turn}/events',timeout=200) as response:
            for line in response:
                if not line.startswith(b'data: '):continue
                event=json.loads(line[6:]);kind,data=event['type'],event['data']
                if kind=='text.delta':
                    if first is None:first=(time.monotonic()-start)*1000
                    text+=data['delta']
                elif kind=='tool.result':tools.append(data)
                elif kind=='error':print(json.dumps({"error":data}));raise SystemExit(1)
                elif kind=='turn.finished':status=data['status'];break
        if status!='completed' or not text.strip():raise SystemExit('Inference did not complete')
        if 'calculator' in message and not any(t.get('result')=='100' and not t.get('failed') for t in tools):raise SystemExit('Tool calling did not pass')
        print(json.dumps({"provider":args.provider,"model":args.model,"status":status,"first_text_ms":first,"text":text,"tools":tools},ensure_ascii=False))
    print('Scripted checks passed (not real inference).' if args.provider=='scripted' else 'Real inference and tool checks passed for the selected account/model; limits remain provider-owned.')

if __name__=='__main__':main()
