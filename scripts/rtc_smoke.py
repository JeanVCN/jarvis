import asyncio,json,time,os
import aiohttp,av,numpy as np
from aiortc import RTCPeerConnection,RTCSessionDescription,AudioStreamTrack
from fractions import Fraction

class Silence(AudioStreamTrack):
 async def recv(self):
  await asyncio.sleep(.02)
  frame=av.AudioFrame.from_ndarray(np.zeros((1,960),np.int16),format='s16',layout='mono');frame.sample_rate=48000
  self.pts=getattr(self,'pts',0)+960;frame.pts=self.pts;frame.time_base=Fraction(1,48000);return frame

async def main():
 pc=RTCPeerConnection();pc.addTrack(Silence());dc=pc.createDataChannel('jarvis');events=[];audible=asyncio.Event();opened=asyncio.Event()
 @dc.on('open')
 def open():opened.set()
 @dc.on('message')
 def message(raw):
  try:events.append(json.loads(raw))
  except Exception:pass
 @pc.on('track')
 def track(track):
  async def consume():
   try:
    while True:
     f=await track.recv()
     if np.abs(f.to_ndarray()).max()>20:audible.set()
   except Exception:pass
  asyncio.create_task(consume())
 async with aiohttp.ClientSession() as http:
  await pc.setLocalDescription(await pc.createOffer())
  async with http.post(os.getenv('JARVIS_TEST_VOICE','http://127.0.0.1:18100')+'/offer',json={'sdp':pc.localDescription.sdp,'type':'offer','profile':{'stt':'fixture','tts':'tone','voice':'pf_dora','turn':'silence'},'provider':'scripted'}) as r:
   data=await r.json();assert r.status==200,data
  await pc.setRemoteDescription(RTCSessionDescription(sdp=data['sdp'],type=data['type']))
  await asyncio.wait_for(opened.wait(),10)
  async def ping():
   while True:dc.send('ping');await asyncio.sleep(1)
  task=asyncio.create_task(ping())
  dc.send(json.dumps({'type':'text','text':'Quero uma resposta longa.','client_epoch':1}))
  await asyncio.wait_for(audible.wait(),15)
  dc.send(json.dumps({'type':'interrupt','client_epoch':2}));await asyncio.sleep(.5)
  dc.send(json.dumps({'type':'text','text':'Use uma ferramenta de teste.','client_epoch':2}))
  for _ in range(150):
   await asyncio.sleep(.1)
   if any(e['type']=='text.delta' and '100.' in e['data'].get('delta','') for e in events):break
  assert any(e['type']=='tool.result' and e['data'].get('result')=='100' for e in events),events[-10:]
  assert any(e['type']=='audio.segment' for e in events)
  assert any(e['type']=='tool.result' and e.get('client_epoch')==2 for e in events)
  assert any(e['type']=='turn.interrupted' for e in events)
  print('WebRTC synthetic audio + text streaming + interruption + calculator passed:',len(events),'events')
  task.cancel()
  await pc.close()
  await asyncio.sleep(.2)
  second=RTCPeerConnection();second.addTrack(Silence());second_dc=second.createDataChannel('jarvis')
  second_opened=asyncio.Event();second_events=[]
  @second_dc.on('open')
  def second_open():second_opened.set()
  @second_dc.on('message')
  def second_message(raw):
   try:second_events.append(json.loads(raw))
   except Exception:pass
  await second.setLocalDescription(await second.createOffer())
  async with http.post(os.getenv('JARVIS_TEST_VOICE','http://127.0.0.1:18100')+'/offer',json={'sdp':second.localDescription.sdp,'type':'offer','profile':{'stt':'fixture','tts':'tone','voice':'pf_dora','turn':'silence'},'provider':'scripted','conversation_id':data['conversation_id'],'resume_session':data['session_id'],'client_epoch':2}) as r:
   resumed=await r.json();assert r.status==200,resumed
  assert resumed['conversation_id']==data['conversation_id']
  await second.setRemoteDescription(RTCSessionDescription(sdp=resumed['sdp'],type=resumed['type']))
  await asyncio.wait_for(second_opened.wait(),10)
  await asyncio.sleep(.2)
  assert not any(e['type']=='tool.call' for e in second_events)
  await second.close()
  print('Transport reconnect preserved conversation identity without replaying tools')
asyncio.run(main())
