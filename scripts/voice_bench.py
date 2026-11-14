#!/usr/bin/env python3
"""Summarize browser events or compare the same local recordings/texts. Never uploads audio."""
import argparse
import csv
import json
import math
import os
from pathlib import Path
import statistics
import sys
import time

FIELDS = ["profile","scenario","device","network","sample","stt_errors","pronunciation","naturalness_1_5","end_to_stt_ms","first_text_ms","first_sentence_ms","first_pcm_ms","first_audio_ms","interrupt_silence_ms","cpu_percent","rss_mb","swap_mb","temperature_c","false_interrupts","late_audio","reconnection","quota","notes"]

def percentile(values,p):
    values=sorted(values)
    return values[max(0, math.ceil(p*len(values))-1)] if values else None

def summary(path):
    run=json.loads(Path(path).read_text());turns={};pending_end=None;interrupt=None;latency=[]
    for e in run["events"]:
        kind,data,at=e["type"],e["data"],e["at"]
        if kind=="speech.ended": pending_end=at-data.get("endpoint_delay_ms",0)
        key=data.get("turn_id")
        if key:
            t=turns.setdefault(key,{})
            if pending_end is not None and kind=="stt.final": t["speech_end"]=pending_end;pending_end=None
            names={"stt.final":"stt","text.delta":"text","tts.started":"sentence","tts.first_pcm":"pcm","client.first_audio":"audio"}
            if kind in names:t.setdefault(names[kind],at)
        if kind=="client.interrupt":interrupt=at
    rows=[]
    for key,t in turns.items():
        row={"turn_id":key}
        for stage in ["stt","text","sentence","pcm","audio"]:
            if stage in t and "speech_end" in t:row[stage+"_ms"]=round(t[stage]-t["speech_end"],1)
        if "audio_ms" in row:latency.append(row["audio_ms"])
        rows.append(row)
    return {"profile":run.get("profile"),"samples":rows,"end_to_audio":{"count":len(latency),"median_ms":statistics.median(latency) if latency else None,"p95_ms":percentile(latency,.95)},"limitations":["Client timestamps measure event arrival and estimated AudioContext output, not acoustic output.","Interruption-to-acoustic-silence requires loopback recording or manual device measurement; it is not inferred from a mute command.","Synthetic fixture/tone results do not validate STT/TTS quality."]}

def wer(reference,hypothesis):
    import re
    words=lambda s:re.findall(r"\w+",s.casefold())
    a,b=words(reference),words(hypothesis);row=list(range(len(b)+1))
    for i,x in enumerate(a,1):
        nxt=[i]
        for j,y in enumerate(b,1):nxt.append(min(nxt[-1]+1,row[j]+1,row[j-1]+(x!=y)))
        row=nxt
    return row[-1]/max(1,len(a))

def main():
    p=argparse.ArgumentParser();p.add_argument("export",nargs="?");p.add_argument("--template");p.add_argument("--corpus");p.add_argument("--stt",choices=["base","small"]);p.add_argument("--tts",choices=["kokoro","pocket"]);p.add_argument("--voice");p.add_argument("--output",default="results/components.json");args=p.parse_args()
    if args.template:
        dest=Path(args.template);dest.parent.mkdir(parents=True,exist_ok=True)
        if not dest.exists():
            with dest.open("w") as f:csv.writer(f).writerow(FIELDS)
        print(dest);return
    if args.corpus:
        if bool(args.stt)==bool(args.tts):p.error("Choose one component: --stt OR --tts")
        sys.path.insert(0,str(Path(__file__).resolve().parents[1]/"voice-agent/src"))
        os.environ.setdefault("HF_HOME",str(Path(__file__).resolve().parents[1]/".local/huggingface"))
        os.environ.setdefault("TORCH_HOME",str(Path(__file__).resolve().parents[1]/".local/torch"))
        import numpy as np
        import soundfile as sf
        import threading
        from jarvis_voice_agent.engines import Engines,Profile
        # Avoid loading an unused inference engine during isolated comparisons.
        profile=Profile(stt=args.stt or "fixture",tts=args.tts or "tone",voice=args.voice or ("rafael" if args.tts=="pocket" else "pf_dora"))
        os.environ["JARVIS_TEST_AUDIO"]="true"
        begin=time.monotonic();engine=Engines(profile);load_ms=(time.monotonic()-begin)*1000
        rows=[];corpus=Path(args.corpus);output=Path(args.output);output.parent.mkdir(parents=True,exist_ok=True)
        for i,sample in enumerate(json.loads(corpus.read_text())):
            begin=time.monotonic()
            if args.stt:
                audio,rate=sf.read(corpus.parent/sample["audio"],dtype="float32",always_2d=True)
                from scipy.signal import resample_poly
                mono=audio.mean(axis=1);gcd=math.gcd(rate,16000);mono=resample_poly(mono,16000//gcd,rate//gcd)
                result=engine.transcribe(np.clip(mono*32767,-32768,32767).astype("<i2").tobytes())
                row={"id":sample["id"],"transcript":result,"reference":sample["text"],"wer":wer(sample["text"],result),"elapsed_ms":(time.monotonic()-begin)*1000}
            else:
                chunks=[];first=None;rate=24000
                for audio,rate in engine.synthesize(sample["text"],threading.Event()):
                    if first is None:first=(time.monotonic()-begin)*1000
                    chunks.append(audio)
                if not chunks:raise RuntimeError("TTS returned no audio")
                audio=np.concatenate(chunks);path=output.parent/f"{args.tts}-{profile.voice}-{i}.wav";sf.write(path,audio,rate)
                row={"id":sample["id"],"first_pcm_ms":first,"total_ms":(time.monotonic()-begin)*1000,"audio_seconds":len(audio)/rate,"file":str(path),"manual_quality":"pending"}
            rows.append(row)
        output.write_text(json.dumps({"profile":profile.__dict__,"load_ms":load_ms,"samples":rows},ensure_ascii=False,indent=2));print(output);return
    if args.export:print(json.dumps(summary(args.export),ensure_ascii=False,indent=2));return
    p.print_help()

if __name__=="__main__":main()
