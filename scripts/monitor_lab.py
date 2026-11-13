#!/usr/bin/env python3
"""Sample owned service trees plus machine memory; never dumps arguments or env."""
import argparse,csv,json,os,subprocess,time
from pathlib import Path
ROOT=Path(__file__).resolve().parents[1]
def sample(roots):
    parents={int(a):int(b) for a,b in (line.split() for line in subprocess.check_output(['ps','-e','-o','pid=,ppid='],text=True).splitlines())}
    selected=set(roots)
    for _ in range(10):selected.update(pid for pid,parent in parents.items() if parent in selected)
    ticks=0;rss=0;swap=0
    for pid in selected:
        try:
            stat=Path(f'/proc/{pid}/stat').read_text().rsplit(')',1)[1].split();ticks+=int(stat[11])+int(stat[12])
            values={line.split(':',1)[0]:line.split(':',1)[1].strip().split()[0] for line in Path(f'/proc/{pid}/status').read_text().splitlines() if ':' in line and line.split(':',1)[1].strip()}
            rss+=int(values.get('VmRSS',0));swap+=int(values.get('VmSwap',0))
        except (OSError,ValueError,IndexError):pass
    temps=[]
    for path in Path('/sys/class/thermal').glob('thermal_zone*/temp'):
        try:temps.append(int(path.read_text())/1000)
        except (OSError,ValueError):pass
    mem={line.split(':')[0]:int(line.split()[1]) for line in Path('/proc/meminfo').read_text().splitlines()}
    return ticks,{'stack_rss_mb':rss/1024,'stack_swap_mb':swap/1024,'temperature_c':max(temps) if temps else '', 'machine_available_mb':mem['MemAvailable']/1024,'machine_swap_used_mb':(mem['SwapTotal']-mem['SwapFree'])/1024}
def main():
    p=argparse.ArgumentParser();p.add_argument('--seconds',type=int,default=120);p.add_argument('--output',default='results/resources.csv');args=p.parse_args()
    roots=json.loads((ROOT/'.local/pids.json').read_text());out=Path(args.output);out.parent.mkdir(parents=True,exist_ok=True)
    previous=None;last=time.monotonic();clock=os.sysconf('SC_CLK_TCK')
    with out.open('w') as f:
        writer=csv.DictWriter(f,fieldnames=['seconds','stack_cpu_percent','stack_rss_mb','stack_swap_mb','temperature_c','machine_available_mb','machine_swap_used_mb']);writer.writeheader()
        for i in range(args.seconds):
            ticks,data=sample(roots);now=time.monotonic();data.update(seconds=i,stack_cpu_percent=0 if previous is None else 100*max(0,ticks-previous)/clock/max(.001,now-last));writer.writerow(data);f.flush();previous,last=ticks,now;time.sleep(1)
    print(out)
if __name__=='__main__':main()
