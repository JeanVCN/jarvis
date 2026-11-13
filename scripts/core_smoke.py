import json,time,uuid,urllib.request,urllib.error,os
base=os.getenv('JARVIS_TEST_CORE','http://127.0.0.1:18080')
def call(path,data=None,headers=None):
 req=urllib.request.Request(base+path,data=json.dumps(data).encode() if data is not None else None,headers=headers or {'Content-Type':'application/json'})
 with urllib.request.urlopen(req,timeout=20) as r:return json.load(r) if r.status!=204 else None
c=call('/api/v2/conversations',{})['conversation_id'];t=uuid.uuid4().hex
payload={'turn_id':t,'message':'Use ferramenta','provider':'scripted','model':''}
call(f'/api/v2/conversations/{c}/turns',payload)
events=[]
with urllib.request.urlopen(base+f'/api/v2/turns/{t}/events') as r:
 for line in r:
  if line.startswith(b'data: '):events.append(json.loads(line[6:]))
assert events[-1]['type']=='turn.finished'
assert sum(e['type']=='tool.result' for e in events)==1
assert call(f'/api/v2/conversations/{c}/turns',payload)['replayed']
assert len(call('/api/v2/conversations/'+c)['turns'])==1
# The model may finish before playback: cancellation must still record interruption.
call(f'/api/v2/turns/{t}/cancel',{})
call(f'/api/v2/turns/{t}/playback',{'segment':0,'ms':0,'finished':True})
assert call('/api/v2/conversations/'+c)['turns'][0]['delivery_status']=='interrupted'
call(f'/api/v2/turns/{t}/playback',{'segment':2,'ms':500})
call(f'/api/v2/turns/{t}/playback',{'segment':1,'ms':2000})
assert call('/api/v2/conversations/'+c)['turns'][0]['played_segment']==2
with urllib.request.urlopen(base+f'/api/v2/turns/{t}/events?after={events[-2]["seq"]}') as r:
 resumed=[json.loads(line[6:]) for line in r if line.startswith(b'data: ')]
assert len(resumed)==1 and resumed[0]['type']=='turn.finished'
t2=uuid.uuid4().hex;call(f'/api/v2/conversations/{c}/turns',{'turn_id':t2,'message':'Resposta longa','provider':'scripted'})
time.sleep(.15);call(f'/api/v2/turns/{t2}/cancel',{});time.sleep(.3)
assert call('/api/v2/conversations/'+c)['turns'][0]['status']=='interrupted'
try:call('/api/v2/conversations',{}, {'Origin':'https://evil.invalid','Content-Type':'application/json'});raise AssertionError('origin allowed')
except urllib.error.HTTPError as e:assert e.code==403
print('Core integration passed: migrations, persisted history, tool idempotence, event replay, playback monotonicity, cancellation, origin validation')
